package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

// ─────────────────────────────────────────────
// Read
// ─────────────────────────────────────────────

func (r *Repository) GetAllLeaveTypesWithEntitlements() ([]models.LeaveTypeData, error) {
	var leaveTypes []models.LeaveTypeData
	err := r.DB.Select(&leaveTypes, `
		SELECT
			lt.id AS leave_type_id,
			lt.name AS leave_type_name,
			COALESCE(lt.default_entitlement, 0) AS default_entitlement,
			lt.intern_entitlement
		FROM Tbl_Leave_Type lt
		WHERE lt.is_early IS NULL OR lt.is_early = FALSE
		ORDER BY lt.id
	`)
	return leaveTypes, err
}

func (r *Repository) GetLeaveBalancesByEmployeeAndYear(employeeID uuid.UUID, year int) ([]models.BalanceData, error) {
	var balanceRecords []models.BalanceData
	err := r.DB.Select(&balanceRecords, `
		SELECT
			leave_type_id,
			COALESCE(opening, 0)  AS opening,
			COALESCE(accrued, 0)  AS accrued,
			COALESCE(used, 0)     AS used,
			COALESCE(adjusted, 0) AS adjusted,
			COALESCE(closing, 0)  AS closing
		FROM Tbl_Leave_balance
		WHERE employee_id = $1 AND year = $2
	`, employeeID, year)
	return balanceRecords, err
}

func (r *Repository) GetLeaveBalanceForAdjustment(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, year int) (models.LeaveBalanceForAdjustment, error) {
	var balance models.LeaveBalanceForAdjustment
	err := tx.Get(&balance, `
		SELECT id, opening, accrued, used, adjusted, closing, employee_id, leave_type_id, year
		FROM Tbl_Leave_balance
		WHERE employee_id=$1 AND leave_type_id=$2 AND year=$3
		FOR UPDATE
	`, employeeID, leaveTypeID, year)
	return balance, err
}

// GetDefaultEntitlementByLeaveTypeID returns the raw default and intern entitlement
// for a leave type. Role resolution is the caller's responsibility.
func (r *Repository) GetDefaultEntitlementByLeaveTypeID(tx *sqlx.Tx, leaveTypeID int, role string) (float64, error) {
	var row struct {
		DefaultEntitlement float64  `db:"default_entitlement"`
		InternEntitlement  *float64 `db:"intern_entitlement"`
	}
	err := tx.Get(&row, `SELECT default_entitlement, intern_entitlement FROM Tbl_Leave_Type WHERE id=$1`, leaveTypeID)
	if err != nil {
		return 0, err
	}
	if role == "INTERN" && row.InternEntitlement != nil {
		return *row.InternEntitlement, nil
	}
	return row.DefaultEntitlement, nil
}

// GetEmployeesByRole returns id and joining_date for all employees matching the given role filter.
// roleFilter: "INTERN" returns only interns; "NON_INTERN" returns everyone else.
func (r *Repository) GetEmployeesByRole(tx *sqlx.Tx, internOnly bool) ([]EmployeeJoinRow, error) {
	var employees []EmployeeJoinRow
	var query string
	if internOnly {
		query = `
			SELECT e.id, e.joining_date, r.type AS role
			FROM Tbl_Employee e
			JOIN Tbl_Role r ON e.role_id = r.id
			WHERE r.type = 'INTERN'
		`
	} else {
		query = `
			SELECT e.id, e.joining_date, r.type AS role
			FROM Tbl_Employee e
			JOIN Tbl_Role r ON e.role_id = r.id
			WHERE r.type != 'INTERN'
		`
	}
	err := tx.Select(&employees, query)
	return employees, err
}

// EmployeeJoinRow is the minimal employee data needed for leave balance calculations.
type EmployeeJoinRow struct {
	ID          uuid.UUID  `db:"id"`
	JoiningDate *time.Time `db:"joining_date"`
	Role        string     `db:"role"`
}

// ─────────────────────────────────────────────
// Write
// ─────────────────────────────────────────────

func (r *Repository) CreateLeaveBalanceForAdjustment(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, year int, defaultEntitlement float64) (models.LeaveBalanceForAdjustment, error) {
	var balance models.LeaveBalanceForAdjustment
	err := tx.QueryRow(`
		INSERT INTO Tbl_Leave_balance
		(employee_id, leave_type_id, year, opening, accrued, used, adjusted, closing, created_at, updated_at)
		VALUES ($1,$2,$3,$4,0,0,0,$4,NOW(),NOW())
		RETURNING id, opening, accrued, used, adjusted, closing, employee_id, leave_type_id, year
	`, employeeID, leaveTypeID, year, defaultEntitlement).
		Scan(&balance.ID, &balance.Opening, &balance.Accrued, &balance.Used, &balance.Adjusted,
			&balance.Closing, &balance.EmployeeID, &balance.LeaveTypeID, &balance.Year)
	return balance, err
}

func (r *Repository) UpdateLeaveBalanceAdjustment(tx *sqlx.Tx, balanceID uuid.UUID, newAdjusted, newClosing float64) error {
	_, err := tx.Exec(`
		UPDATE Tbl_Leave_balance
		SET adjusted=$1, closing=$2, updated_at=NOW()
		WHERE id=$3
	`, newAdjusted, newClosing, balanceID)
	return err
}

func (r *Repository) InsertLeaveAdjustment(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, quantity float64, reason string, createdBy string, year int) error {
	_, err := tx.Exec(`
		INSERT INTO Tbl_Leave_adjustment
		(employee_id, leave_type_id, quantity, reason, created_by, created_at, year)
		VALUES ($1,$2,$3,$4,$5,NOW(),$6)
	`, employeeID, leaveTypeID, quantity, reason, createdBy, year)
	return err
}

func (r *Repository) UpdateLeaveBalanceByEmployeeId(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeId int, days float64) error {
	_, err := tx.Exec(`
		UPDATE Tbl_Leave_balance
		SET used = used + $3, closing = closing - $3, updated_at = NOW()
		WHERE employee_id=$1 AND leave_type_id=$2 AND year = EXTRACT(YEAR FROM CURRENT_DATE)
	`, employeeID, leaveTypeId, days)
	return err
}

func (r *Repository) UpdateWidthrowLeaveBalanceByEmployeeId(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeId int, days float64) error {
	_, err := tx.Exec(`
		UPDATE Tbl_Leave_balance
		SET used = used - $3, closing = closing + $3, updated_at = NOW()
		WHERE employee_id=$1 AND leave_type_id=$2 AND year = EXTRACT(YEAR FROM CURRENT_DATE)
	`, employeeID, leaveTypeId, days)
	return err
}

// UpdateLeaveBalance sets a new opening value and recalculates closing for a single balance row.
func (r *Repository) UpdateLeaveBalance(tx *sqlx.Tx, newOpening int, employeeID uuid.UUID, leaveTypeID int, year int) error {
	_, err := tx.Exec(`
		UPDATE Tbl_Leave_balance
		SET opening = $1, closing = $1 + accrued - used + adjusted, updated_at = NOW()
		WHERE employee_id=$2 AND leave_type_id=$3 AND year=$4
	`, newOpening, employeeID, leaveTypeID, year)
	return err
}
