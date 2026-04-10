package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

// 1. Get leave type entitlement
func (r *Repository) GetLeaveTypeByIdTx(tx *sqlx.Tx, leaveTypeID int) (models.LeaveType, error) {
	var leaves models.LeaveType
	query := `SELECT id, name, is_paid, default_entitlement, is_early, created_at, updated_at FROM Tbl_Leave_type WHERE id=$1`
	err := tx.Get(&leaves,
		query,
		leaveTypeID,
	)
	return leaves, err
}

// 1. Get leave type entitlement
func (r *Repository) GetLeaveTypeById(leaveTypeID int) (models.LeaveType, error) {
	var leaves models.LeaveType
	query := `SELECT id, name, is_paid, default_entitlement, is_early, created_at, updated_at FROM Tbl_Leave_type WHERE id=$1`
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
func (r *Repository) GetLeaveBalance(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int) (float64, error) {
	var balance float64
	err := tx.Get(&balance, `
		SELECT closing 
		FROM Tbl_Leave_balance 
		WHERE employee_id=$1 AND leave_type_id=$2 
		AND year = EXTRACT(YEAR FROM CURRENT_DATE)
	`, employeeID, leaveTypeID)
	return balance, err
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

	err := tx.Select(&result, `
		SELECT l.id, lt.name as leave_type, l.start_date, l.end_date, l.status
		FROM Tbl_Leave l
		JOIN Tbl_Leave_type lt ON l.leave_type_id = lt.id
		WHERE l.employee_id=$1 
		AND l.status IN ('Pending','APPROVED')
		AND l.start_date <= $2 
		AND l.end_date >= $3
	`, employeeID, endDate, startDate)

	return result, err
}

func (r *Repository) InsertLeave(
	tx *sqlx.Tx,
	employeeID uuid.UUID,
	leaveTypeID int,
	leaveTimingID int,
	startDate, endDate time.Time,
	days float64,
	reason string,
	leaveTiming *string,
) (uuid.UUID, error) {

	var leaveID uuid.UUID

	err := tx.QueryRow(`
		INSERT INTO Tbl_Leave 
		(employee_id, leave_type_id, half_id, start_date, end_date, days, status, reason, leave_timing)
		VALUES ($1,$2,$3,$4,$5,$6,'Pending',$7,$8)
		RETURNING id
	`,
		employeeID,
		leaveTypeID,
		leaveTimingID,
		startDate,
		endDate,
		days,
		reason,
		leaveTiming,
	).Scan(&leaveID)

	return leaveID, err
}

func (r *Repository) UpdateLeaveStatus(tx *sql.Tx, leaveID uuid.UUID, status string) error {
	query := `UPDATE Tbl_Leave SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.Exec(query, status, leaveID)
	return err
}

func (r *Repository) GetLeaveById(tx *sqlx.Tx, leaveID uuid.UUID) (models.Leave, error) {
	var leave models.Leave
	query := `SELECT * FROM Tbl_Leave WHERE id=$1 FOR UPDATE`
	err := tx.Get(&leave, query, leaveID)
	return leave, err
}

// Get All Leave Timming
// Get All Leave Timing
func (r *Repository) GetLeaveTiming() ([]models.LeaveTimingResponse, error) {
	var data []models.LeaveTimingResponse
	query := `
		SELECT id, type, timing, created_at, updated_at
		FROM Tbl_Half
		ORDER BY id
	`
	err := r.DB.Select(&data, query)

	return data, err
}

// Get Leave Timing By ID
func (r *Repository) GetLeaveTimingByID(id int) (*models.LeaveTimingResponse, error) {
	var data models.LeaveTimingResponse

	query := `
		SELECT *
		FROM Tbl_Half
		WHERE id = $1
	`

	err := r.DB.Get(&data, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return &data, nil
}

func (r *Repository) UpdateLeaveTiming(tx *sqlx.Tx, id int, timing string) error {
	query := `
		UPDATE Tbl_Half
		SET timing = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	res, err := tx.Exec(query, timing, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// rolde base get leave

// GetAllEmployeeLeaveByMonthYear - Get employee leaves from given month/year onward (current + future).
// When month/year is sent as base, returns leaves where start_date >= first day of that month.
func (r *Repository) GetAllEmployeeLeaveByMonthYear(userID uuid.UUID, month, year int) ([]models.LeaveResponse, error) {
	var result []models.LeaveResponse
	query := `
		SELECT 
			l.id,
			e.full_name AS employee,
			lt.name AS leave_type,
			lt.is_paid AS is_paid,
			COALESCE(h.type, 'FULL') AS leave_timing_type,
			COALESCE(h.timing, 'Full Day') AS leave_timing,
			l.start_date,
			l.end_date,
			l.days,
			COALESCE(l.reason, '') AS reason,
			l.status,
			l.created_at AS applied_at,
			approver.full_name AS approval_name
		FROM Tbl_Leave l
		INNER JOIN Tbl_Employee e ON l.employee_id = e.id
		INNER JOIN Tbl_Leave_Type lt ON lt.id = l.leave_type_id
		LEFT JOIN Tbl_Half h ON l.half_id = h.id
		LEFT JOIN Tbl_Employee approver ON l.approved_by = approver.id
		WHERE l.employee_id = $1
		AND l.start_date >= ($3 || '-' || $2 || '-01')::date
		ORDER BY l.start_date ASC, l.created_at DESC`

	err := r.DB.Select(&result, query, userID, month, year)
	return result, err
}

// MAKE_DATE($3, $2, 1)
// GetAllleavebaseonassignManagerByMonthYear - Get manager's team leaves from given month/year onward (current + future).
// When month/year is sent as base, returns leaves where start_date >= first day of that month.
func (r *Repository) GetAllleavebaseonassignManagerByMonthYear(userID uuid.UUID, month, year int) ([]models.LeaveResponse, error) {
	var result []models.LeaveResponse
	query := `
		SELECT 
			l.id,
			e.full_name AS employee,
			lt.name AS leave_type,
			lt.is_paid AS is_paid,
			COALESCE(h.type, 'FULL') AS leave_timing_type,
			COALESCE(h.timing, 'Full Day') AS leave_timing,
			l.start_date,
			l.end_date,
			l.days,
			COALESCE(l.reason, '') AS reason,
			l.status,
			l.created_at AS applied_at,
			approver.full_name AS approval_name
		FROM Tbl_Leave l
		INNER JOIN Tbl_Employee e ON l.employee_id = e.id
		INNER JOIN Tbl_Leave_Type lt ON lt.id = l.leave_type_id
		LEFT JOIN Tbl_Half h ON l.half_id = h.id
		LEFT JOIN Tbl_Employee approver ON l.approved_by = approver.id
		WHERE (e.manager_id = $1 OR l.employee_id = $1)
		AND l.start_date >= ($3 || '-' || $2 || '-01')::date
		ORDER BY l.start_date ASC, l.created_at DESC`

	err := r.DB.Select(&result, query, userID, month, year)
	return result, err
}

// GetAllLeaveByMonthYear - Get all leaves from given month/year onward (current + future). Admin/HR/SuperAdmin.
// When month/year is sent as base, returns leaves where start_date >= first day of that month.
func (r *Repository) GetAllLeaveByMonthYear(month, year int) ([]models.LeaveResponse, error) {
	var result []models.LeaveResponse
	query := `
		SELECT 
			l.id,
			e.full_name AS employee,
			lt.name AS leave_type,
			l.leave_type_id,
			lt.is_paid AS is_paid,
			COALESCE(h.type, 'FULL') AS leave_timing_type,
			COALESCE(h.timing, 'Full Day') AS leave_timing,
			l.start_date,
			l.end_date,
			l.days,
			COALESCE(l.reason, '') AS reason,
			l.status,
			l.created_at AS applied_at,
			approver.full_name AS approval_name
		FROM Tbl_Leave l
		INNER JOIN Tbl_Employee e ON l.employee_id = e.id
		INNER JOIN Tbl_Leave_Type lt ON lt.id = l.leave_type_id
		LEFT JOIN Tbl_Half h ON l.half_id = h.id
		LEFT JOIN Tbl_Employee approver ON l.approved_by = approver.id
		WHERE l.start_date >= ($2 || '-' || $1 || '-01')::date
		ORDER BY l.start_date ASC, l.created_at DESC`

	err := r.DB.Select(&result, query, month, year)
	return result, err
}

// GetMyLeavesByMonthYear - Get current user's leaves from given month/year onward (current + future).
// When month/year is sent as base, returns leaves where start_date >= first day of that month.
func (r *Repository) GetMyLeavesByMonthYear(userID uuid.UUID, month, year int) ([]models.LeaveResponse, error) {
	var result []models.LeaveResponse
	query := `
		SELECT 
			l.id,
			e.full_name AS employee,
			lt.name AS leave_type,
			lt.is_paid AS is_paid,
			l.leave_type_id,
			COALESCE(h.type, 'FULL') AS leave_timing_type,
			COALESCE(h.timing, 'Full Day') AS leave_timing,
			l.start_date,
			l.end_date,
			l.days,
			COALESCE(l.reason, '') AS reason,
			l.status,
			l.created_at AS applied_at,
			approver.full_name AS approval_name
		FROM Tbl_Leave l
		INNER JOIN Tbl_Employee e ON l.employee_id = e.id
		INNER JOIN Tbl_Leave_Type lt ON lt.id = l.leave_type_id
		LEFT JOIN Tbl_Half h ON l.half_id = h.id
		LEFT JOIN Tbl_Employee approver ON l.approved_by = approver.id
		WHERE l.employee_id = $1
		AND l.start_date >= MAKE_DATE($3, $2, 1)
		ORDER BY l.start_date ASC, l.created_at DESC`

	err := r.DB.Select(&result, query, userID, month, year)
	return result, err
}

// UpdateLeaveBalancesForEntitlementChange updates leave balances when default entitlement changes
// This updates opening and closing balances for all employees with this leave type in the current year
// Example: If default entitlement changes from 18 to 20:
//   - Opening 18 → 20 (adds +2)
//   - Closing 18 → 20 (adds +2, maintains available balance)
//   - Closing 15 → 17 (if 3 days used, maintains same available balance)
func (r *Repository) UpdateLeaveBalancesForEntitlementChange(tx *sqlx.Tx, leaveTypeID int, oldDefaultEntitlement, newDefaultEntitlement int, currentYear int) error {
	// Calculate the difference
	difference := float64(newDefaultEntitlement - oldDefaultEntitlement)

	// Only update if there's a change
	if difference == 0 {
		return nil
	}

	// Update all leave balances for this leave type in current year
	// Update opening: add difference to current opening
	// Update closing: add difference to current closing (maintains available balance)
	// This ensures:
	// - If opening was equal to old default, it becomes equal to new default
	// - Available balance (closing) is adjusted proportionally
	query := `
		UPDATE Tbl_Leave_balance
		SET opening = opening + $1,
		    closing = closing + $1,
		    updated_at = NOW()
		WHERE leave_type_id = $2 
		AND year = $3
	`

	result, err := tx.Exec(query, difference, leaveTypeID, currentYear)
	if err != nil {
		return err
	}

	// Log how many balances were updated (optional, for debugging)
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Updated %d leave balances for leave_type_id=%d (entitlement: %d → %d, year: %d)\n",
			rowsAffected, leaveTypeID, oldDefaultEntitlement, newDefaultEntitlement, currentYear)
	}

	return nil
}

func (r *Repository) UpdatePendingLeave(tx *sqlx.Tx, leaveID uuid.UUID, empID uuid.UUID, input models.LeaveUpdateInput, NewDays float64) error {

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
