package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/access_role"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

type EmployeeService struct {
	repo    *repositories.Repository
	leaveSvc *LeaveBalanceService
}

func NewEmployeeService(repo *repositories.Repository, leaveSvc *LeaveBalanceService) *EmployeeService {
	return &EmployeeService{repo: repo, leaveSvc: leaveSvc}
}

// ─────────────────────────────────────────────
// GetAllEmployees
// ─────────────────────────────────────────────

func (s *EmployeeService) GetAllEmployees(params models.EmployeeFilterParams, role string) (*models.PaginatedEmployeeResponse, error) {
	return s.repo.GetAllEmployees(params, role)
}

// ─────────────────────────────────────────────
// GetMyTeam
// ─────────────────────────────────────────────

func (s *EmployeeService) GetMyTeam(managerID uuid.UUID, params models.TeamFilterParams) ([]models.EmployeeResponse, error) {
	return s.repo.GetEmployeesByManagerID(managerID, params)
}

// ─────────────────────────────────────────────
// GetEmployeeByID
// ─────────────────────────────────────────────

func (s *EmployeeService) GetEmployeeByID(empID uuid.UUID) (*models.EmployeeResponse, error) {
	return s.repo.GetEmployeeByID(empID)
}

// ─────────────────────────────────────────────
// CreateEmployee
// ─────────────────────────────────────────────

type CreateEmployeeResult struct {
	GeneratedPassword string
}

func (s *EmployeeService) CreateEmployee(tx *sqlx.Tx, input models.EmployeeInput, callerRole string) (*CreateEmployeeResult, error) {
	if access_role.IsAdminOrHR(callerRole) && input.Role == constant.ROLE_SUPER_ADMIN {
		return nil, fmt.Errorf("HR and ADMIN cannot create SUPERADMIN users")
	}
	if !strings.HasSuffix(input.Email, "@zenithive.com") {
		return nil, fmt.Errorf("email must end with @zenithive.com")
	}
	exists, err := s.repo.CheckEmailExists(input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already exists")
	}
	roleID, err := s.repo.GetRoleID(input.Role)
	if err != nil {
		return nil, fmt.Errorf("role not found")
	}
	generatedPassword, err := utils.GenerateSecurePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secure password")
	}
	hash, err := utils.HashPassword(generatedPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password")
	}
	if input.Salary == nil {
		zero := 0.0
		input.Salary = &zero
	}
	id, err := s.repo.InsertEmployee(tx, input.FullName, input.Email, roleID, hash, input.Salary, input.JoiningDate)
	if err != nil || id == uuid.Nil {
		return nil, fmt.Errorf("failed to create employee")
	}

	if err := s.leaveSvc.AllocateForNewEmployee(tx, id, input.Role, input.JoiningDate); err != nil {
		return nil, err
	}

	return &CreateEmployeeResult{GeneratedPassword: generatedPassword}, nil
}

// ─────────────────────────────────────────────
// UpdateEmployeeRole
// ─────────────────────────────────────────────

type UpdateRoleResult struct {
	UpdatedID string
	OldRole   string
}

func (s *EmployeeService) UpdateEmployeeRole(tx *sqlx.Tx, empID uuid.UUID, newRole, callerRole string, callerID uuid.UUID) (*UpdateRoleResult, error) {
	if access_role.IsAdminOrHR(callerRole) && callerID == empID {
		return nil, fmt.Errorf("ADMIN and HR cannot change their own role. Only SUPERADMIN can change roles")
	}

	currentRole, isManager, err := s.repo.GetEmployeeCurrentRoleAndManagerStatus(empID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch employee role: %w", err)
	}

	if access_role.IsAdminOrHR(callerRole) && currentRole == constant.ROLE_SUPER_ADMIN {
		return nil, fmt.Errorf("HR and ADMIN cannot modify SUPERADMIN users")
	}
	if access_role.IsAdminOrHR(callerRole) && newRole == constant.ROLE_SUPER_ADMIN {
		return nil, fmt.Errorf("HR and ADMIN cannot promote users to SUPERADMIN")
	}
	if currentRole == newRole {
		return nil, fmt.Errorf("employee already has this role")
	}
	if isManager && newRole != constant.ROLE_MANAGER {
		return nil, fmt.Errorf("cannot change role of employee who is a manager with subordinates")
	}

	updatedID, err := s.repo.UpdateEmployeeRole(tx, empID, newRole)
	if err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}
	if err := s.repo.AdjustLeaveBalancesForRoleChange(tx, empID, currentRole, newRole, time.Now().Year()); err != nil {
		return nil, fmt.Errorf("failed to adjust leave balances for role change: %w", err)
	}
	return &UpdateRoleResult{UpdatedID: updatedID, OldRole: currentRole}, nil
}

// ─────────────────────────────────────────────
// ToggleEmployeeStatus (deactivate / activate)
// ─────────────────────────────────────────────

func (s *EmployeeService) ToggleEmployeeStatus(tx *sqlx.Tx, empID uuid.UUID, callerRole string) (string, error) {
	target, err := s.repo.GetEmployeeByID(empID)
	if err != nil {
		return "", fmt.Errorf("employee not found")
	}
	if callerRole == constant.ROLE_ADMIN && target.Role == constant.ROLE_SUPER_ADMIN {
		return "", fmt.Errorf("HR and ADMIN cannot modify SUPERADMIN users")
	}
	return s.repo.DeleteEmployeeStatus(tx, empID)
}

// ─────────────────────────────────────────────
// UpdateEmployeeManager
// ─────────────────────────────────────────────

func (s *EmployeeService) UpdateEmployeeManager(empID, managerID, callerID uuid.UUID, callerRole string) error {
	target, err := s.repo.GetEmployeeByID(empID)
	if err != nil {
		return fmt.Errorf("employee not found")
	}
	if access_role.IsAdminOrHR(callerRole) && target.Role == constant.ROLE_SUPER_ADMIN {
		return fmt.Errorf("HR and ADMIN cannot modify SUPERADMIN users")
	}
	if empID == managerID {
		return fmt.Errorf("cannot assign employee as their own manager")
	}
	if callerID == managerID && callerRole != constant.ROLE_SUPER_ADMIN {
		return fmt.Errorf("you cannot assign yourself as a manager to others. Only SUPERADMIN can do this")
	}

	// Validate manager exists, is active, and has MANAGER role
	var mgrRole, mgrStatus string
	if err := s.repo.DB.Get(&mgrRole, `
		SELECT r.type FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id WHERE e.id=$1`, managerID); err != nil {
		return fmt.Errorf("manager not found")
	}
	if err := s.repo.DB.Get(&mgrStatus, `SELECT status FROM Tbl_Employee WHERE id=$1`, managerID); err != nil || mgrStatus != "active" {
		return fmt.Errorf("manager is deactivated")
	}
	if mgrRole != constant.ROLE_MANAGER {
		return fmt.Errorf("assigned employee is not a manager")
	}
	return s.repo.UpdateManager(empID, managerID)
}

// ─────────────────────────────────────────────
// UpdateEmployeeInfo
// ─────────────────────────────────────────────

func (s *EmployeeService) UpdateEmployeeInfo(tx *sqlx.Tx, empID, callerID uuid.UUID, callerRole string, input models.UpdateEmployeeInfoInput) error {
	existing, err := s.repo.GetEmployeeByID(empID)
	if err != nil {
		return fmt.Errorf("employee not found")
	}
	if access_role.IsAdminOrHR(callerRole) && existing.Role == constant.ROLE_SUPER_ADMIN {
		return fmt.Errorf("HR and ADMIN cannot modify SUPERADMIN users")
	}

	if input.BirthDate != nil {
		today := time.Now().Truncate(24 * time.Hour)
		if !input.BirthDate.Truncate(24 * time.Hour).Before(today) {
			return fmt.Errorf("birth_date must be a past date")
		}
	}

	isAdmin := access_role.IsAdminLike(callerRole)
	isSelf := callerID == empID

	if (input.Email != nil || input.Salary != nil || input.JoiningDate != nil || input.BirthDate != nil || input.EndingDate != nil) && !isAdmin {
		return fmt.Errorf("only SUPERADMIN and ADMIN can update email, salary, joining date, and ending date")
	}
	if input.FullName != nil && !isSelf && !isAdmin {
		return fmt.Errorf("you can only update your own name")
	}

	// Resolve final email
	finalEmail := existing.Email
	if input.Email != nil {
		if !strings.HasSuffix(*input.Email, "@zenithive.com") {
			return fmt.Errorf("email must end with @zenithive.com")
		}
		if existing.Email != *input.Email {
			exists, err := s.repo.CheckEmailExists(*input.Email)
			if err != nil {
				return fmt.Errorf("failed to check email: %w", err)
			}
			if exists {
				return fmt.Errorf("email already exists")
			}
		}
		finalEmail = *input.Email
	}

	// Resolve final values (keep existing if not provided)
	finalName := existing.FullName
	if input.FullName != nil {
		finalName = *input.FullName
	}
	finalSalary := existing.Salary
	if input.Salary != nil {
		finalSalary = input.Salary
	}
	finalJoining := existing.JoiningDate
	if input.JoiningDate != nil {
		finalJoining = input.JoiningDate
	}
	finalBirth := existing.BirthDate
	if input.BirthDate != nil {
		finalBirth = input.BirthDate
	}
	finalEnding := existing.EndingDate
	if input.EndingDate != nil {
		finalEnding = input.EndingDate
	}

	if tx != nil {
		// joining_date changed — update + recalculate leave balances atomically
		if err := s.repo.UpdateEmployeeInfoTx(tx, empID, finalName, finalEmail, finalSalary, finalJoining, finalBirth, finalEnding); err != nil {
			return fmt.Errorf("failed to update employee: %w", err)
		}
		empRole, err := s.repo.GetEmployeeRole(empID)
		if err != nil {
			return fmt.Errorf("failed to fetch employee role: %w", err)
		}
		if err := s.leaveSvc.RecalculateForJoiningDateChange(tx, empID, empRole, finalJoining); err != nil {
			return fmt.Errorf("failed to recalculate leave balances: %w", err)
		}
		return nil
	}
	return s.repo.UpdateEmployeeInfo(empID, finalName, finalEmail, finalSalary, finalJoining, finalBirth, finalEnding)
}

// ─────────────────────────────────────────────
// UpdateEmployeePassword
// ─────────────────────────────────────────────

type UpdatePasswordResult struct {
	EmployeeEmail    string
	EmployeeFullName string
	UpdaterEmail     string
}

func (s *EmployeeService) UpdateEmployeePassword(empID uuid.UUID, newPassword, callerRole, callerID string) (*UpdatePasswordResult, error) {
	if len(newPassword) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters long")
	}
	existing, err := s.repo.GetEmployeeByID(empID)
	if err != nil {
		return nil, fmt.Errorf("employee not found")
	}
	if access_role.IsAdminOrHR(callerRole) && existing.Role == constant.ROLE_SUPER_ADMIN {
		return nil, fmt.Errorf("HR and ADMIN cannot modify SUPERADMIN users")
	}
	hashed, err := utils.HashPassword(newPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	if err := s.repo.UpdateEmployeePassword(empID, hashed); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	// Fetch notification details
	var updaterEmail string
	_ = s.repo.DB.Get(&updaterEmail, `SELECT email FROM Tbl_Employee WHERE id=$1`, callerID)
	if updaterEmail == "" {
		updaterEmail = "admin@zenithive.com"
	}
	return &UpdatePasswordResult{
		EmployeeEmail:    existing.Email,
		EmployeeFullName: existing.FullName,
		UpdaterEmail:     updaterEmail,
	}, nil
}

// ─────────────────────────────────────────────
// UpdateEmployeeDesignation
// ─────────────────────────────────────────────

func (s *EmployeeService) UpdateEmployeeDesignation(empID uuid.UUID, designationIDStr *string, callerRole string) (*uuid.UUID, error) {
	target, err := s.repo.GetEmployeeByID(empID)
	if err != nil {
		return nil, fmt.Errorf("employee not found")
	}
	if access_role.IsAdminOrHR(callerRole) && target.Role == constant.ROLE_SUPER_ADMIN {
		return nil, fmt.Errorf("HR and ADMIN cannot modify SUPERADMIN users")
	}

	var designationID *uuid.UUID
	if designationIDStr != nil && *designationIDStr != "" {
		parsed, err := uuid.Parse(*designationIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid designation ID")
		}
		if _, err := s.repo.GetDesignationByID(parsed); err != nil {
			return nil, fmt.Errorf("designation not found")
		}
		designationID = &parsed
	}

	if err := s.repo.UpdateEmployeeDesignation(empID, designationID); err != nil {
		return nil, fmt.Errorf("failed to update designation: %w", err)
	}
	return designationID, nil
}

// ─────────────────────────────────────────────
// GetTodayBirthdays
// ─────────────────────────────────────────────

type BirthdayResult struct {
	Entries []models.BirthdayEntry
	Date    string
}

func (s *EmployeeService) GetTodayBirthdays() (*BirthdayResult, error) {
	tmpl, err := s.repo.GetBirthdayMessageTemplate()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}
	employees, err := s.repo.GetTodayBirthdays()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch birthdays: %w", err)
	}
	entries := make([]models.BirthdayEntry, 0, len(employees))
	for _, emp := range employees {
		entries = append(entries, models.BirthdayEntry{
			ID:      emp.ID,
			Name:    emp.Name,
			Email:   emp.Email,
			Message: RenderBirthdayMessage(tmpl, emp.Name, emp.BirthDate),
		})
	}
	return &BirthdayResult{
		Entries: entries,
		Date:    time.Now().Format("2006-01-02"),
	}, nil
}

// ─────────────────────────────────────────────
// GetBirthdays
// ─────────────────────────────────────────────

func (s *EmployeeService) GetBirthdays(month, year int) (interface{}, error) {
	rows, err := s.repo.GetBirthdays(month, year)
	if err != nil {
		return nil, err
	}
	return Calculation(rows, month, year), nil
}
