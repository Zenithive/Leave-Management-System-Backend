package repositories

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type LeaveFlowLog interface {
	Create(ctx context.Context, tx *sqlx.Tx, leaveFlow *models.LeaveFlow) error
	GetByLeaveID(ctx context.Context, leaveID uuid.UUID) (*models.LeaveFlowDB, error)
	UpdateApprovalLog(ctx context.Context, tx *sqlx.Tx, leaveID uuid.UUID, log []models.LeaveFlowStage) error
	// GetApproverNames fetches employee names for a set of IDs in one query.
	GetApproverNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
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

	query := `SELECT * FROM Tbl_Leave_Flow WHERE leave_id = $1 AND deleted_at IS NULL`

	err := r.DB.GetContext(ctx, &flow, query, leaveID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &flow, nil
}

// UpdateApprovalLog re-marshals the mutated stage slice and writes it back to the DB.
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

// GetApproverNames fetches full_name for a set of employee UUIDs in one query.
// Returns a map of id → name for enriching approval log stages.
func (r *leaveFlowLog) GetApproverNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]string{}, nil
	}

	type row struct {
		ID       uuid.UUID `db:"id"`
		FullName string    `db:"full_name"`
	}

	var rows []row
	query, args, err := sqlx.In(
		`SELECT id, full_name FROM Tbl_Employee WHERE id IN (?)`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	// sqlx.In uses ? placeholders — rebind for postgres ($1, $2 …)
	query = r.DB.Rebind(query)
	if err := r.DB.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		result[row.ID] = row.FullName
	}
	return result, nil
}
