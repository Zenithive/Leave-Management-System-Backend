package models

import (
	"time"

	"github.com/google/uuid"
)

// Leave - Database model for Tbl_Leave
type Leave struct {
	ID            uuid.UUID  `db:"id"`
	EmployeeID    uuid.UUID  `db:"employee_id"`
	LeaveTypeID   int        `db:"leave_type_id"`
	LeaveTimingID *int       `db:"half_id"`      // References Tbl_Half
	LeaveTiming   *string    `db:"leave_timing"` // For early leave timing
	StartDate     time.Time  `db:"start_date"`
	EndDate       time.Time  `db:"end_date"`
	Days          float64    `db:"days"`
	Status        string     `db:"status"`
	AppliedByID   *uuid.UUID `db:"applied_by"`
	ApprovedByID  *uuid.UUID `db:"approved_by"`
	Reason        string     `db:"reason"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

// LeaveInput - Request payload for creating a new leave application
type LeaveInput struct {
	EmployeeID    uuid.UUID  `json:"employee_id" validate:"required"`
	LeaveTypeID   int        `json:"leave_type_id" validate:"required"`
	LeaveTimingID *int       `json:"leave_timing_id,omitempty"` // 1=First Half, 2=Second Half, 3=Full Day
	LeaveTiming   *string    `json:"leave_timing,omitempty"`    // For early leave: HH:MM format
	StartDate     time.Time  `json:"start_date" validate:"required"`
	EndDate       time.Time  `json:"end_date" validate:"required"`
	Reason        string     `json:"reason" validate:"required,min=10,max=500"`
	Days          *float64   `json:"days,omitempty"`        // Calculated by system
	Status        string     `json:"status,omitempty"`      // Calculated by system
	AppliedByID   *uuid.UUID `json:"applied_by,omitempty"`  // Who applied (manager on behalf)
	ApprovedByID  *uuid.UUID `json:"approved_by,omitempty"` // Who approved
}

type action string

const (
	APPROVE action = "APPROVE"
	REJECT  action = "REJECT"
)

type ActionLeaveReq struct {
	Action  string `json:"action" validate:"required"` // APPROVE / REJECT / WITHDRAW
	Remarks string `json:"remarks,omitempty"`          // Optional note from the approver
}
