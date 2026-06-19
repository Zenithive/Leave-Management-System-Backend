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

// LeaveResponse - API response for leave details (with joins)
type LeaveResponse struct {
	ID              string           `db:"id" json:"id"`
	Employee        string           `db:"employee" json:"employee"`
	LeaveType       string           `db:"leave_type" json:"leave_type"`
	LeaveTypeID     int              `db:"leave_type_id" json:"leave_type_id"`
	IsPaid          bool             `db:"is_paid" json:"is_paid"`
	IsEarly         *bool            `db:"is_early" json:"is_early"`
	LeaveTimingType string           `db:"leave_timing_type" json:"leave_timing_type"` // FIRST_HALF, SECOND_HALF, FULL, EARLY
	LeaveTiming     *string          `db:"leave_timing" json:"leave_timing"`           // Timing details
	StartDate       time.Time        `db:"start_date" json:"start_date"`
	EndDate         time.Time        `db:"end_date" json:"end_date"`
	Days            float64          `db:"days" json:"days"`
	Reason          string           `db:"reason" json:"reason"`
	Status          string           `db:"status" json:"status"`
	AppliedAt       time.Time        `db:"applied_at" json:"applied_at"`
	ApprovalName    *string          `db:"approval_name" json:"approval_name,omitempty"`
	ApprovalLog     []LeaveFlowStage `json:"approval_log" db:"approval_log"`
}
