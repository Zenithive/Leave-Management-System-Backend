package repositories

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AccruableLeaveType holds the leave types that should receive monthly accrual.
// Only paid, non-early leave types are accrued.
type AccruableLeaveType struct {
	ID int `db:"id"`
}

// AccruableEmployee holds the minimal data needed to run accrual for one employee.
type AccruableEmployee struct {
	ID   uuid.UUID `db:"id"`
	Role string    `db:"role"`
}

// GetAccruableLeaveTypes returns all paid, non-early leave types.
// These are the only types that receive monthly accrual credits.
func (r *Repository) GetAccruableLeaveTypes() ([]AccruableLeaveType, error) {
	var types []AccruableLeaveType
	err := r.DB.Select(&types, `
		SELECT id
		FROM Tbl_Leave_type
		WHERE is_paid = TRUE
		  AND (is_early IS NULL OR is_early = FALSE)
		ORDER BY id
	`)
	return types, err
}

// GetAllActiveEmployeesForAccrual returns id and role for every active employee.
func (r *Repository) GetAllActiveEmployeesForAccrual() ([]AccruableEmployee, error) {
	var employees []AccruableEmployee
	err := r.DB.Select(&employees, `
		SELECT e.id, r.type AS role
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		WHERE e.status = 'active'
	`)
	return employees, err
}

// IsAccrualAlreadyRun returns true if accrual has already been credited for
// this employee + leave_type + month + year combination.
func (r *Repository) IsAccrualAlreadyRun(
	tx *sqlx.Tx,
	employeeID uuid.UUID,
	leaveTypeID int,
	month, year int,
) (bool, error) {
	var count int
	err := tx.Get(&count, `
		SELECT COUNT(*)
		FROM Tbl_Leave_accrual_log
		WHERE employee_id   = $1
		  AND leave_type_id = $2
		  AND month         = $3
		  AND year          = $4
	`, employeeID, leaveTypeID, month, year)
	return count > 0, err
}

// CreditMonthlyAccrual adds `days` to the accrued and closing columns of the
// leave balance row for the given employee + leave_type + year, then records
// the accrual in the log table so it cannot be double-credited.
//
// If no balance row exists yet for this year (e.g. employee joined mid-year),
// it creates one with opening = 0 before crediting.
func (r *Repository) CreditMonthlyAccrual(
	tx *sqlx.Tx,
	employeeID uuid.UUID,
	leaveTypeID int,
	year int,
	month int,
	days float64,
) error {
	// Upsert the balance row — creates it if missing, leaves it alone if present.
	_, err := tx.Exec(`
		INSERT INTO Tbl_Leave_balance
			(employee_id, leave_type_id, year, opening, accrued, used, adjusted, closing)
		VALUES ($1, $2, $3, 0, 0, 0, 0, 0)
		ON CONFLICT (employee_id, leave_type_id, year) DO NOTHING
	`, employeeID, leaveTypeID, year)
	if err != nil {
		return err
	}

	// Add the accrual to accrued and closing.
	_, err = tx.Exec(`
		UPDATE Tbl_Leave_balance
		SET accrued    = accrued + $1,
		    closing    = closing + $1,
		    updated_at = NOW()
		WHERE employee_id   = $2
		  AND leave_type_id = $3
		  AND year          = $4
	`, days, employeeID, leaveTypeID, year)
	if err != nil {
		return err
	}

	// Record in the log so this month is never credited again.
	_, err = tx.Exec(`
		INSERT INTO Tbl_Leave_accrual_log
			(employee_id, leave_type_id, month, year, days_credited)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT ON CONSTRAINT uq_accrual_log DO NOTHING
	`, employeeID, leaveTypeID, month, year, days)
	return err
}
