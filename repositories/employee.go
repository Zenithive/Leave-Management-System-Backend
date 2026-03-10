package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

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

// GetAllEmployees returns list of active employees with optional role/designation filters.
// HR: salary not selected (nil in response). ADMIN/SUPER_ADMIN: salary included.
// Single query with JOINs for manager_name and designation_name (no N+1).
func (r *Repository) GetAllEmployees(roleFilter, designationFilter, role string) ([]models.EmployeeResponse, error) {
	salaryCol := "NULL::double precision AS salary"
	if role == constant.ROLE_ADMIN || role == constant.ROLE_SUPER_ADMIN {
		salaryCol = "e.salary"
	}
	query := fmt.Sprintf(`
        SELECT 
            e.id, e.full_name, e.email, e.status,
            r.type AS role, e.manager_id, e.designation_id,
            %s, e.joining_date, e.ending_date,
            e.created_at, e.updated_at,
            m.full_name AS manager_name,
            d.designation_name
        FROM Tbl_Employee e
        JOIN Tbl_Role r ON e.role_id = r.id
        LEFT JOIN Tbl_Employee m ON e.manager_id = m.id
        LEFT JOIN Tbl_Designation d ON e.designation_id = d.id
        WHERE e.status = 'active'
    `, salaryCol)

	args := []interface{}{}
	argCount := 1
	if roleFilter != "" {
		query += fmt.Sprintf(" AND r.type = $%d", argCount)
		args = append(args, roleFilter)
		argCount++
	}
	if designationFilter != "" {
		query += fmt.Sprintf(" AND d.designation_name = $%d", argCount)
		args = append(args, designationFilter)
		argCount++
	}
	query += " ORDER BY e.full_name"

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []models.EmployeeResponse
	for rows.Next() {
		var emp models.EmployeeResponse
		err := rows.Scan(
			&emp.ID,
			&emp.FullName,
			&emp.Email,
			&emp.Status,
			&emp.Role,
			&emp.ManagerID,
			&emp.DesignationID,
			&emp.Salary,
			&emp.JoiningDate,
			&emp.EndingDate,
			&emp.CreatedAt,
			&emp.UpdatedAt,
			&emp.ManagerName,
			&emp.DesignationName,
		)
		if err != nil {
			return nil, err
		}
		employees = append(employees, emp)
	}
	return employees, nil
}
