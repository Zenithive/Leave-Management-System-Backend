package repositories

import (
	"database/sql"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
)

// GetAllLeaveTypesWithEntitlements fetches all non-early leave types with their default entitlements.
// Early leave types (is_early = true) are excluded because they don't have a balance bucket.
func (r *Repository) GetAllLeaveTypesWithEntitlements() ([]models.LeaveTypeData, error) {
	var leaveTypes []models.LeaveTypeData
	query := `
		SELECT 
			lt.id AS leave_type_id,
			lt.name AS leave_type_name,
			COALESCE(lt.default_entitlement, 0) AS default_entitlement,
			lt.intern_entitlement
		FROM Tbl_Leave_Type lt
		WHERE lt.is_early IS NULL OR lt.is_early = FALSE
		ORDER BY lt.id
	`
	err := r.DB.Select(&leaveTypes, query)
	return leaveTypes, err
}

// GetLeaveBalancesByEmployeeAndYear fetches leave balances for a specific employee and year
func (r *Repository) GetLeaveBalancesByEmployeeAndYear(employeeID uuid.UUID, year int) ([]models.BalanceData, error) {
	var balanceRecords []models.BalanceData
	query := `
		SELECT 
			leave_type_id,
			COALESCE(opening, 0) AS opening,
			COALESCE(accrued, 0) AS accrued,
			COALESCE(used, 0) AS used,
			COALESCE(adjusted, 0) AS adjusted,
			COALESCE(closing, 0) AS closing
		FROM Tbl_Leave_balance
		WHERE employee_id = $1 AND year = $2
	`
	err := r.DB.Select(&balanceRecords, query, employeeID, year)

	return balanceRecords, err
}

// GetLeaveBalanceForAdjustment fetches leave balance for adjustment with FOR UPDATE lock
func (r *Repository) GetLeaveBalanceForAdjustment(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, year int) (models.LeaveBalanceForAdjustment, error) {
	var balance models.LeaveBalanceForAdjustment
	query := `
		SELECT 
			id,
			opening,
			accrued,
			used,
			adjusted,
			closing,
			employee_id,
			leave_type_id,
			year
		FROM Tbl_Leave_balance
		WHERE employee_id=$1 AND leave_type_id=$2 AND year=$3
		FOR UPDATE
	`
	err := tx.Get(&balance, query, employeeID, leaveTypeID, year)
	return balance, err
}

// GetDefaultEntitlementByLeaveTypeID fetches default entitlement for a leave type.
// If role is INTERN and intern_entitlement is set, it returns that instead.
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

// GetTotalPaidLeaveBalance returns the sum of all closing balances for paid, non-early leave types
// for the given employee in the current year. This is used to validate unpaid leave applications.
func (r *Repository) GetTotalPaidLeaveBalance(tx *sqlx.Tx, employeeID uuid.UUID) (float64, error) {
	var totalBalance float64
	err := tx.Get(&totalBalance, `
		SELECT COALESCE(SUM(lb.closing), 0)
		FROM Tbl_Leave_balance lb
		JOIN Tbl_Leave_Type lt ON lb.leave_type_id = lt.id
		WHERE lb.employee_id = $1
		  AND lb.year = EXTRACT(YEAR FROM CURRENT_DATE)
		  AND lt.is_paid = TRUE
		  AND (lt.is_early IS NULL OR lt.is_early = FALSE)
	`, employeeID)
	return totalBalance, err
}

// GetTotalPendingPaidLeaveDays returns the sum of all pending/manager-approved leave days
// for paid, non-early leave types for the given employee in the current year.
// This is used to validate unpaid leave applications - employees with pending paid leaves
// should not be allowed to apply for unpaid leave.
func (r *Repository) GetTotalPendingPaidLeaveDays(tx *sqlx.Tx, employeeID uuid.UUID) (float64, error) {
	var totalPendingDays float64
	err := tx.Get(&totalPendingDays, `
		SELECT COALESCE(SUM(l.days), 0)
		FROM Tbl_Leave l
		JOIN Tbl_Leave_Type lt ON l.leave_type_id = lt.id
		WHERE l.employee_id = $1
		  AND l.status IN ('Pending', 'MANAGER_APPROVED')
		  AND EXTRACT(YEAR FROM l.start_date) = EXTRACT(YEAR FROM CURRENT_DATE)
		  AND lt.is_paid = TRUE
		  AND (lt.is_early IS NULL OR lt.is_early = FALSE)
	`, employeeID)
	return totalPendingDays, err
}

// CreateLeaveBalanceForAdjustment creates a new leave balance record
func (r *Repository) CreateLeaveBalanceForAdjustment(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, year int, defaultEntitlement float64) (models.LeaveBalanceForAdjustment, error) {
	var balance models.LeaveBalanceForAdjustment
	err := tx.QueryRow(`
		INSERT INTO Tbl_Leave_balance
		(employee_id, leave_type_id, year, opening, accrued, used, adjusted, closing, created_at, updated_at)
		VALUES ($1,$2,$3,$4,0,0,0,$4,NOW(),NOW())
		RETURNING id, opening, accrued, used, adjusted, closing, employee_id, leave_type_id, year
	`, employeeID, leaveTypeID, year, defaultEntitlement).
		Scan(&balance.ID, &balance.Opening, &balance.Accrued, &balance.Used, &balance.Adjusted, &balance.Closing, &balance.EmployeeID, &balance.LeaveTypeID, &balance.Year)
	return balance, err
}

// UpdateLeaveBalanceAdjustment updates adjusted and closing values for leave balance
func (r *Repository) UpdateLeaveBalanceAdjustment(tx *sqlx.Tx, balanceID uuid.UUID, newAdjusted, newClosing float64) error {
	query := `
		UPDATE Tbl_Leave_balance
		SET adjusted=$1, closing=$2, updated_at=NOW()
		WHERE id=$3
	`
	_, err := tx.Exec(query, newAdjusted, newClosing, balanceID)
	return err
}

// InsertLeaveAdjustment inserts a record into leave adjustment log
func (r *Repository) InsertLeaveAdjustment(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int, quantity float64, reason string, createdBy string, year int) error {
	query := `
		INSERT INTO Tbl_Leave_adjustment
		(employee_id, leave_type_id, quantity, reason, created_by, created_at, year)
		VALUES ($1,$2,$3,$4,$5,NOW(),$6)
	`
	_, err := tx.Exec(query, employeeID, leaveTypeID, quantity, reason, createdBy, year)
	return err
}

func (r *Repository) UpdateLeaveBalanceByEmployeeId(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeId int, Days float64) error {
	query := `UPDATE Tbl_Leave_balance SET used = used + $3, closing = closing - $3, updated_at = NOW() WHERE employee_id=$1 AND leave_type_id=$2 AND year = EXTRACT(YEAR FROM CURRENT_DATE)`
	_, err := tx.Exec(query, employeeID, leaveTypeId, Days)
	return err
}
func (r *Repository) UpdateWidthrowLeaveBalanceByEmployeeId(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeId int, Days float64) error {
	query := `UPDATE Tbl_Leave_balance SET used = used - $3, closing = closing + $3, updated_at = NOW() WHERE employee_id=$1 AND leave_type_id=$2 AND year = EXTRACT(YEAR FROM CURRENT_DATE)`
	_, err := tx.Exec(query, employeeID, leaveTypeId, Days)
	return err
}

// UpdateInternLeaveBalancesForEntitlementChange recalculates leave balances for INTERN employees
// when intern_entitlement changes for a leave type.
//
// newInternEntitlement is the entitlement INTERNs should now have (already resolved by the caller:
// if intern_entitlement is being cleared, the caller passes newDefaultEntitlement as the new value).
//
// For each INTERN employee with a balance row:
//   - If joined in the current year: new opening = prorated(newInternEntitlement, joinMonth)
//   - Otherwise: new opening = newInternEntitlement
//   - closing = new_opening - used + adjusted
func (r *Repository) UpdateInternLeaveBalancesForEntitlementChange(tx *sqlx.Tx, leaveTypeID int, newInternEntitlement int, currentYear int) error {
	type empRow struct {
		ID          uuid.UUID  `db:"id"`
		JoiningDate *time.Time `db:"joining_date"`
	}
	var employees []empRow
	err := tx.Select(&employees, `SELECT e.id, e.joining_date
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		WHERE  r.type  = 'INTERN'
	`)
	for _, emp := range employees {
		var newOpening int
		if emp.JoiningDate != nil && emp.JoiningDate.Year() == currentYear {
			newOpening = proratedLeave(newInternEntitlement, int(emp.JoiningDate.Month()))
		} else {
			newOpening = newInternEntitlement
		}
		_, err := r.GetLeaveBalance(tx, emp.ID, leaveTypeID)
		if err == sql.ErrNoRows {
			// Create balance if it doesn't exist
			if err := r.CreateLeaveBalance(tx, emp.ID, leaveTypeID, newOpening); err != nil {
				return err
			}
		} else {
			err = r.UpdateLeaveBalance(tx, newOpening, emp.ID, leaveTypeID, currentYear)
		}
	}
	return err
}

// UpdateLeaveBalancesForEntitlementChange recalculates leave balances for all non-INTERN employees
// when default_entitlement changes for a leave type.
//
// For each affected employee:
//   - If joined in a prior year (or no joining_date): new opening = newDefaultEntitlement
//   - If joined in the current year: new opening = prorated(newDefaultEntitlement, joinMonth)
//   - closing = new_opening - used + adjusted
//
// This is a full recalculation from the new entitlement, not a diff-based patch,
// so it is safe to call multiple times and always produces a consistent result.
func (r *Repository) UpdateLeaveBalancesForEntitlementChange(tx *sqlx.Tx, leaveTypeID int, oldDefaultEntitlement, newDefaultEntitlement int, currentYear int) error {

	type empRow struct {
		ID          uuid.UUID  `db:"id"`
		JoiningDate *time.Time `db:"joining_date"`
	}
	var employees []empRow
	err := tx.Select(&employees, `
		SELECT e.id, e.joining_date
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		WHERE  r.type != 'INTERN'
	`)

	for _, emp := range employees {
		var newOpening int
		if emp.JoiningDate != nil && emp.JoiningDate.Year() == currentYear {
			newOpening = proratedLeave(newDefaultEntitlement, int(emp.JoiningDate.Month()))
		} else {
			newOpening = newDefaultEntitlement
		}
		_, err := r.GetLeaveBalance(tx, emp.ID, leaveTypeID)
		if err == sql.ErrNoRows {
			// Create balance if it doesn't exist
			if err := r.CreateLeaveBalance(tx, emp.ID, leaveTypeID, newOpening); err != nil {
				return err
			}
		} else {
			err = r.UpdateLeaveBalance(tx, newOpening, emp.ID, leaveTypeID, currentYear)
		}
	}
	return err
}

// AdjustLeaveBalancesForRoleChange recalculates all leave balances for an employee
// when their role changes (any role change, not just INTERN boundary).
//
// For every non-early leave type the employee has a balance row for:
//   - Resolve the correct full entitlement for the NEW role
//     (intern_entitlement if newRole==INTERN and it is set, otherwise default_entitlement)
//   - If employee joined in the current year: new opening = prorated(entitlement, joinMonth)
//   - Otherwise: new opening = entitlement
//   - closing = new_opening - used + adjusted
func (r *Repository) AdjustLeaveBalancesForRoleChange(tx *sqlx.Tx, employeeID uuid.UUID, oldRole, newRole string, currentYear int) error {
	// No leave balance change when neither side is INTERN
	if oldRole != "INTERN" && newRole != "INTERN" {
		return nil
	}
	var joiningDate *time.Time
	_ = tx.Get(&joiningDate, `SELECT joining_date FROM Tbl_Employee WHERE id = $1`, employeeID)
	isJoiningThisYear := joiningDate != nil && joiningDate.Year() == currentYear

	leaveTypes, err := r.GetAllLeaveTypes(tx)

	for _, lt := range leaveTypes {
		// Resolve entitlement for the new role
		newEntitlement := lt.DefaultEntitlement
		if newRole == "INTERN" && lt.InternEntitlement != nil {
			newEntitlement = *lt.InternEntitlement
		}

		var newOpening int
		if isJoiningThisYear {
			newOpening = proratedLeave(newEntitlement, int(joiningDate.Month()))
		} else {
			newOpening = newEntitlement
		}

		err = r.UpdateLeaveBalance(tx, newOpening, employeeID, lt.ID, currentYear)
	}
	return err
}

// BulkAllocateLeaveBalanceForNewLeaveType allocates a leave balance row for every active employee
// when a new leave type is created. Skips employees who already have a row (ON CONFLICT DO NOTHING).
// For INTERN employees, intern_entitlement is used if set; otherwise default_entitlement is used.
// Employees who joined in the current year get a prorated entitlement based on their joining month.
func (r *Repository) BulkAllocateLeaveBalanceForNewLeaveType(tx *sqlx.Tx, leaveTypeID int, defaultEntitlement int, internEntitlement *int, employees []ActiveEmployeeRole) error {
	currentYear := time.Now().Year()

	for _, emp := range employees {
		entitlement := defaultEntitlement
		if emp.Role == "INTERN" && internEntitlement != nil {
			entitlement = *internEntitlement
		}
		// Prorate if the employee joined in the current year
		if emp.JoiningDate != nil && emp.JoiningDate.Year() == currentYear {
			entitlement = proratedLeave(entitlement, int(emp.JoiningDate.Month()))
		}
		if err := r.CreateLeaveBalance(tx, emp.ID, leaveTypeID, entitlement); err != nil {
			return err
		}
	}
	return nil
}

// RecalculateLeaveBalancesForJoiningDateChange recalculates opening and closing for all
// current-year leave balances of an employee when their joining_date changes.
//
// Logic:
//   - If new joining year == current year → prorate opening by new joining month
//   - If new joining year != current year → restore full entitlement as opening
//
// closing is recalculated as: new_opening - used + adjusted
func (r *Repository) RecalculateLeaveBalancesForJoiningDateChange(tx *sqlx.Tx, employeeID uuid.UUID, newJoiningDate *time.Time, empRole string, currentYear int) error {
	// Fetch all leave types (non-early) with entitlements
	leaveTypes, err := r.GetAllLeaveTypes(tx)

	isJoiningThisYear := newJoiningDate != nil && newJoiningDate.Year() == currentYear
	for _, lt := range leaveTypes {
		// Pick the correct full entitlement for this employee's role
		fullEntitlement := lt.DefaultEntitlement
		if empRole == "INTERN" && lt.InternEntitlement != nil {
			fullEntitlement = *lt.InternEntitlement
		}
		var newOpening int
		if isJoiningThisYear {
			newOpening = proratedLeave(fullEntitlement, int(newJoiningDate.Month()))
		} else {
			newOpening = fullEntitlement
		}
		err = r.UpdateLeaveBalance(tx, newOpening, employeeID, lt.ID, currentYear)
	}
	return err
}

func (r *Repository) UpdateLeaveBalance(tx *sqlx.Tx, newOpening int, employeeID uuid.UUID, ID int, currentYear int) error {
	_, err := tx.Exec(`
			UPDATE Tbl_Leave_balance SET opening    = $1,  closing    = $1 - used + adjusted, updated_at = NOW()
			WHERE employee_id   = $2
			  AND leave_type_id = $3
			  AND year          = $4
		`, newOpening, employeeID, ID, currentYear)
	return err
}

// proratedLeave calculates floor((yearlyLeave * remainingMonths) / 12).
// remainingMonths = 12 - joinMonth + 1 (includes the joining month itself).
func proratedLeave(yearlyLeave int, joinMonth int) int {
	if joinMonth < 1 || joinMonth > 12 {
		return yearlyLeave
	}
	remainingMonths := 12 - joinMonth + 1
	return int(math.Floor(float64(yearlyLeave) * float64(remainingMonths) / 12))
}
