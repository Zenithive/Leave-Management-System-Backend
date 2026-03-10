package repositories

import "github.com/google/uuid"

// 1. Get employee status
func (r *Repository) GetEmployeeStatus(employeeID uuid.UUID) (string, error) {
	var status string
	err := r.DB.Get(&status, `
		SELECT status FROM Tbl_Employee WHERE id=$1
	`, employeeID)
	return status, err
}

// ------------------ UPDATE EMPLOYEE DESIGNATION ------------------
func (r *Repository) UpdateEmployeeDesignation(empID uuid.UUID, designationID *uuid.UUID) error {
	query := `
		UPDATE Tbl_Employee
		SET designation_id = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.DB.Exec(query, designationID, empID)
	return err
}
