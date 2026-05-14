package service

// LeaveBalanceService owns all business logic for leave balances:
//
//   - Allocating balances for new employees (with proration + role-based entitlement)
//   - Recalculating balances when joining_date changes
//   - Fetching and calculating the displayed balance for an employee
//   - Adjusting a balance (admin manual correction)

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

// GetBalancesResult is returned by GetBalances.
type GetBalancesResult struct {
	EmployeeID uuid.UUID
	Year       int
	Balances   []models.Balance
}

// GetBalances fetches and calculates the leave balance summary for an employee.
// EMPLOYEE and INTERN can only view their own balances.
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

	// Map model types to service types for CalculateLeaveBalances
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

	return &GetBalancesResult{
		EmployeeID: employeeID,
		Year:       currentYear,
		Balances:   balances,
	}, nil
}

// ─────────────────────────────────────────────
// Adjust
// ─────────────────────────────────────────────

// AdjustResult is returned by Adjust.
type AdjustResult struct {
	NewAdjusted float64
	NewClosing  float64
	Year        int
}

// Adjust applies a manual leave balance correction for an employee.
// Only ADMIN and SUPERADMIN can call this.
// If no balance row exists for the year, one is created using the correct entitlement.
func (s *LeaveBalanceService) Adjust(tx *sqlx.Tx, employeeID uuid.UUID, input models.AdjustLeaveBalanceInput, callerID string) (*AdjustResult, error) {
	currentYear := time.Now().Year()

	balance, err := s.repo.GetLeaveBalanceForAdjustment(tx, employeeID, input.LeaveTypeID, currentYear)
	if err == sql.ErrNoRows {
		// No row yet — resolve correct entitlement and create it
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

	return &AdjustResult{
		NewAdjusted: newAdjusted,
		NewClosing:  newClosing,
		Year:        currentYear,
	}, nil
}

// ─────────────────────────────────────────────
// AllocateForNewEmployee
// ─────────────────────────────────────────────

// AllocateForNewEmployee creates leave balance rows for every non-early leave type
// for a newly created employee. Prorates entitlement when the employee joins mid-year.
func (s *LeaveBalanceService) AllocateForNewEmployee(tx *sqlx.Tx, empID uuid.UUID, role string, joiningDate *time.Time) error {
	leaveTypes, err := s.repo.GetAllLeaveType()
	if err != nil {
		return fmt.Errorf("failed to fetch leave types: %w", err)
	}

	currentYear := time.Now().Year()
	isJoiningThisYear := joiningDate != nil && joiningDate.Year() == currentYear

	for _, lt := range leaveTypes {
		if lt.IsEarly != nil && *lt.IsEarly {
			continue
		}
		entitlement := s.resolveEntitlement(lt.DefaultEntitlement, lt.InternEntitlement, role)
		if isJoiningThisYear {
			entitlement = CalculateProratedLeave(entitlement, int(joiningDate.Month()))
		}
		if err := s.repo.CreateLeaveBalance(tx, empID, lt.ID, entitlement); err != nil {
			return fmt.Errorf("failed to allocate leave balance for type %d: %w", lt.ID, err)
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
	return s.repo.RecalculateLeaveBalancesForJoiningDateChange(tx, empID, newJoiningDate, role, time.Now().Year())
}

// ─────────────────────────────────────────────
// Private helpers
// ─────────────────────────────────────────────

func (s *LeaveBalanceService) resolveEntitlement(defaultEntitlement int, internEntitlement *int, role string) int {
	if role == constant.ROLE_INTERN && internEntitlement != nil {
		return *internEntitlement
	}
	return defaultEntitlement
}
