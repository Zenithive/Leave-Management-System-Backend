package repositories

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
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

// UpdateInternLeaveBalancesForEntitlementChange updates leave balances for INTERN employees
// when intern_entitlement changes for a leave type.
// Recalculates closing correctly as: (opening + diff) + accrued - used + adjusted
func (r *Repository) UpdateInternLeaveBalancesForEntitlementChange(tx *sqlx.Tx, leaveTypeID int, oldInternEntitlement, newInternEntitlement int, currentYear int) error {
	difference := float64(newInternEntitlement - oldInternEntitlement)
	if difference == 0 {
		return nil
	}

	query := `
		UPDATE Tbl_Leave_balance lb
		SET opening    = lb.opening + $1,
		    closing    = (lb.opening + $1) + lb.accrued - lb.used + lb.adjusted,
		    updated_at = NOW()
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		WHERE lb.employee_id   = e.id
		  AND lb.leave_type_id = $2
		  AND lb.year          = $3
		  AND r.type           = 'INTERN'
	`
	_, err := tx.Exec(query, difference, leaveTypeID, currentYear)
	return err
}

// AdjustLeaveBalancesForRoleChange updates all leave balances for an employee
// when their role changes between INTERN and a non-INTERN role.
// It adjusts opening and closing by the difference between intern_entitlement and default_entitlement
// for each leave type that has an intern_entitlement set.
func (r *Repository) AdjustLeaveBalancesForRoleChange(tx *sqlx.Tx, employeeID uuid.UUID, oldRole, newRole string, currentYear int) error {
	// Only act when crossing the INTERN boundary
	if oldRole != "INTERN" && newRole != "INTERN" {
		return nil
	}

	// Fetch all leave types that have a distinct intern_entitlement
	type leaveTypeRow struct {
		ID                 int `db:"id"`
		DefaultEntitlement int `db:"default_entitlement"`
		InternEntitlement  int `db:"intern_entitlement"`
	}
	var leaveTypes []leaveTypeRow
	err := tx.Select(&leaveTypes, `
		SELECT id, default_entitlement, intern_entitlement
		FROM Tbl_Leave_type
		WHERE intern_entitlement IS NOT NULL
		  AND intern_entitlement != default_entitlement
	`)
	if err != nil {
		return err
	}

	for _, lt := range leaveTypes {
		var diff float64
		if oldRole == "INTERN" && newRole != "INTERN" {
			// Upgrading from INTERN: switch from intern_entitlement → default_entitlement
			diff = float64(lt.DefaultEntitlement - lt.InternEntitlement)
		} else {
			// Downgrading to INTERN: switch from default_entitlement → intern_entitlement
			diff = float64(lt.InternEntitlement - lt.DefaultEntitlement)
		}

		if diff == 0 {
			continue
		}

		// Only update if a balance row already exists for this employee + leave type + year
		_, err := tx.Exec(`
			UPDATE Tbl_Leave_balance
			SET opening  = opening  + $1,
			    closing  = closing  + $1,
			    updated_at = NOW()
			WHERE employee_id  = $2
			  AND leave_type_id = $3
			  AND year          = $4
		`, diff, employeeID, lt.ID, currentYear)
		if err != nil {
			return err
		}
	}

	return nil
}

// BulkAllocateLeaveBalanceForNewLeaveType allocates a leave balance row for every active employee
// when a new leave type is created. Skips employees who already have a row (ON CONFLICT DO NOTHING).
// For INTERN employees, intern_entitlement is used if set; otherwise default_entitlement is used.
func (r *Repository) BulkAllocateLeaveBalanceForNewLeaveType(
	tx *sqlx.Tx,
	leaveTypeID int,
	defaultEntitlement int,
	internEntitlement *int,
	employees []ActiveEmployeeRole,
) error {
	for _, emp := range employees {
		entitlement := defaultEntitlement
		if emp.Role == "INTERN" && internEntitlement != nil {
			entitlement = *internEntitlement
		}
		if err := r.CreateLeaveBalance(tx, emp.ID, leaveTypeID, entitlement); err != nil {
			return err
		}

	}
	return nil
}
