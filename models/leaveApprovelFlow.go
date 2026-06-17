package models

import (
	"encoding/json"
	"time"
)

// ------------------------------
// Role Types
// ------------------------------
type ApproverRole string

const (
	ApproverSuperAdmin ApproverRole = "SUPERADMIN"
	ApproverHR         ApproverRole = "HR"
	ApproverAdmin      ApproverRole = "ADMIN"
	ApproverManager    ApproverRole = "MANAGER"
)

// ------------------------------
// Single Stage in Flow
// ------------------------------
type ApprovalStage struct {
	StageNo      int          `json:"stage_no" binding:"required,min=1"`
	ApproverRole ApproverRole `json:"approver_role" binding:"required,oneof=SUPERADMIN HR ADMIN MANAGER"`
}

type LeaveApprovalFlow struct {
	ID        string          `json:"id" db:"id"`
	Name      string          `json:"name" db:"name"`
	Flow      json.RawMessage `json:"flow" db:"flow"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// ------------------------------
// Request (Applicant → Multiple Approvers)
// ------------------------------
type LeaveApprovalFlowRequest struct {
	Name string          `json:"name"`
	Flow []ApprovalStage `json:"flow"`
}

// ------------------------------
// Response Model
// ------------------------------
type LeaveApprovalFlowResponse struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Flow []ApprovalStage `json:"flow"`
}
