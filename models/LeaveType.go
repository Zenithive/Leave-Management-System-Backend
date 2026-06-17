package models

import "time"

type LeaveTypeInput struct {
	Name               string `json:"name" validate:"required"`
	IsPaid             *bool  `json:"is_paid,omitempty"`
	IsEarly            *bool  `json:"is_early,omitempty" validate:"omitempty"`
	IsWorkFromHome     *bool  `json:"is_work_from_home,omitempty"`
	DefaultEntitlement *int   `json:"default_entitlement,omitempty"`
	InternEntitlement  *int   `json:"intern_entitlement,omitempty"`
	LeaveCount         *int   `json:"leave_count,omitempty" validate:"omitempty,gt=0"`
	ApprovalFlowID     string `json:"approval_flow_id,omitempty"`
}

// ----------------- LEAVE TYPE -----------------
type LeaveType struct {
	ID                 int       `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	IsPaid             bool      `json:"is_paid" db:"is_paid"`
	DefaultEntitlement int       `json:"default_entitlement" db:"default_entitlement"`
	InternEntitlement  *int      `json:"intern_entitlement,omitempty" db:"intern_entitlement"`
	IsEarly            *bool     `json:"is_early,omitempty" db:"is_early"`
	IsWorkFromHome     bool      `json:"is_work_from_home" db:"is_work_from_home"`
	ApprovalFlowID     *string   `json:"approval_flow_id,omitempty" db:"approval_flow_id"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

type LeaveTypeResponse struct {
	LeaveType    LeaveType                  `json:"leave_type"`
	ApprovalFlow *LeaveApprovalFlowResponse `json:"approval_flow,omitempty"`
}

type LeaveTypeRow struct {
	ID                 int  `db:"id"`
	DefaultEntitlement int  `db:"default_entitlement"`
	InternEntitlement  *int `db:"intern_entitlement"`
}

func MappPayload(leavetype *LeaveType, leaveApprovalFlow *LeaveApprovalFlowResponse) *LeaveTypeResponse {
	return &LeaveTypeResponse{
		LeaveType: *leavetype,
		ApprovalFlow: &LeaveApprovalFlowResponse{
			ID:       *leavetype.ApprovalFlowID,
			Name:     leaveApprovalFlow.Name,
			IsSystem: leaveApprovalFlow.IsSystem,
			Flow:     leaveApprovalFlow.Flow,
		},
	}
}
