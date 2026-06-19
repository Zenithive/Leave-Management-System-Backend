package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

type LeaveFlowRepository interface {
	InsertLeave(tx *sqlx.Tx, leave *models.LeaveInput, leaveTimingStr *string) (uuid.UUID, error)
	GetByID(ctx context.Context, leaveID string) (*models.Leave, error)
}

type leaveFlow struct {
	DB *sqlx.DB
}

func NewLeaveFlow(db *sqlx.DB) LeaveFlowRepository {
	return &leaveFlow{
		DB: db,
	}
}

func (r *leaveFlow) InsertLeave(tx *sqlx.Tx, leave *models.LeaveInput, leaveTimingStr *string) (uuid.UUID, error) {

	var leaveID uuid.UUID

	err := tx.QueryRow(`
		INSERT INTO Tbl_Leave 
		(employee_id, leave_type_id, half_id, start_date, end_date, days, status, reason, leave_timing)
		VALUES ($1,$2,$3,$4,$5,$6,'Pending',$7,$8)
		RETURNING id
	`,
		leave.EmployeeID,
		leave.LeaveTypeID,
		leave.LeaveTimingID,
		leave.StartDate,
		leave.EndDate,
		leave.Days,
		leave.Reason,
		leaveTimingStr,
	).Scan(&leaveID)

	return leaveID, err
}

func (r *leaveFlow) GetByID(ctx context.Context, leaveID string) (*models.Leave, error) {

	var leave models.Leave

	query := `SELECT * FROM tbl_leave WHERE id = $1`

	err := r.DB.GetContext(
		ctx,
		&leave,
		query,
		leaveID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &leave, nil
}
