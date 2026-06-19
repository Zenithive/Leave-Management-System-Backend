package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
)

type LeaveFlowLog interface {
	Create(ctx context.Context, tx *sqlx.Tx, leaveID uuid.UUID, leaveTypeRes *models.LeaveTypeResponse, role string) error
	GetByLeaveID(ctx context.Context, leaveID uuid.UUID) (*models.LeaveFlow, error)
	UpdateApprovalLog(ctx context.Context, tx *sqlx.Tx, leaveID uuid.UUID, log []models.LeaveFlowStage) error
}

type leaveFlowLog struct {
	DB                 *sqlx.DB
	LeavePolicyService LeavePolicyService
	Repo               repositories.LeaveFlowLog
}

func NewLeaveFlowLog(db *sqlx.DB, leavePolicyService LeavePolicyService, leaveFlowLogRepo repositories.LeaveFlowLog) LeaveFlowLog {
	return &leaveFlowLog{
		DB:                 db,
		LeavePolicyService: leavePolicyService,
		Repo:               leaveFlowLogRepo,
	}
}

var roleLevels = map[string]int{
	"INTERN":     1,
	"EMPLOYEE":   2,
	"MANAGER":    3,
	"HR":         4,
	"ADMIN":      5,
	"SUPERADMIN": 6,
}

func getRoleLevel(role string) int {
	return roleLevels[role]
}

func (s *leaveFlowLog) Create(ctx context.Context, tx *sqlx.Tx, leaveID uuid.UUID, leaveTypeRes *models.LeaveTypeResponse, role string) error {

	if leaveTypeRes == nil || leaveTypeRes.ApprovalFlow == nil {
		return nil
	}
	applicantLevel := getRoleLevel(role)

	approvalLog := make([]models.LeaveFlowStage, 0, len(leaveTypeRes.ApprovalFlow.Flow))

	for _, stage := range leaveTypeRes.ApprovalFlow.Flow {
		if getRoleLevel(string(stage.ApproverRole)) <= applicantLevel {
			continue
		}

		approvalLog = append(approvalLog, models.LeaveFlowStage{
			StageNo:      stage.StageNo,
			ApproverRole: stage.ApproverRole,
			State:        models.WAITING,
		})
	}

	// No approvers required
	if len(approvalLog) == 0 {
		return nil
	}

	return s.Repo.Create(ctx, tx, &models.LeaveFlow{
		LeaveID:     leaveID,
		ApprovalLog: approvalLog,
	})
}

func (s *leaveFlowLog) GetByLeaveID(ctx context.Context, leaveID uuid.UUID) (*models.LeaveFlow, error) {

	dbFlow, err := s.Repo.GetByLeaveID(ctx, leaveID)
	if err != nil {
		return nil, err
	}

	var approvalLog []models.LeaveFlowStage

	if len(dbFlow.ApprovalLog) > 0 {
		if err := json.Unmarshal(dbFlow.ApprovalLog, &approvalLog); err != nil {
			return nil, err
		}
	}

	return &models.LeaveFlow{
		ID:          dbFlow.ID,
		LeaveID:     dbFlow.LeaveID,
		ApprovalLog: approvalLog,
		CreatedAt:   dbFlow.CreatedAt,
		UpdatedAt:   dbFlow.UpdatedAt,
		DeletedAt:   dbFlow.DeletedAt,
	}, nil
}

// UpdateApprovalLog delegates to the repository to persist the mutated stage slice.
func (s *leaveFlowLog) UpdateApprovalLog(ctx context.Context, tx *sqlx.Tx, leaveID uuid.UUID, log []models.LeaveFlowStage) error {
	return s.Repo.UpdateApprovalLog(ctx, tx, leaveID, log)
}
