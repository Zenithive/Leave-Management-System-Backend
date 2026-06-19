package repositories

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

type LeaveFlowLog interface {
	Create(ctx context.Context, tx *sqlx.Tx, leaveFlow *models.LeaveFlow) error
	GetByLeaveID(ctx context.Context, leaveID uuid.UUID) (*models.LeaveFlowDB, error)
	UpdateApprovalLog(ctx context.Context, tx *sqlx.Tx, leaveID uuid.UUID, log []models.LeaveFlowStage) error
}

type leaveFlowLog struct {
	DB *sqlx.DB
}

func NewLeaveFlowLog(db *sqlx.DB) LeaveFlowLog {
	return &leaveFlowLog{
		DB: db,
	}
}

func (s *leaveFlowLog) Create(ctx context.Context, tx *sqlx.Tx, leaveFlow *models.LeaveFlow) error {

	approvalLogJSON, err := json.Marshal(leaveFlow.ApprovalLog)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO Tbl_Leave_Flow (
			leave_id,
			approval_log
		)
		VALUES ($1, $2)
	`
	_, err = tx.ExecContext(
		ctx,
		query,
		leaveFlow.LeaveID,
		approvalLogJSON,
	)
	return err
}

func (r *leaveFlowLog) GetByLeaveID(ctx context.Context, leaveID uuid.UUID) (*models.LeaveFlowDB, error) {

	var flow models.LeaveFlowDB

	query := `
		SELECT * FROM Tbl_Leave_Flow WHERE leave_id = $1 AND deleted_at IS NULL`

	err := r.DB.GetContext(ctx, &flow, query, leaveID)
	if err != nil {
		return nil, err
	}

	return &flow, nil
}

// UpdateApprovalLog re-marshals the mutated stage slice and writes it back to the DB.
// Called inside an open transaction after a processor has stamped the relevant stage.
func (r *leaveFlowLog) UpdateApprovalLog(ctx context.Context, tx *sqlx.Tx, leaveID uuid.UUID, log []models.LeaveFlowStage) error {
	data, err := json.Marshal(log)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE Tbl_Leave_Flow SET approval_log = $1, updated_at = NOW() WHERE leave_id = $2 AND deleted_at IS NULL`,
		data, leaveID,
	)
	return err
}
