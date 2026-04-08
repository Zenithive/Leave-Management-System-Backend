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

// GetAllEmployees returns list of employees with advanced filtering, sorting, and pagination.
// HR: salary not selected (nil in response). ADMIN/SUPER_ADMIN: salary included.
// Single query with JOINs for manager_name and designation_name (no N+1).
func (r *Repository) GetAllEmployees(params models.EmployeeFilterParams, role string) (*models.PaginatedEmployeeResponse, error) {
	// Salary column based on role
	salaryCol := "NULL::double precision AS salary"
	if role == constant.ROLE_ADMIN || role == constant.ROLE_SUPER_ADMIN {
		salaryCol = "e.salary"
	}

	// Build WHERE clause dynamically
	whereConditions := []string{}
	args := []interface{}{}
	argCount := 1

	// Status filter (optional - can filter active/inactive or show all)
	if params.Status != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("e.status = $%d", argCount))
		args = append(args, params.Status)
		argCount++
	}

	// Name filter (partial match, case-insensitive)
	if params.Name != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("e.full_name ILIKE $%d", argCount))
		args = append(args, "%"+params.Name+"%")
		argCount++
	}

	// Email filter (partial match, case-insensitive)
	if params.Email != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("e.email ILIKE $%d", argCount))
		args = append(args, "%"+params.Email+"%")
		argCount++
	}

	// Role filter (exact match)
	if params.Role != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("r.type = $%d", argCount))
		args = append(args, params.Role)
		argCount++
	}

	// Designation filter (exact match)
	if params.Designation != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("d.designation_name = $%d", argCount))
		args = append(args, params.Designation)
		argCount++
	}

	// Manager name filter (partial match, case-insensitive)
	if params.ManagerName != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("m.full_name ILIKE $%d", argCount))
		args = append(args, "%"+params.ManagerName+"%")
		argCount++
	}

	// Build WHERE clause
	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + whereConditions[0]
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + whereConditions[i]
		}
	}

	// Count total records (for pagination metadata)
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		LEFT JOIN Tbl_Employee m ON e.manager_id = m.id
		LEFT JOIN Tbl_Designation d ON e.designation_id = d.id
		%s
	`, whereClause)

	var totalCount int
	err := r.DB.Get(&totalCount, countQuery, args...)
	if err != nil {
		return nil, err
	}

	// Build ORDER BY clause
	orderBy := "e.full_name ASC" // default
	sortMap := map[string]string{
		"name":         "e.full_name",
		"email":        "e.email",
		"joining_date": "e.joining_date",
		"salary":       "e.salary",
		"manager_name": "m.full_name",
		"role":         "r.type",
		"status":       "e.status",
	}

	if params.SortBy != "" {
		if dbColumn, ok := sortMap[params.SortBy]; ok {
			sortOrder := "ASC"
			if params.SortOrder == "desc" {
				sortOrder = "DESC"
			}
			orderBy = fmt.Sprintf("%s %s", dbColumn, sortOrder)
		}
	}

	// Pagination defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 10 // default page size
	}
	if params.PageSize > 100 {
		params.PageSize = 100 // max page size
	}

	offset := (params.Page - 1) * params.PageSize

	// Main query with pagination
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
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, salaryCol, whereClause, orderBy, argCount, argCount+1)

	args = append(args, params.PageSize, offset)

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

	// Calculate total pages
	totalPages := (totalCount + params.PageSize - 1) / params.PageSize

	return &models.PaginatedEmployeeResponse{
		Employees:  employees,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}
