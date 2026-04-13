package models

import "time"

type LeaveTypeInput struct {
	Name               string `json:"name" validate:"required"`
	IsPaid             *bool  `json:"is_paid,omitempty"`
	IsEarly            *bool  `json:"is_early,omitempty" validate:"omitempty"`
	DefaultEntitlement *int   `json:"default_entitlement,omitempty"`
	LeaveCount         *int   `json:"leave_count,omitempty" validate:"omitempty,gt=0"`
}

// ----------------- LEAVE TYPE -----------------
type LeaveType struct {
	ID                 int       `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	IsPaid             bool      `json:"is_paid" db:"is_paid"`
	DefaultEntitlement int       `json:"default_entitlement" db:"default_entitlement"`
	IsEarly            *bool     `json:"is_early,omitempty" db:"is_early"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}
