package repositories

import (
	"database/sql"
	"time"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// 1. Get leave type entitlement
func (r *Repository) GetLeaveTypeByIdTx(tx *sqlx.Tx, leaveTypeID int) (models.LeaveType, error) {
	var leaves models.LeaveType
	query := `SELECT id, name, is_paid, default_entitlement, intern_entitlement, is_early, is_work_from_home, created_at, updated_at FROM Tbl_Leave_type WHERE id=$1`
	err := tx.Get(&leaves,
		query,
		leaveTypeID,
	)
	return leaves, err
}

// 1. Get leave type entitlement
func (r *Repository) GetLeaveTypeById(leaveTypeID int) (models.LeaveType, error) {
	var leaves models.LeaveType
	query := `SELECT id, name, is_paid, default_entitlement, intern_entitlement, is_early, is_work_from_home, created_at, updated_at FROM Tbl_Leave_type WHERE id=$1`
	err := r.DB.Get(&leaves,
		query,
		leaveTypeID,
	)
	return leaves, err
}

func (q *Repository) GetLeaveTypeByLeaveID(leaveID uuid.UUID) (int, error) {
	var leaveTypeID int
	err := q.DB.Get(&leaveTypeID, `
        SELECT leave_type_id 
        FROM Tbl_Leave 
        WHERE id = $1
    `, leaveID)

	if err != nil {
		return 0, err
	}

	return leaveTypeID, nil
}

// 3. Get leave balance (inside TX)
// Returns 0 (not an error) when no balance row exists for the employee/type/year.
func (r *Repository) GetLeaveBalance(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int) (float64, error) {
	var balance float64

	err := tx.Get(&balance, `
		SELECT closing
		FROM Tbl_Leave_balance 
		WHERE employee_id=$1 AND leave_type_id=$2 
		AND year = EXTRACT(YEAR FROM CURRENT_DATE)
	`, employeeID, leaveTypeID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return balance, err
}

// GetPendingLeaveDays returns the total days of Pending  leaves
// for the current year for a given employee and leave type.
// These leaves are not yet deducted from the closing balance, so they must be
// accounted for when checking if a new leave application has sufficient balance.
func (r *Repository) GetPendingLeaveDays(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, excludeLeaveID *uuid.UUID) (float64, error) {
	var pendingDays float64

	query := `
		SELECT COALESCE(SUM(days), 0)
		FROM Tbl_Leave
		WHERE employee_id = $1
		  AND leave_type_id = $2
		  AND status IN ('Pending')
		  AND EXTRACT(YEAR FROM start_date) = EXTRACT(YEAR FROM CURRENT_DATE)
	`
	var err error
	if excludeLeaveID != nil {
		// When editing an existing leave, exclude it from the pending sum
		query += ` AND id != $3`
		err = tx.Get(&pendingDays, query, employeeID, leaveTypeID, *excludeLeaveID)
	} else {
		err = tx.Get(&pendingDays, query, employeeID, leaveTypeID)
	}
	return pendingDays, err
}

// create leave balance
func (r *Repository) CreateLeaveBalance(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, entitlement int) error {
	_, err := tx.Exec(`
		INSERT INTO Tbl_Leave_balance 
			(employee_id, leave_type_id, year, opening, accrued, used, adjusted, closing)
		VALUES ($1, $2, EXTRACT(YEAR FROM CURRENT_DATE), $3, 0, 0, 0, $3)
	`, employeeID, leaveTypeID, entitlement)
	return err
}

// 5. Check overlapping leaves
func (r *Repository) GetOverlappingLeaves(
	tx *sqlx.Tx,
	employeeID uuid.UUID,
	startDate, endDate time.Time,
	excludeLeaveID *uuid.UUID,
) ([]struct {
	ID        uuid.UUID `db:"id"`
	LeaveType string    `db:"leave_type"`
	StartDate time.Time `db:"start_date"`
	EndDate   time.Time `db:"end_date"`
	Status    string    `db:"status"`
}, error) {

	var result []struct {
		ID        uuid.UUID `db:"id"`
		LeaveType string    `db:"leave_type"`
		StartDate time.Time `db:"start_date"`
		EndDate   time.Time `db:"end_date"`
		Status    string    `db:"status"`
	}

	query := `
		SELECT l.id, lt.name as leave_type, l.start_date, l.end_date, l.status
		FROM Tbl_Leave l
		JOIN Tbl_Leave_type lt ON l.leave_type_id = lt.id
		WHERE l.employee_id=$1 
		AND l.status IN ('Pending','APPROVED')
		AND l.start_date <= $2 
		AND l.end_date >= $3
	`

	var err error
	if excludeLeaveID != nil {
		query += ` AND l.id != $4`
		err = tx.Select(&result, query, employeeID, endDate, startDate, *excludeLeaveID)
	} else {
		err = tx.Select(&result, query, employeeID, endDate, startDate)
	}

	return result, err
}

func (r *Repository) UpdateLeaveStatus(tx *sql.Tx, leaveID uuid.UUID, status string) error {
	query := `UPDATE Tbl_Leave SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.Exec(query, status, leaveID)
	return err
}

func (r *Repository) GetLeaveById(leaveID uuid.UUID) (models.Leave, error) {
	var leave models.Leave
	query := `SELECT * FROM Tbl_Leave WHERE id=$1 FOR UPDATE`
	err := r.DB.Get(&leave, query, leaveID)
	return leave, err
}

// GetLeaveDaysByID returns the days value of a specific leave inside a transaction.
// Used during edit validation to add back the original leave's days to available balance.
func (r *Repository) GetLeaveDaysByID(tx *sqlx.Tx, leaveID uuid.UUID) (float64, error) {
	var days float64
	err := tx.Get(&days, `SELECT days FROM Tbl_Leave WHERE id = $1`, leaveID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return days, err
}

func (r *Repository) GetLeaveApprovalNameByEmployeeID(approvalId uuid.UUID) (string, error) {
	var approvalName string
	query := `SELECT full_name FROM Tbl_Employee WHERE id=$1`
	err := r.DB.Get(&approvalName, query, approvalId)
	return approvalName, err
}

// GetMyLeavesByMonthYear - Get current user's leaves from given month/year onward (current + future).
// When month/year is sent as base, returns leaves where start_date >= first day of that month.

func (r *Repository) UpdateLeaveStatusWithResion(tx *sqlx.Tx, withdrawalReason string, currentUserID uuid.UUID, leaveID uuid.UUID, status string) error {
	_, err := tx.Exec(`
			UPDATE Tbl_Leave 
			SET status=$1, reason=$2, approved_by=$3, updated_at=NOW() 
			WHERE id=$4
		`, status, withdrawalReason, currentUserID, leaveID)
	return err
}

// GetEarlyLeaveThisMonth checks if the employee already has an early leave
// for the given leave type in the same month/year as refDate.
// Returns the existing leave if found, nil if not.
func (r *Repository) GetEarlyLeaveThisMonth(
	tx *sqlx.Tx,
	employeeID uuid.UUID,
	leaveTypeID int,
	refDate time.Time,
) (*models.Leave, error) {

	var leave models.Leave

	err := tx.Get(&leave, `
		SELECT l.*
		FROM Tbl_Leave l
		JOIN Tbl_Leave_type lt ON lt.id = l.leave_type_id
		WHERE l.employee_id = $1
		  AND l.leave_type_id = $2
		  AND lt.is_early = TRUE
		  AND l.status IN ('Pending', 'APPROVED')
		  AND EXTRACT(YEAR  FROM l.start_date) = EXTRACT(YEAR  FROM $3::date)
		  AND EXTRACT(MONTH FROM l.start_date) = EXTRACT(MONTH FROM $3::date)
		LIMIT 1
	`, employeeID, leaveTypeID, refDate)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &leave, nil
}

// GetTodaysActiveLeaves fetches all leaves that are active today
// (start_date <= today AND end_date >= today)
// Used for daily Slack notification cron job
func (r *Repository) GetTodaysActiveLeaves() ([]models.DailyLeaveRecord, error) {
	var leaves []models.DailyLeaveRecord

	query := `
		SELECT 
			e.full_name AS employee_name,
			lt.name AS leave_type,
			l.start_date,
			l.end_date,
			l.days,
			l.status,
			approver.full_name AS approved_by
		FROM Tbl_Leave l
		JOIN Tbl_Employee e ON l.employee_id = e.id
		JOIN Tbl_Leave_Type lt ON l.leave_type_id = lt.id
		LEFT JOIN Tbl_Employee approver ON l.approved_by = approver.id
		WHERE l.start_date <= CURRENT_DATE
		  AND l.end_date >= CURRENT_DATE
		  AND l.status IN ('APPROVED', 'MANAGER_APPROVED','Pending')
		ORDER BY e.full_name
	`

	err := r.DB.Select(&leaves, query)
	return leaves, err
}
