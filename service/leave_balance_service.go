package service

// LeaveBalanceService is the single owner of all leave balance business logic:
//
//   - Entitlement resolution (intern vs default)
//   - Proration based on joining month
//   - Allocation for new employees
//   - Bulk allocation when a new leave type is created
//   - Recalculation when joining_date changes
//   - Recalculation when role changes (INTERN boundary)
//   - Recalculation when entitlement values change
//   - Fetching and displaying balances
//   - Manual adjustment

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/access_role"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

type LeaveBalanceService struct {
	repo *repositories.Repository
}

func NewLeaveBalanceService(repo *repositories.Repository) *LeaveBalanceService {
	return &LeaveBalanceService{repo: repo}
}

// ─────────────────────────────────────────────
// GetBalances
// ─────────────────────────────────────────────

type GetBalancesResult struct {
	EmployeeID uuid.UUID
	Year       int
	Balances   []models.Balance
}

func (s *LeaveBalanceService) GetBalances(employeeID, callerID uuid.UUID, callerRole string) (*GetBalancesResult, error) {
	if access_role.IsEmployeeLike(callerRole) && callerID != employeeID {
		return nil, fmt.Errorf("employees can only view their own balances")
	}

	currentYear := time.Now().Year()

	leaveTypes, err := s.repo.GetAllLeaveTypesWithEntitlements()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leave types: %w", err)
	}
	balanceRecords, err := s.repo.GetLeaveBalancesByEmployeeAndYear(employeeID, currentYear)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leave balances: %w", err)
	}

	svcLeaveTypes := make([]LeaveTypeData, len(leaveTypes))
	for i, lt := range leaveTypes {
		svcLeaveTypes[i] = LeaveTypeData{
			LeaveTypeID:        lt.LeaveTypeID,
			LeaveTypeName:      lt.LeaveTypeName,
			DefaultEntitlement: lt.DefaultEntitlement,
			InternEntitlement:  lt.InternEntitlement,
		}
	}
	svcBalances := make([]LeaveBalanceData, len(balanceRecords))
	for i, br := range balanceRecords {
		svcBalances[i] = LeaveBalanceData{
			LeaveTypeID: br.LeaveTypeID,
			Opening:     br.Opening,
			Accrued:     br.Accrued,
			Used:        br.Used,
			Adjusted:    br.Adjusted,
			Closing:     br.Closing,
		}
	}

	calculated := CalculateLeaveBalances(svcLeaveTypes, svcBalances)
	balances := make([]models.Balance, len(calculated))
	for i, cb := range calculated {
		balances[i] = models.Balance{
			LeaveTypeID: cb.LeaveTypeID,
			LeaveType:   cb.LeaveType,
			Opening:     cb.Opening,
			Accrued:     cb.Accrued,
			Used:        cb.Used,
			Adjusted:    cb.Adjusted,
			Total:       cb.Total,
			Available:   cb.Available,
		}
	}

	return &GetBalancesResult{EmployeeID: employeeID, Year: currentYear, Balances: balances}, nil
}

// ─────────────────────────────────────────────
// Adjust
// ─────────────────────────────────────────────

type AdjustResult struct {
	NewAdjusted float64
	NewClosing  float64
	Year        int
}

func (s *LeaveBalanceService) Adjust(tx *sqlx.Tx, employeeID uuid.UUID, input models.AdjustLeaveBalanceInput, callerID string) (*AdjustResult, error) {
	currentYear := time.Now().Year()

	balance, err := s.repo.GetLeaveBalanceForAdjustment(tx, employeeID, input.LeaveTypeID, currentYear)
	if err == sql.ErrNoRows {
		empRole, err := s.repo.GetEmployeeRole(employeeID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch employee role: %w", err)
		}
		defaultEntitlement, err := s.repo.GetDefaultEntitlementByLeaveTypeID(tx, input.LeaveTypeID, empRole)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch leave type entitlement: %w", err)
		}
		balance, err = s.repo.CreateLeaveBalanceForAdjustment(tx, employeeID, input.LeaveTypeID, currentYear, defaultEntitlement)
		if err != nil {
			return nil, fmt.Errorf("failed to create leave balance: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch leave balance: %w", err)
	}

	newAdjusted := balance.Adjusted + input.Quantity
	newClosing := balance.Opening + balance.Accrued - balance.Used + newAdjusted

	if err := s.repo.UpdateLeaveBalanceAdjustment(tx, balance.ID, newAdjusted, newClosing); err != nil {
		return nil, fmt.Errorf("failed to update leave balance: %w", err)
	}
	if err := s.repo.InsertLeaveAdjustment(tx, employeeID, input.LeaveTypeID, input.Quantity, input.Reason, callerID, currentYear); err != nil {
		return nil, fmt.Errorf("failed to record adjustment log: %w", err)
	}

	return &AdjustResult{NewAdjusted: newAdjusted, NewClosing: newClosing, Year: currentYear}, nil
}

// ─────────────────────────────────────────────
// AllocateForNewEmployee
// ─────────────────────────────────────────────

// AllocateForNewEmployee creates a leave balance row for every non-early leave type
// for a newly created employee. Prorates when joining mid-year.
func (s *LeaveBalanceService) AllocateForNewEmployee(tx *sqlx.Tx, empID uuid.UUID, role string, joiningDate *time.Time) error {
	leaveTypes, err := s.repo.GetAllLeaveType()
	if err != nil {
		return fmt.Errorf("failed to fetch leave types: %w", err)
	}
	currentYear := time.Now().Year()
	for _, lt := range leaveTypes {
		if lt.IsEarly != nil && *lt.IsEarly {
			continue
		}
		entitlement := s.computeEntitlement(lt.DefaultEntitlement, lt.InternEntitlement, role, joiningDate, currentYear)
		if err := s.repo.CreateLeaveBalance(tx, empID, lt.ID, entitlement); err != nil {
			return fmt.Errorf("failed to allocate leave balance for type %d: %w", lt.ID, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────
// BulkAllocateForNewLeaveType
// ─────────────────────────────────────────────

// BulkAllocateForNewLeaveType allocates a balance row for every active employee
// when a new leave type is created. Entitlement resolution and proration happen here.
func (s *LeaveBalanceService) BulkAllocateForNewLeaveType(tx *sqlx.Tx, leaveTypeID, defaultEntitlement int, internEntitlement *int) error {
	employees, err := s.repo.GetAllActiveEmployeesWithRoles(tx)
	if err != nil {
		return fmt.Errorf("failed to fetch active employees: %w", err)
	}
	currentYear := time.Now().Year()
	for _, emp := range employees {
		entitlement := s.computeEntitlement(defaultEntitlement, internEntitlement, emp.Role, emp.JoiningDate, currentYear)
		if err := s.repo.CreateLeaveBalance(tx, emp.ID, leaveTypeID, entitlement); err != nil {
			return fmt.Errorf("failed to allocate leave balance for employee %s: %w", emp.ID, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────
// RecalculateForJoiningDateChange
// ─────────────────────────────────────────────

// RecalculateForJoiningDateChange recalculates opening/closing for all current-year
// leave balances when an employee's joining_date is updated.
func (s *LeaveBalanceService) RecalculateForJoiningDateChange(tx *sqlx.Tx, empID uuid.UUID, role string, newJoiningDate *time.Time) error {
	leaveTypes, err := s.repo.GetAllLeaveTypes(tx)
	if err != nil {
		return fmt.Errorf("failed to fetch leave types: %w", err)
	}
	currentYear := time.Now().Year()
	for _, lt := range leaveTypes {
		newOpening := s.computeEntitlement(lt.DefaultEntitlement, lt.InternEntitlement, role, newJoiningDate, currentYear)
		if err := s.repo.UpdateLeaveBalance(tx, newOpening, empID, lt.ID, currentYear); err != nil {
			return fmt.Errorf("failed to update leave balance for type %d: %w", lt.ID, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────
// AdjustForRoleChange
// ─────────────────────────────────────────────

// AdjustForRoleChange recalculates leave balances when an employee's role changes.
// Only acts when the change crosses the INTERN boundary (INTERN→other or other→INTERN).
func (s *LeaveBalanceService) AdjustForRoleChange(tx *sqlx.Tx, employeeID uuid.UUID, oldRole, newRole string) error {
	if oldRole != constant.ROLE_INTERN && newRole != constant.ROLE_INTERN {
		return nil // no balance change outside the INTERN boundary
	}

	var joiningDate *time.Time
	_ = tx.Get(&joiningDate, `SELECT joining_date FROM Tbl_Employee WHERE id=$1`, employeeID)

	leaveTypes, err := s.repo.GetAllLeaveTypes(tx)
	if err != nil {
		return fmt.Errorf("failed to fetch leave types: %w", err)
	}
	currentYear := time.Now().Year()
	for _, lt := range leaveTypes {
		newOpening := s.computeEntitlement(lt.DefaultEntitlement, lt.InternEntitlement, newRole, joiningDate, currentYear)
		if err := s.repo.UpdateLeaveBalance(tx, newOpening, employeeID, lt.ID, currentYear); err != nil {
			return fmt.Errorf("failed to update leave balance for type %d: %w", lt.ID, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────
// RecalculateForEntitlementChange
// ─────────────────────────────────────────────

// RecalculateForEntitlementChange recalculates leave balances for all employees
// when a leave type's entitlement values change. Skipped for is_early types.
func (s *LeaveBalanceService) RecalculateForEntitlementChange(tx *sqlx.Tx, leaveTypeID, oldDefault, newDefault, newInternEffective int, isEarly bool) error {
	if isEarly {
		return nil
	}
	currentYear := time.Now().Year()

	// Non-intern employees
	nonInterns, err := s.repo.GetEmployeesByRole(tx, false)
	if err != nil {
		return fmt.Errorf("failed to fetch non-intern employees: %w", err)
	}
	for _, emp := range nonInterns {
		newOpening := s.computeEntitlement(newDefault, nil, emp.Role, emp.JoiningDate, currentYear)
		_, balErr := s.repo.GetLeaveBalance(tx, emp.ID, leaveTypeID)
		if balErr == sql.ErrNoRows {
			if err := s.repo.CreateLeaveBalance(tx, emp.ID, leaveTypeID, newOpening); err != nil {
				return fmt.Errorf("failed to create leave balance for employee %s: %w", emp.ID, err)
			}
		} else {
			if err := s.repo.UpdateLeaveBalance(tx, newOpening, emp.ID, leaveTypeID, currentYear); err != nil {
				return fmt.Errorf("failed to update leave balance for employee %s: %w", emp.ID, err)
			}
		}
	}

	// Intern employees
	interns, err := s.repo.GetEmployeesByRole(tx, true)
	if err != nil {
		return fmt.Errorf("failed to fetch intern employees: %w", err)
	}
	for _, emp := range interns {
		internEntitlement := newInternEffective
		newOpening := s.computeEntitlement(internEntitlement, nil, constant.ROLE_INTERN, emp.JoiningDate, currentYear)
		_, balErr := s.repo.GetLeaveBalance(tx, emp.ID, leaveTypeID)
		if balErr == sql.ErrNoRows {
			if err := s.repo.CreateLeaveBalance(tx, emp.ID, leaveTypeID, newOpening); err != nil {
				return fmt.Errorf("failed to create leave balance for intern %s: %w", emp.ID, err)
			}
		} else {
			if err := s.repo.UpdateLeaveBalance(tx, newOpening, emp.ID, leaveTypeID, currentYear); err != nil {
				return fmt.Errorf("failed to update leave balance for intern %s: %w", emp.ID, err)
			}
		}
	}
	return nil
}

// ─────────────────────────────────────────────
// Private helpers
// ─────────────────────────────────────────────

// resolveEntitlement picks the correct entitlement for a role.
func (s *LeaveBalanceService) resolveEntitlement(defaultEntitlement int, internEntitlement *int, role string) int {
	if role == constant.ROLE_INTERN && internEntitlement != nil {
		return *internEntitlement
	}
	return defaultEntitlement
}

// computeEntitlement is the single source of truth for:
//  1. Role-based entitlement selection (intern vs default)
//  2. Proration when the employee joins mid-year
func (s *LeaveBalanceService) computeEntitlement(defaultEntitlement int, internEntitlement *int, role string, joiningDate *time.Time, currentYear int) int {
	entitlement := s.resolveEntitlement(defaultEntitlement, internEntitlement, role)
	if joiningDate != nil && joiningDate.Year() == currentYear {
		entitlement = CalculateProratedLeave(entitlement, int(joiningDate.Month()))
	}
	return entitlement
}
