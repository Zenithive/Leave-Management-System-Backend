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
