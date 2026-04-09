package repositories

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

func (r *Repository) AddLeaveType(tx *sqlx.Tx, input models.LeaveTypeInput) (models.LeaveType, error) {
	var leave models.LeaveType
	query := `
		INSERT INTO Tbl_Leave_type (name, is_paid, default_entitlement , is_early)
		VALUES ($1, $2, $3 , $4)
		RETURNING id, created_at, updated_at, is_early
	`
	err := tx.QueryRow(query, input.Name, *input.IsPaid, *input.DefaultEntitlement, *input.IsEarly).
		Scan(&leave.ID, &leave.CreatedAt, &leave.UpdatedAt, &leave.IsEarly)
	return leave, err
}

// UpdateLeaveType - Update leave policy
func (r *Repository) UpdateLeaveType(tx *sqlx.Tx, leaveTypeID int, input models.LeaveTypeInput) error {
	query := `
		UPDATE Tbl_Leave_type 
		SET name = $1, is_paid = $2, default_entitlement = $3, updated_at = NOW()
		WHERE id = $4
	`
	result, err := tx.Exec(query, input.Name, *input.IsPaid, *input.DefaultEntitlement, leaveTypeID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// DeleteLeaveType - Delete leave policy
func (r *Repository) DeleteLeaveType(tx *sqlx.Tx, leaveTypeID int) error {
	// Check if leave type is being used in any leave applications
	var count int
	err := tx.Get(&count, "SELECT COUNT(*) FROM Tbl_Leave WHERE leave_type_id = $1", leaveTypeID)
	if err != nil {
		return err
	}

	if count > 0 {
		return sql.ErrNoRows // Using this to indicate constraint violation
	}

	query := `DELETE FROM Tbl_Leave_type WHERE id = $1`
	result, err := tx.Exec(query, leaveTypeID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repository) GetAllLeaveType() ([]models.LeaveType, error) {
	var leaveType []models.LeaveType
	query := `SELECT id, name, is_paid, default_entitlement, is_early ,created_at, updated_at FROM Tbl_Leave_type ORDER BY id`
	err := r.DB.Select(&leaveType, query)
	return leaveType, err
}
