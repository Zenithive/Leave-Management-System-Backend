package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

type LeaveFlowRepository interface {
	InsertLeave(tx *sqlx.Tx, leave *models.LeaveInput, leaveTimingStr *string) (uuid.UUID, error)
	GetAllEmployeeLeaveByMonthYear(userID uuid.UUID, month, year int) ([]models.LeaveResponse, error)
	GetAllleavebaseonassignManagerByMonthYear(userID uuid.UUID, month, year int) ([]models.LeaveResponse, error)
	GetAllLeaveByMonthYear(month, year int) ([]models.LeaveResponse, error)
	GetMyLeavesByMonthYear(userID uuid.UUID, month, year int) ([]models.LeaveResponse, error)
	GetByID(ctx context.Context, leaveID string) (*models.Leave, error)
	UpdateLeaveStatus(leaveID string, status string) error
	UpdateLeaveStatusTx(tx *sql.Tx, leaveID uuid.UUID, status string, approverID uuid.UUID) error
	UpdateLeave(tx *sqlx.Tx, leaveID uuid.UUID, empID uuid.UUID, input *models.LeaveInput, NewDays float64) error
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

func (r *leaveFlow) GetAllEmployeeLeaveByMonthYear(userID uuid.UUID, month, year int) ([]models.LeaveResponse, error) {

	var result []models.LeaveResponse

	query := `
	SELECT 
		l.id,
		e.full_name AS employee,
		lt.name AS leave_type,
		l.leave_type_id,
		lt.is_paid,
		lt.is_early,

		CASE 
			WHEN lt.is_early = true 
			THEN l.leave_timing
			ELSE COALESCE(h.timing, 'Full Day')
		END AS leave_timing,

		CASE 
			WHEN lt.is_early = true 
			THEN 'EARLY'
			ELSE COALESCE(h.type, 'FULL')
		END AS leave_timing_type,

		l.start_date,
		l.end_date,
		l.days,
		COALESCE(l.reason,'') AS reason,
		l.status,
		l.created_at AS applied_at,
		approver.full_name AS approval_name

	FROM Tbl_Leave l

	INNER JOIN Tbl_Employee e
		ON l.employee_id = e.id

	INNER JOIN Tbl_Leave_Type lt
		ON lt.id = l.leave_type_id

	LEFT JOIN Tbl_Half h
		ON l.half_id = h.id

	LEFT JOIN Tbl_Employee approver
		ON l.approved_by = approver.id

	WHERE 
		l.employee_id = $1

		AND l.start_date::date <= (
			DATE_TRUNC('month', MAKE_DATE($3, $2, 1))
			+ INTERVAL '1 month - 1 day'
		)::date

		AND l.end_date::date >=
			DATE_TRUNC('month', MAKE_DATE($3, $2, 1))::date

	ORDER BY 
		l.start_date ASC,
		l.created_at DESC
	`

	err := r.DB.Select(&result, query, userID, month, year)

	return result, err
}

func (r *leaveFlow) GetAllleavebaseonassignManagerByMonthYear(userID uuid.UUID, month, year int) ([]models.LeaveResponse, error) {

	var result []models.LeaveResponse

	query := `
	SELECT l.id,
		e.full_name AS employee,
		lt.name AS leave_type,
		l.leave_type_id,
		lt.is_paid,
		lt.is_early,

		CASE 
			WHEN lt.is_early = true 
			THEN l.leave_timing
			ELSE COALESCE(h.timing, 'Full Day')
		END AS leave_timing,

		CASE 
			WHEN lt.is_early = true 
			THEN 'EARLY'
			ELSE COALESCE(h.type, 'FULL')
		END AS leave_timing_type,

		l.start_date,
		l.end_date,
		l.days,
		COALESCE(l.reason,'') AS reason,
		l.status,
		l.created_at AS applied_at,
		approver.full_name AS approval_name

	FROM Tbl_Leave l

	INNER JOIN Tbl_Employee e
		ON l.employee_id = e.id

	INNER JOIN Tbl_Leave_Type lt
		ON lt.id = l.leave_type_id

	LEFT JOIN Tbl_Half h
		ON l.half_id = h.id

	LEFT JOIN Tbl_Employee approver
		ON l.approved_by = approver.id

	WHERE
		(e.manager_id = $1 OR l.employee_id = $1)

		AND l.start_date::date <= (
			DATE_TRUNC('month', MAKE_DATE($3, $2, 1))
			+ INTERVAL '1 month - 1 day'
		)::date

		AND l.end_date::date >=
			DATE_TRUNC('month', MAKE_DATE($3, $2, 1))::date

	ORDER BY
		l.start_date ASC,
		l.created_at DESC
	`

	err := r.DB.Select(&result, query, userID, month, year)

	return result, err
}

func (r *leaveFlow) GetAllLeaveByMonthYear(month, year int) ([]models.LeaveResponse, error) {

	var result []models.LeaveResponse

	query := `
	SELECT 
		l.id,
		e.full_name AS employee,
		lt.name AS leave_type,
		l.leave_type_id,
		lt.is_paid,
		lt.is_early,

		CASE 
			WHEN lt.is_early = true 
			THEN l.leave_timing
			ELSE COALESCE(h.timing, 'Full Day')
		END AS leave_timing,

		CASE 
			WHEN lt.is_early = true 
			THEN 'EARLY'
			ELSE COALESCE(h.type, 'FULL')
		END AS leave_timing_type,

		l.start_date,
		l.end_date,
		l.days,
		COALESCE(l.reason,'') AS reason,
		l.status,
		l.created_at AS applied_at,
		approver.full_name AS approval_name

	FROM Tbl_Leave l

	INNER JOIN Tbl_Employee e
		ON l.employee_id = e.id

	INNER JOIN Tbl_Leave_Type lt
		ON lt.id = l.leave_type_id

	LEFT JOIN Tbl_Half h
		ON l.half_id = h.id

	LEFT JOIN Tbl_Employee approver
		ON l.approved_by = approver.id

	WHERE
		l.start_date::date <= (
			DATE_TRUNC('month', MAKE_DATE($2, $1, 1))
			+ INTERVAL '1 month - 1 day'
		)::date

		AND l.end_date::date >=
			DATE_TRUNC('month', MAKE_DATE($2, $1, 1))::date

	ORDER BY
		l.start_date ASC,
		l.created_at DESC
	`

	err := r.DB.Select(&result, query, month, year)

	return result, err
}

func (r *leaveFlow) GetMyLeavesByMonthYear(userID uuid.UUID, month, year int) ([]models.LeaveResponse, error) {

	var result []models.LeaveResponse

	query := `
	SELECT 
		l.id,
		e.full_name AS employee,
		lt.name AS leave_type,
		lt.is_paid,
		lt.is_early,
		l.leave_type_id,

		CASE 
			WHEN lt.is_early = true 
			THEN l.leave_timing
			ELSE COALESCE(h.timing, 'Full Day')
		END AS leave_timing,

		CASE 
			WHEN lt.is_early = true 
			THEN 'EARLY'
			ELSE COALESCE(h.type, 'FULL')
		END AS leave_timing_type,

		l.start_date,
		l.end_date,
		l.days,
		COALESCE(l.reason, '') AS reason,
		l.status,
		l.created_at AS applied_at,
		approver.full_name AS approval_name

	FROM Tbl_Leave l

	INNER JOIN Tbl_Employee e 
		ON l.employee_id = e.id

	INNER JOIN Tbl_Leave_Type lt 
		ON lt.id = l.leave_type_id

	LEFT JOIN Tbl_Half h 
		ON l.half_id = h.id

	LEFT JOIN Tbl_Employee approver 
		ON l.approved_by = approver.id

	WHERE 
		l.employee_id = $1

		AND l.start_date::date <= (
			DATE_TRUNC('month', MAKE_DATE($3, $2, 1))
			+ INTERVAL '1 month - 1 day'
		)::date

		AND l.end_date::date >=
			DATE_TRUNC('month', MAKE_DATE($3, $2, 1))::date

	ORDER BY 
		l.start_date ASC,
		l.created_at DESC
	`

	err := r.DB.Select(&result, query, userID, month, year)

	return result, err
}

func (r *leaveFlow) UpdateLeaveStatusTx(tx *sql.Tx, leaveID uuid.UUID, status string, approverID uuid.UUID) error {
	query := `UPDATE Tbl_Leave SET status = $1, approved_by = $2, updated_at = NOW() WHERE id = $3`
	_, err := tx.Exec(query, status, approverID, leaveID)
	return err
}
func (r *leaveFlow) UpdateLeaveStatus(leaveID string, status string) error {
	query := `UPDATE Tbl_Leave SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.DB.Exec(query, status, leaveID)
	return err
}

func (r *leaveFlow) UpdateLeave(tx *sqlx.Tx, leaveID uuid.UUID, empID uuid.UUID, input *models.LeaveInput, NewDays float64) error {

	// 2. RE-CALCULATE DAYS using your existing service
	// Ensure you pass the correct timingID (1, 2, or 3)

	query := `
        UPDATE Tbl_Leave
        SET 
            start_date = $1, 
            end_date = $2, 
            leave_type_id = $3, 
            reason = $4,
			days = $5,           
            half_id = $6,
            updated_at = NOW()
        WHERE id = $7 
          AND employee_id = $8 
          AND status = 'Pending'`

	result, err := tx.Exec(query,
		input.StartDate,
		input.EndDate,
		input.LeaveTypeID,
		input.Reason,
		NewDays,
		input.LeaveTimingID,
		leaveID,
		empID,
	)
	if err != nil {
		return err
	}

	// Check if any row was actually updated
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("leave cannot be edited: either it does not exist, you don't own it, or it is already processed")
	}

	return nil
}
