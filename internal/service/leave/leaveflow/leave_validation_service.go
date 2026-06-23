package leaveflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/repositories"
)

// LeaveValidationService provides reusable validation functions for leave operations
type LeaveValidationService struct {
	repo *repositories.Repository
}

func NewLeaveValidationService(repo *repositories.Repository) *LeaveValidationService {
	return &LeaveValidationService{repo: repo}
}

// ValidateLeaveTimingID validates timing ID is 1, 2, or 3
func (s *LeaveValidationService) ValidateLeaveTimingID(timingID *int) error {
	if timingID != nil && (*timingID < 1 || *timingID > 3) {
		return fmt.Errorf("invalid leave timing ID. Must be 1 (First Half), 2 (Second Half), or 3 (Full Day)")
	}
	return nil
}

// ValidateLeaveReason validates reason length (10-500 characters)
func (s *LeaveValidationService) ValidateLeaveReason(reason string) error {
	trimmed := strings.TrimSpace(reason)
	if len(trimmed) < 10 {
		return fmt.Errorf("leave reason must be at least 10 characters long")
	}
	if len(trimmed) > 500 {
		return fmt.Errorf("leave reason is too long. Maximum 500 characters allowed")
	}
	return nil
}

// ValidateLeaveDates validates end date is not before start date
func (s *LeaveValidationService) ValidateLeaveDates(startDate, endDate time.Time) error {
	if endDate.Before(startDate) {
		return fmt.Errorf("end date cannot be earlier than start date")
	}
	return nil
}

// LeaveTypeInfo contains leave type details needed for validation
type LeaveTypeInfo struct {
	LeaveType models.LeaveType
	TimingID  int
}

// GetLeaveTypeAndResolveTimingID fetches leave type and resolves timing ID
// For IsEarly leave types, timing ID is forced to 3 (Full Day)
func (s *LeaveValidationService) GetLeaveTypeAndResolveTimingID(leaveTypeID int, requestedTimingID *int) (*LeaveTypeInfo, error) {
	leaveType, err := s.repo.GetLeaveTypeById(leaveTypeID)
	if err != nil {
		return nil, fmt.Errorf("invalid leave type: %w", err)
	}

	timingID := 3 // Default to Full Day
	if requestedTimingID != nil {
		timingID = *requestedTimingID
	}

	// For IsEarly leave types, timing is not applicable - force full day
	if leaveType.IsEarly != nil && *leaveType.IsEarly {
		timingID = 3
	}

	return &LeaveTypeInfo{
		LeaveType: leaveType,
		TimingID:  timingID,
	}, nil
}

// ValidateLeaveBalance checks if employee has sufficient leave balance.
// For non-early leave types only.
//
// When excludeLeaveID is set (edit flow), the original leave's days are added back
// to the effective balance — because editing replaces that leave, not adds to it.
func (s *LeaveValidationService) ValidateLeaveBalance(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, requiredDays float64, excludeLeaveID *uuid.UUID) error {
	balance, err := s.repo.GetLeaveBalance(tx, employeeID, leaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to fetch leave balance: %w", err)
	}

	// Subtract already-pending (not yet deducted) leaves from available balance.
	// GetPendingLeaveDays already excludes the leave being edited (excludeLeaveID),
	// so those days are not double-counted in pendingDays.
	pendingDays, err := s.repo.GetPendingLeaveDays(tx, employeeID, leaveTypeID, excludeLeaveID)
	if err != nil {
		return fmt.Errorf("failed to fetch pending leave days: %w", err)
	}

	// For edit: add back the original leave's days so the balance reflects the
	// days that will be freed when this leave is replaced.
	var originalDays float64
	if excludeLeaveID != nil {
		originalDays, err = s.repo.GetLeaveDaysByID(tx, *excludeLeaveID)
		if err != nil {
			return fmt.Errorf("failed to fetch original leave days: %w", err)
		}
	}
	effectiveBalance := balance - pendingDays + originalDays

	// Check balance
	if effectiveBalance < requiredDays {
		return fmt.Errorf(
			"insufficient leave balance. Available: %.1f days (%.1f closing - %.1f pending + %.1f original), Required: %.1f days",
			effectiveBalance, balance, pendingDays, originalDays, requiredDays,
		)
	}

	return nil
}

// ValidateEarlyLeaveLimit checks if employee already has an early leave for the month
// For IsEarly leave types only.
func (s *LeaveValidationService) ValidateEarlyLeaveLimit(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, refDate time.Time) error {
	existing, err := s.repo.GetEarlyLeaveThisMonth(tx, employeeID, leaveTypeID, refDate)
	if err != nil {
		return fmt.Errorf("failed to check early leave: %w", err)
	}
	if existing != nil {
		return fmt.Errorf(
			"early leave already taken this month: %s to %s (Status: %s). Only one early leave per month is allowed",
			existing.StartDate.Format("2006-01-02"),
			existing.EndDate.Format("2006-01-02"),
			existing.Status,
		)
	}
	return nil
}

// ValidateOverlappingLeaves checks if there are overlapping leave applications
func (s *LeaveValidationService) ValidateOverlappingLeaves(tx *sqlx.Tx, employeeID uuid.UUID, startDate, endDate time.Time, excludeLeaveID *uuid.UUID) error {
	overlaps, err := s.repo.GetOverlappingLeaves(tx, employeeID, startDate, endDate, excludeLeaveID)
	if err != nil {
		return fmt.Errorf("failed to check overlapping leave: %w", err)
	}

	// Filter out excluded leave (for edit operation)
	for _, ov := range overlaps {
		if excludeLeaveID == nil || ov.ID != *excludeLeaveID {
			return fmt.Errorf(
				"overlapping leave exists: %s from %s to %s (Status: %s). Please cancel or modify the existing leave first",
				ov.LeaveType,
				ov.StartDate.Format("2006-01-02"),
				ov.EndDate.Format("2006-01-02"),
				ov.Status,
			)
		}
	}

	return nil
}

// ValidateLeaveApplicationParams contains all parameters needed for comprehensive validation
type ValidateLeaveApplicationParams struct {
	EmployeeID     uuid.UUID
	LeaveTypeID    int
	StartDate      time.Time
	EndDate        time.Time
	LeaveDays      float64
	ExcludeLeaveID *uuid.UUID // For edit operations, exclude this leave from overlap/pending checks
}

// ValidateLeaveApplication performs comprehensive validation for leave application/edit
// Combines balance check, early leave limit, and overlapping leave checks
func (s *LeaveValidationService) ValidateLeaveApplication(tx *sqlx.Tx, params ValidateLeaveApplicationParams, leaveType models.LeaveType) error {
	// Check if it's an early leave type
	isEarly := leaveType.IsEarly != nil && *leaveType.IsEarly

	if isEarly {
		// Early leave: check one-per-month limit
		if err := s.ValidateEarlyLeaveLimit(tx, params.EmployeeID, params.LeaveTypeID, params.StartDate); err != nil {
			return err
		}
	} else {
		// Non-early leave: check balance
		if err := s.ValidateLeaveBalance(tx, params.EmployeeID, params.LeaveTypeID, params.LeaveDays, params.ExcludeLeaveID); err != nil {
			return err
		}
	}

	// Check overlapping leaves (applies to all leave types)
	if err := s.ValidateOverlappingLeaves(tx, params.EmployeeID, params.StartDate, params.EndDate, params.ExcludeLeaveID); err != nil {
		return err
	}

	return nil
}
