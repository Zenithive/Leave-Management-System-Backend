package service

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

type LeaveTypeService struct {
	repo  *repositories.Repository
	lbSvc *LeaveBalanceService
}

func NewLeaveTypeService(repo *repositories.Repository, lbSvc *LeaveBalanceService) *LeaveTypeService {
	return &LeaveTypeService{repo: repo, lbSvc: lbSvc}
}

// ─────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────

// Create inserts a new leave type, applies defaults, bulk-allocates leave balances
// for all active employees (skipped for is_early types), and logs the action.
func (s *LeaveTypeService) Create(tx *sqlx.Tx, input *models.LeaveTypeInput, callerID uuid.UUID) (models.LeaveType, error) {
	applyLeaveTypeDefaults(input)

	if *input.LeaveCount <= 0 {
		return models.LeaveType{}, fmt.Errorf("leave_count must be greater than 0")
	}

	leave, err := s.repo.AddLeaveType(tx, *input)
	if err != nil {
		return models.LeaveType{}, fmt.Errorf("failed to insert leave type: %w", err)
	}
	// Populate response fields from input
	leave.Name = input.Name
	leave.IsPaid = *input.IsPaid
	leave.DefaultEntitlement = *input.DefaultEntitlement
	leave.InternEntitlement = input.InternEntitlement
	leave.IsEarly = input.IsEarly

	// Bulk-allocate leave balances for all active employees (skip is_early types)
	isEarly := input.IsEarly != nil && *input.IsEarly
	if !isEarly {
		if err := s.lbSvc.BulkAllocateForNewLeaveType(tx, leave.ID, *input.DefaultEntitlement, input.InternEntitlement); err != nil {
			return models.LeaveType{}, err
		}
	}

	if err := s.repo.AddLog(models.NewCommon(constant.ComponentLeaveType, constant.ActionCreate, callerID), tx); err != nil {
		return models.LeaveType{}, fmt.Errorf("failed to log action: %w", err)
	}

	return leave, nil
}

// ─────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────

// Update modifies an existing leave type and recalculates leave balances for all
// employees when entitlement values change.
func (s *LeaveTypeService) Update(tx *sqlx.Tx, leaveTypeID int, input *models.LeaveTypeInput, callerID uuid.UUID) error {
	applyLeaveTypeDefaults(input)

	if *input.DefaultEntitlement < 0 {
		return fmt.Errorf("default entitlement cannot be negative")
	}

	// Fetch existing to get old entitlement + is_early flag
	oldLeaveType, err := s.repo.GetLeaveTypeByIdTx(tx, leaveTypeID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("leave type not found")
	}
	if err != nil {
		return fmt.Errorf("failed to fetch leave type: %w", err)
	}

	if err := s.repo.UpdateLeaveType(tx, leaveTypeID, *input); err != nil {
		return fmt.Errorf("failed to update leave type: %w", err)
	}

	// Resolve effective intern entitlement for recalculation:
	// - If intern_entitlement provided → use it
	// - If not provided (nil) → fall back to new default_entitlement
	newInternEffective := *input.DefaultEntitlement
	if input.InternEntitlement != nil {
		newInternEffective = *input.InternEntitlement
	}

	isEarly := oldLeaveType.IsEarly != nil && *oldLeaveType.IsEarly
	if err := s.lbSvc.RecalculateForEntitlementChange(
		tx,
		leaveTypeID,
		oldLeaveType.DefaultEntitlement,
		*input.DefaultEntitlement,
		newInternEffective,
		isEarly,
	); err != nil {
		return err
	}

	if err := s.repo.AddLog(models.NewCommon(constant.ComponentLeaveType, constant.ActionUpdate, callerID), tx); err != nil {
		return fmt.Errorf("failed to log action: %w", err)
	}

	return nil
}

// ─────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────

// Delete removes a leave type. Returns a conflict error if it is referenced by
// existing leave applications.
func (s *LeaveTypeService) Delete(tx *sqlx.Tx, leaveTypeID int, callerID uuid.UUID) error {
	_, err := s.repo.GetLeaveTypeByIdTx(tx, leaveTypeID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("leave type not found")
	}
	if err != nil {
		return fmt.Errorf("failed to fetch leave type: %w", err)
	}

	err = s.repo.DeleteLeaveType(tx, leaveTypeID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("cannot delete leave type: it is being used in existing leave applications")
	}
	if err != nil {
		return fmt.Errorf("failed to delete leave type: %w", err)
	}

	if err := s.repo.AddLog(models.NewCommon(constant.ComponentLeaveType, constant.ActionDelete, callerID), tx); err != nil {
		return fmt.Errorf("failed to log action: %w", err)
	}

	return nil
}

// ─────────────────────────────────────────────
// GetAll
// ─────────────────────────────────────────────

// GetAll returns all leave types.
func (s *LeaveTypeService) GetAll() ([]models.LeaveType, error) {
	leaveTypes, err := s.repo.GetAllLeaveType()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leave types: %w", err)
	}
	return leaveTypes, nil
}

// ─────────────────────────────────────────────
// Private helpers
// ─────────────────────────────────────────────

// applyLeaveTypeDefaults fills in nil pointer fields with sensible defaults in-place.
// Takes a pointer so no copy is made and mutations are visible to the caller.
func applyLeaveTypeDefaults(input *models.LeaveTypeInput) {
	if input.IsPaid == nil {
		v := false
		input.IsPaid = &v
	}
	if input.DefaultEntitlement == nil {
		v := 0
		input.DefaultEntitlement = &v
	}
	if input.LeaveCount == nil {
		v := 2
		input.LeaveCount = &v
	}
	if input.IsEarly == nil {
		v := false
		input.IsEarly = &v
	}
}
