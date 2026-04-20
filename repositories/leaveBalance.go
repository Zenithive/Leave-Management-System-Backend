package repositories

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

// GetAllLeaveTypesWithEntitlements fetches all leave types with their default entitlements
func (r *Repository) GetAllLeaveTypesWithEntitlements() ([]models.LeaveTypeData, error) {
	var leaveTypes []models.LeaveTypeData
	query := `
		SELECT 
			lt.id AS leave_type_id,
			lt.name AS leave_type_name,
			COALESCE(lt.default_entitlement, 0) AS default_entitlement
		FROM Tbl_Leave_Type lt
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

// GetDefaultEntitlementByLeaveTypeID fetches default entitlement for a leave type
func (r *Repository) GetDefaultEntitlementByLeaveTypeID(tx *sqlx.Tx, leaveTypeID int) (float64, error) {
	var defaultEntitlement float64
	err := tx.Get(&defaultEntitlement, `SELECT default_entitlement FROM Tbl_Leave_Type WHERE id=$1`, leaveTypeID)
	return defaultEntitlement, err
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
	query := `UPDATE Tbl_Leave_balance SET used = used + $3, closing = closing - $3, updated_at = NOW() WHERE employee_id=$1 AND leave_type_id=$2`
	_, err := tx.Exec(query, employeeID, leaveTypeId, Days)
	return err
}
func (r *Repository) UpdateWidthrowLeaveBalanceByEmployeeId(tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeId int, Days float64) error {
	query := `UPDATE Tbl_Leave_balance SET used = used - $3, closing = closing + $3, updated_at = NOW() WHERE employee_id=$1 AND leave_type_id=$2`
	_, err := tx.Exec(query, employeeID, leaveTypeId, Days)
	return err
}
