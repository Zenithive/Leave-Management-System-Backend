package repositories

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

// employeeSortMap maps API sort_by keys → safe SQL column expressions.
// Used by both GetAllEmployees and GetEmployeesByManagerID via resolveSortField.
var employeeSortMap = map[string]string{
	"name":         "e.full_name",
	"email":        "e.email",
	"joining_date": "e.joining_date",
	"ending_date":  "COALESCE(e.ending_date, '9999-12-31')",
	"salary":       "e.salary",
	"manager_name": "COALESCE(m.full_name, '')",
	"role":         "r.type",
	"status":       "e.status",
}

// birthdaySortExpr returns a SQL ORDER BY expression that sorts employees by upcoming
// birthday relative to today's (month/day). Ascending = soonest, Descending = furthest.
// NULLs are always placed last regardless of direction.
func birthdaySortExpr(order string) string {
	today := time.Now()
	dateStr := fmt.Sprintf("%04d-%02d-%02d", today.Year(), int(today.Month()), today.Day())

	if order == "desc" {
		return fmt.Sprintf(`
			CASE WHEN e.birth_date IS NULL THEN -1
			ELSE MOD(
				CAST(EXTRACT(DOY FROM e.birth_date) AS INT)
				- CAST(EXTRACT(DOY FROM DATE '%s') AS INT)
				+ 366, 366
			) END DESC`, dateStr)
	}
	return fmt.Sprintf(`
		CASE WHEN e.birth_date IS NULL THEN 999999
		ELSE MOD(
			CAST(EXTRACT(DOY FROM e.birth_date) AS INT)
			- CAST(EXTRACT(DOY FROM DATE '%s') AS INT)
			+ 366, 366
		) END ASC`, dateStr)
}

// resolveEmployeeSort returns the final ORDER BY clause for a given sort_by + sort_order.
// Falls back to "e.full_name ASC" for unknown keys.
func resolveEmployeeSort(sortBy, sortOrder string) string {
	dir := "ASC"
	if sortOrder == "desc" {
		dir = "DESC"
	}

	if sortBy == "birth_date" {
		return birthdaySortExpr(sortOrder)
	}

	col := resolveSortField(employeeSortMap, sortBy, "e.full_name")
	return fmt.Sprintf("%s %s", col, dir)
}

// 1. Get employee status
func (r *Repository) GetEmployeeStatus(employeeID uuid.UUID) (string, error) {
	var status string
	err := r.DB.Get(&status, `SELECT status FROM Tbl_Employee WHERE id=$1`, employeeID)
	return status, err
}

// ------------------ UPDATE EMPLOYEE DESIGNATION ------------------
func (r *Repository) UpdateEmployeeDesignation(empID uuid.UUID, designationID *uuid.UUID) error {
	_, err := r.DB.Exec(`
		UPDATE Tbl_Employee SET designation_id = $1, updated_at = NOW() WHERE id = $2
	`, designationID, empID)
	return err
}

// GetAllEmployees returns a paginated, filtered, sorted list of employees.
// HR: salary is NULL. ADMIN/SUPERADMIN: salary included.
// Filters: search (name/email/manager), role, designation, status, manager (exact).
// Sort: name, email, joining_date, ending_date, salary, birth_date, manager_name, role, status.
func (r *Repository) GetAllEmployees(params models.EmployeeFilterParams, role string) (*models.PaginatedEmployeeResponse, error) {
	salaryCol := "NULL::double precision AS salary"
	if role == constant.ROLE_ADMIN || role == constant.ROLE_SUPER_ADMIN {
		salaryCol = "e.salary"
	}

	// Build WHERE conditions dynamically
	conditions := []string{}
	args := []interface{}{}
	n := 1 // arg counter

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(e.full_name ILIKE $%d OR e.email ILIKE $%d OR m.full_name ILIKE $%d)",
			n, n, n,
		))
		args = append(args, "%"+params.Search+"%")
		n++
	}
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", n))
		args = append(args, params.Status)
		n++
	}
	if params.Role != "" {
		conditions = append(conditions, fmt.Sprintf("r.type = $%d", n))
		args = append(args, params.Role)
		n++
	}
	if params.Designation != "" {
		conditions = append(conditions, fmt.Sprintf("d.designation_name = $%d", n))
		args = append(args, params.Designation)
		n++
	}
	if params.Manager != "" {
		conditions = append(conditions, fmt.Sprintf("m.full_name = $%d", n))
		args = append(args, params.Manager)
		n++
	}

	whereClause := buildWhere(conditions)

	baseJoins := `
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		LEFT JOIN Tbl_Employee m ON e.manager_id = m.id
		LEFT JOIN Tbl_Designation d ON e.designation_id = d.id
	`

	// Count total for pagination
	var totalCount int
	if err := r.DB.Get(&totalCount,
		"SELECT COUNT(*) "+baseJoins+whereClause, args...); err != nil {
		return nil, err
	}

	// Pagination defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	offset := (params.Page - 1) * params.PageSize

	orderBy := resolveEmployeeSort(params.SortBy, params.SortOrder)

	query := fmt.Sprintf(`
		SELECT
			e.id, e.full_name, e.email, e.status,
			r.type AS role, e.manager_id, e.designation_id,
			%s, e.joining_date, e.birth_date, e.ending_date,
			e.created_at, e.updated_at,
			m.full_name AS manager_name,
			d.designation_name
		%s%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, salaryCol, baseJoins, whereClause, orderBy, n, n+1)

	args = append(args, params.PageSize, offset)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	employees := []models.EmployeeResponse{}
	for rows.Next() {
		var emp models.EmployeeResponse
		if err := rows.Scan(
			&emp.ID, &emp.FullName, &emp.Email, &emp.Status,
			&emp.Role, &emp.ManagerID, &emp.DesignationID,
			&emp.Salary, &emp.JoiningDate, &emp.BirthDate, &emp.EndingDate,
			&emp.CreatedAt, &emp.UpdatedAt,
			&emp.ManagerName, &emp.DesignationName,
		); err != nil {
			return nil, err
		}
		employees = append(employees, emp)
	}

	totalPages := (totalCount + params.PageSize - 1) / params.PageSize

	return &models.PaginatedEmployeeResponse{
		Employees:  employees,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// buildWhere joins conditions into a WHERE clause string.
func buildWhere(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	clause := " WHERE " + conditions[0]
	for _, c := range conditions[1:] {
		clause += " AND " + c
	}
	return clause
}

func (r *Repository) GetHrEamil() []string {
	var hrEmails []string
	r.DB.Select(&hrEmails, `
		SELECT e.email FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		WHERE r.type = 'HR' AND e.status = 'active'
	`)
	return hrEmails
}

// GetTodayBirthdays returns all active employees whose birth_date month+day matches today.
func (r *Repository) GetTodayBirthdays() ([]models.BirthdayEmployee, error) {
	rows, err := r.DB.Query(`
		SELECT id::text, full_name, email, birth_date
		FROM Tbl_Employee
		WHERE status = 'active'
		  AND birth_date IS NOT NULL
		  AND EXTRACT(MONTH FROM birth_date) = EXTRACT(MONTH FROM CURRENT_DATE)
		  AND EXTRACT(DAY   FROM birth_date) = EXTRACT(DAY   FROM CURRENT_DATE)
		ORDER BY full_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.BirthdayEmployee
	for rows.Next() {
		var emp models.BirthdayEmployee
		if err := rows.Scan(&emp.ID, &emp.Name, &emp.Email, &emp.BirthDate); err != nil {
			return nil, err
		}
		result = append(result, emp)
	}
	return result, nil
}

// GetBirthdays fetches active employees based on month/year calendar filters.
//
// month=4&year=2026  → employees whose birth month = April (any year), shown in context of 2026
// year=2026          → all employees, shown across full year 2026
// (no params)        → upcoming 30 days from today (year-wrap safe)
func (r *Repository) GetBirthdays(month, year int) ([]models.BirthdayEmployee, error) {
	base := `
	SELECT id::text, full_name, email, birth_date
	FROM Tbl_Employee
	WHERE status = 'active'
	  AND birth_date IS NOT NULL
	`

	var (
		query string
		args  []interface{}
	)

	switch {
	case month > 0 && year > 0:
		// Specific month of a specific year — match by birth month only
		query = base + `
		AND EXTRACT(MONTH FROM birth_date) = $1
		ORDER BY EXTRACT(DAY FROM birth_date)
		`
		args = append(args, month)

	case year > 0:
		// Full year view — return all employees, service will classify by that year
		query = base + `ORDER BY EXTRACT(MONTH FROM birth_date), EXTRACT(DAY FROM birth_date)`

	default:
		// Upcoming 30 days (handles year-wrap into next year)
		// Order by days-from-today so the soonest birthday comes first
		query = base + `
		AND (
			make_date(
				EXTRACT(YEAR FROM CURRENT_DATE)::int,
				EXTRACT(MONTH FROM birth_date)::int,
				EXTRACT(DAY FROM birth_date)::int
			) BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '30 days'
			OR
			make_date(
				EXTRACT(YEAR FROM CURRENT_DATE)::int + 1,
				EXTRACT(MONTH FROM birth_date)::int,
				EXTRACT(DAY FROM birth_date)::int
			) BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '30 days'
		)
		ORDER BY
			LEAST(
				make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int,     EXTRACT(MONTH FROM birth_date)::int, EXTRACT(DAY FROM birth_date)::int),
				make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int + 1, EXTRACT(MONTH FROM birth_date)::int, EXTRACT(DAY FROM birth_date)::int)
			)
		`
	}

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.BirthdayEmployee
	for rows.Next() {
		var emp models.BirthdayEmployee
		if err := rows.Scan(&emp.ID, &emp.Name, &emp.Email, &emp.BirthDate); err != nil {
			return nil, err
		}
		result = append(result, emp)
	}
	return result, nil
}
