package models

import (
	"time"
)

// ============================================================================
// LEAVE APPLICATION MODELS
// ============================================================================

// LeaveUpdateInput - Request payload for editing a pending leave application
type LeaveUpdateInput struct {
	LeaveTypeID   int       `json:"leave_type_id" validate:"required"`
	LeaveTimingID *int      `json:"leave_timing_id,omitempty"`
	LeaveTiming   *string   `json:"leave_timing,omitempty"`
	StartDate     time.Time `json:"start_date" validate:"required"`
	EndDate       time.Time `json:"end_date" validate:"required"`
	Reason        string    `json:"reason" validate:"required,min=10,max=500"`
}

// LeaveCountSummary - Statistics of leaves by status
type LeaveCountSummary struct {
	Total     int `json:"total"`
	Pending   int `json:"pending"`
	Approved  int `json:"approved"`
	Rejected  int `json:"rejected"`
	Cancelled int `json:"cancelled"`
	Withdrawn int `json:"withdrawn"`
}

// BuildLeaveCountSummary computes status counts from a slice of LeaveResponse
// Reusable across all role-based leave list endpoints
func BuildLeaveCountSummary(leaves []LeaveResponse) *LeaveCountSummary {
	summary := LeaveCountSummary{Total: len(leaves)}
	for _, l := range leaves {
		switch l.Status {
		case "Pending":
			summary.Pending++
		case "APPROVED":
			summary.Approved++
		case "REJECTED":
			summary.Rejected++
		case "CANCELLED":
			summary.Cancelled++
		case "WITHDRAWN":
			summary.Withdrawn++
		}
	}
	return &summary
}

// DailyLeaveRecord - Used for daily Slack notification of active leaves
type DailyLeaveRecord struct {
	EmployeeName string    `db:"employee_name"`
	LeaveType    string    `db:"leave_type"`
	StartDate    time.Time `db:"start_date"`
	EndDate      time.Time `db:"end_date"`
	Days         float64   `db:"days"`
	Status       string    `db:"status"`
	ApprovedBy   *string   `db:"approved_by"`
}
