package models

import "time"

// Leave Timing
type LeaveTimingResponse struct {
	ID        int        `json:"id" db:"id"`
	Type      string     `json:"type" db:"type"`
	Timing    string     `json:"timing" db:"timing"`
	CreatedAt *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
}
type UpdateLeaveTimingReq struct {
	ID     int    `uri:"id" validate:"required,oneof=1 2 3"`
	Timing string `json:"timing" validate:"required"`
}

type GetLeaveTimingByIDReq struct {
	ID int `uri:"id" validate:"required,oneof=1 2 3"`
}

// LeaveUpdateInput is used when an employee edits their own pending leave
type LeaveUpdateInput struct {
	LeaveTypeID   int       `json:"leave_type_id" validate:"required"`
	LeaveTimingID *int      `json:"leave_timing_id,omitempty"`
	LeaveTiming   *string   `json:"leave_timing,omitempty"`
	StartDate     time.Time `json:"start_date" validate:"required"`
	EndDate       time.Time `json:"end_date" validate:"required"`
	Reason        string    `json:"reason" validate:"required,min=10,max=500"`
}
