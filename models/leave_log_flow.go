package models

import (
	"time"

	"github.com/google/uuid"
)

type State string

const (
	WAITING   State = "WAITING"
	APPROVED  State = "APPROVED"
	REJECTED  State = "REJECTED"
	WITHDRAWN State = "WITHDRAWN"
	SKIPPED   State = "SKIPPED"
)

type LeaveFlowStage struct {
	StageNo      int          `json:"stage_no" binding:"required,min=1"`
	ApproverRole ApproverRole `json:"approver_role" binding:"required,oneof=SUPERADMIN HR ADMIN MANAGER"`

	State State `json:"state"`

	ApprovedBy *uuid.UUID `json:"approved_by,omitempty"`

	Remarks string `json:"remarks,omitempty"`

	ActionAt *time.Time `json:"action_at,omitempty"`
}

type LeaveFlow struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	LeaveID     uuid.UUID        `json:"leave_id" db:"leave_id"`
	ApprovalLog []LeaveFlowStage `json:"approval_log" db:"approval_log"`

	CreatedAt *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

type LeaveFlowDB struct {
	ID          uuid.UUID  `db:"id"`
	LeaveID     uuid.UUID  `db:"leave_id"`
	ApprovalLog []byte     `db:"approval_log"`
	CreatedAt   *time.Time `db:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}
