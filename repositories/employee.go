package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

// ─────────────────────────────────────────────
// Sort helpers
// ─────────────────────────────────────────────

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

// ─────────────────────────────────────────────
// Auth
// ─────────────────────────────────────────────

func (r *Repository) GetEmployeeByEmail(email string) (EmployeeAuthData, error) {
	var emp EmployeeAuthData
	err := r.DB.Get(&emp, `
		SELECT e.id, e.email, e.password, r.type AS role, e.status
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		WHERE e.email = $1 AND e.status = 'active'
		LIMIT 1
	`, email)
	return emp, err
}

// ─────────────────────────────────────────────
// Read
// ─────────────────────────────────────────────

func (r *Repository) GetEmployeeByID(empID uuid.UUID) (*models.EmployeeResponse, error) {
	var emp models.EmployeeResponse
	err := r.DB.QueryRow(`
		SELECT
			e.id, e.full_name, e.email, e.status,
			r.type AS role, e.manager_id, e.designation_id,
			e.salary, e.joining_date, e.birth_date, e.ending_date,
			e.created_at, e.updated_at,
			m.full_name AS manager_name,
			d.designation_name
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		LEFT JOIN Tbl_Employee m ON e.manager_id = m.id
		LEFT JOIN Tbl_Designation d ON e.designation_id = d.id
		WHERE e.id = $1
	`, empID).Scan(
		&emp.ID, &emp.FullName, &emp.Email, &emp.Status,
		&emp.Role, &emp.ManagerID, &emp.DesignationID,
		&emp.Salary, &emp.JoiningDate, &emp.BirthDate, &emp.EndingDate,
		&emp.CreatedAt, &emp.UpdatedAt,
		&emp.ManagerName, &emp.DesignationName,
	)
	if err != nil {
		return nil, err
	}
	return &emp, nil
}

func (r *Repository) GetEmployeeStatus(employeeID uuid.UUID) (string, error) {
	var status string
	err := r.DB.Get(&status, `SELECT status FROM Tbl_Employee WHERE id=$1`, employeeID)
	return status, err
}

func (r *Repository) GetEmployeeRole(employeeID uuid.UUID) (string, error) {
	var role string
	err := r.DB.Get(&role, `
		SELECT r.type FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		WHERE e.id = $1
	`, employeeID)
	return role, err
}

func (r *Repository) GetEmployeeCurrentRole(empID string) (string, error) {
	var role string
	err := r.DB.QueryRow(`
		SELECT R.TYPE FROM TBL_EMPLOYEE E
		JOIN TBL_ROLE R ON E.ROLE_ID = R.ID
		WHERE E.ID = $1
	`, empID).Scan(&role)
	return role, err
}

func (r *Repository) GetEmployeeCurrentRoleAndManagerStatus(empID uuid.UUID) (string, bool, error) {
	var role string
	var count int
	err := r.DB.QueryRow(`
		SELECT r.type,
		       (SELECT COUNT(*) FROM Tbl_Employee e2 WHERE e2.manager_id=e.id) AS sub_count
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id=r.id
		WHERE e.id=$1
	`, empID).Scan(&role, &count)
	if err != nil {
		return "", false, err
	}
	return role, count > 0, nil
}

func (r *Repository) GetAllEmployees(params models.EmployeeFilterParams, role string) (*models.PaginatedEmployeeResponse, error) {
	salaryCol := "NULL::double precision AS salary"
	if role == constant.ROLE_ADMIN || role == constant.ROLE_SUPER_ADMIN {
		salaryCol = "e.salary"
	}

	conditions := []string{}
	args := []interface{}{}
	n := 1

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(e.full_name ILIKE $%d OR e.email ILIKE $%d OR m.full_name ILIKE $%d)", n, n, n,
		))
		args = append(args, "%"+params.Search+"%")
		n++
	}
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", n))
		args = append(args, params.Status)
		n++
	}
	if len(params.Roles) > 0 {
		placeholders := make([]string, len(params.Roles))
		for i, r := range params.Roles {
			placeholders[i] = fmt.Sprintf("$%d", n)
			args = append(args, r)
			n++
		}
		conditions = append(conditions, fmt.Sprintf("r.type IN (%s)", strings.Join(placeholders, ",")))
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

	var totalCount int
	if err := r.DB.Get(&totalCount, "SELECT COUNT(*) "+baseJoins+whereClause, args...); err != nil {
		return nil, err
	}

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

func (r *Repository) GetEmployeesByManagerID(managerID uuid.UUID, params models.TeamFilterParams) ([]models.EmployeeResponse, error) {
	orderBy := resolveEmployeeSort(params.SortBy, params.SortOrder)
	query := fmt.Sprintf(`
		SELECT
			e.id, e.full_name, e.email, e.status,
			r.type AS role, e.manager_id, e.designation_id,
			e.salary, e.joining_date, e.birth_date, e.ending_date,
			e.created_at, e.updated_at,
			m.full_name AS manager_name,
			d.designation_name
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		LEFT JOIN Tbl_Employee m ON e.manager_id = m.id
		LEFT JOIN Tbl_Designation d ON e.designation_id = d.id
		WHERE e.manager_id = $1
		ORDER BY %s
	`, orderBy)

	rows, err := r.DB.Query(query, managerID)
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
	return employees, nil
}

// ─────────────────────────────────────────────
// Create
// ─────────────────────────────────────────────

func (r *Repository) CheckEmailExists(email string) (bool, error) {
	var existing string
	err := r.DB.QueryRow(`SELECT email FROM Tbl_Employee WHERE email=$1`, email).Scan(&existing)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *Repository) GetRoleID(role string) (string, error) {
	var id string
	err := r.DB.QueryRow(`SELECT id FROM Tbl_Role WHERE type=$1`, role).Scan(&id)
	return id, err
}

func (r *Repository) InsertEmployee(tx *sqlx.Tx, fullName, email, roleID, password string, salary *float64, joining *time.Time) (uuid.UUID, error) {
	var employeeID uuid.UUID
	err := tx.QueryRow(`
		INSERT INTO Tbl_Employee (full_name, email, role_id, password, salary, joining_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, fullName, email, roleID, password, salary, joining).Scan(&employeeID)
	if err != nil {
		return employeeID, err
	}
	return employeeID, nil
}

// ─────────────────────────────────────────────
// Update
// ─────────────────────────────────────────────

func (r *Repository) UpdateEmployeeInfo(empID uuid.UUID, fullName, email string, salary *float64, joiningDate, birthDate, endingDate *time.Time) error {
	_, err := r.DB.Exec(`
		UPDATE Tbl_Employee
		SET full_name=$1, email=$2, salary=$3, joining_date=$4, birth_date=$5, ending_date=$6, updated_at=NOW()
		WHERE id=$7
	`, fullName, email, salary, joiningDate, birthDate, endingDate, empID)
	return err
}

func (r *Repository) UpdateEmployeeInfoTx(tx *sqlx.Tx, empID uuid.UUID, fullName, email string, salary *float64, joiningDate, birthDate, endingDate *time.Time) error {
	_, err := tx.Exec(`
		UPDATE Tbl_Employee
		SET full_name=$1, email=$2, salary=$3, joining_date=$4, birth_date=$5, ending_date=$6, updated_at=NOW()
		WHERE id=$7
	`, fullName, email, salary, joiningDate, birthDate, endingDate, empID)
	return err
}

func (r *Repository) UpdateEmployeePassword(empID uuid.UUID, hashedPassword string) error {
	_, err := r.DB.Exec(`
		UPDATE Tbl_Employee SET password=$1, updated_at=NOW() WHERE id=$2
	`, hashedPassword, empID)
	return err
}

func (r *Repository) UpdateEmployeeRole(tx *sqlx.Tx, empID uuid.UUID, newRole string) (string, error) {
	var id string
	err := tx.QueryRow(`
		UPDATE TBL_EMPLOYEE
		SET ROLE_ID=(SELECT ID FROM TBL_ROLE WHERE TYPE=$1), UPDATED_AT=NOW()
		WHERE ID=$2
		RETURNING ID
	`, newRole, empID).Scan(&id)
	return id, err
}

// UpdateEmployeeRoleTx is an alias kept for backward compatibility.
func (r *Repository) UpdateEmployeeRoleTx(tx *sqlx.Tx, empID uuid.UUID, newRole string) (string, error) {
	return r.UpdateEmployeeRole(tx, empID, newRole)
}

func (r *Repository) UpdateEmployeeDesignation(empID uuid.UUID, designationID *uuid.UUID) error {
	_, err := r.DB.Exec(`
		UPDATE Tbl_Employee SET designation_id=$1, updated_at=NOW() WHERE id=$2
	`, designationID, empID)
	return err
}

func (r *Repository) UpdateManager(empID, managerID uuid.UUID) error {
	_, err := r.DB.Exec(`
		UPDATE TBL_EMPLOYEE SET MANAGER_ID=$1, UPDATED_AT=NOW() WHERE ID=$2
	`, managerID, empID)
	return err
}

func (r *Repository) DeleteEmployeeStatus(tx *sqlx.Tx, id uuid.UUID) (string, error) {
	var currentStatus string
	if err := tx.QueryRow(`SELECT status FROM Tbl_Employee WHERE id=$1`, id).Scan(&currentStatus); err != nil {
		return "", fmt.Errorf("employee not found: %w", err)
	}

	newStatus := "active"
	if currentStatus == "active" {
		newStatus = "deactive"
	}

	if _, err := tx.Exec(`UPDATE Tbl_Employee SET status=$1, updated_at=NOW() WHERE id=$2`, newStatus, id); err != nil {
		return "", err
	}

	if newStatus == "deactive" {
		if err := r.restoreEmployeeEquipment(tx, id); err != nil {
			return "", err
		}
	}
	return newStatus, nil
}

func (r *Repository) restoreEmployeeEquipment(tx *sqlx.Tx, employeeID uuid.UUID) error {
	rows, err := tx.Query(`SELECT DISTINCT equipment_id FROM tbl_equipment_assignment WHERE employee_id=$1`, employeeID)
	if err != nil {
		return fmt.Errorf("failed to fetch assignments: %w", err)
	}
	defer rows.Close()

	var equipmentIDs []uuid.UUID
	for rows.Next() {
		var eqID uuid.UUID
		if err := rows.Scan(&eqID); err != nil {
			return fmt.Errorf("failed to scan equipment id: %w", err)
		}
		equipmentIDs = append(equipmentIDs, eqID)
	}

	for _, eqID := range equipmentIDs {
		if err := r.RemoveEquipment(tx, models.RemoveEquipmentRequest{
			EmployeeID:  employeeID,
			EquipmentID: eqID,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ─────────────────────────────────────────────
// Misc
// ─────────────────────────────────────────────

func (r *Repository) ManagerExists(id uuid.UUID) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM TBL_EMPLOYEE WHERE ID=$1)`, id).Scan(&exists)
	return exists, err
}

func (r *Repository) ChackManagerPermission() (bool, error) {
	var exists bool
	err := r.DB.QueryRow(`SELECT allow_manager_add_leave FROM Tbl_Company_Settings LIMIT 1`).Scan(&exists)
	return exists, err
}

func (r *Repository) GetAdminAndEmployeeEmail(id uuid.UUID) ([]string, error) {
	var recipients []string
	var managerEmail string
	if err := r.DB.Get(&managerEmail, `
		SELECT e2.email FROM Tbl_Employee e1
		JOIN Tbl_Employee e2 ON e1.manager_id = e2.id
		WHERE e1.id = $1
	`, id); err == nil && managerEmail != "" {
		recipients = append(recipients, managerEmail)
	}
	var adminEmails []string
	r.DB.Select(&adminEmails, `
		SELECT e.email FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		WHERE r.type IN ('ADMIN', 'SUPERADMIN', 'HR') AND e.status = 'active'
	`)
	recipients = append(recipients, adminEmails...)
	return recipients, nil
}

func (r *Repository) GetEmployeeDetailsForNotification(id uuid.UUID) (struct {
	Email    string `db:"email"`
	FullName string `db:"full_name"`
}, error) {
	var empDetails struct {
		Email    string `db:"email"`
		FullName string `db:"full_name"`
	}
	err := r.DB.Get(&empDetails, "SELECT email, full_name FROM Tbl_Employee WHERE id=$1", id)
	return empDetails, err
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

// ─────────────────────────────────────────────
// Birthday
// ─────────────────────────────────────────────

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

func (r *Repository) GetBirthdays(month, year int) ([]models.BirthdayEmployee, error) {
	base := `
		SELECT id::text, full_name, email, birth_date
		FROM Tbl_Employee
		WHERE status = 'active' AND birth_date IS NOT NULL
	`
	var query string
	var args []interface{}

	switch {
	case month > 0 && year > 0:
		query = base + `AND EXTRACT(MONTH FROM birth_date) = $1 ORDER BY EXTRACT(DAY FROM birth_date)`
		args = append(args, month)
	case year > 0:
		query = base + `ORDER BY EXTRACT(MONTH FROM birth_date), EXTRACT(DAY FROM birth_date)`
	default:
		query = base + `
		AND (
			make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int, EXTRACT(MONTH FROM birth_date)::int, EXTRACT(DAY FROM birth_date)::int)
				BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '30 days'
			OR
			make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int + 1, EXTRACT(MONTH FROM birth_date)::int, EXTRACT(DAY FROM birth_date)::int)
				BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '30 days'
		)
		ORDER BY LEAST(
			make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int,     EXTRACT(MONTH FROM birth_date)::int, EXTRACT(DAY FROM birth_date)::int),
			make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int + 1, EXTRACT(MONTH FROM birth_date)::int, EXTRACT(DAY FROM birth_date)::int)
		)`
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

// ─────────────────────────────────────────────
// Active employees (used by leave allocation)
// ─────────────────────────────────────────────

type ActiveEmployeeRole struct {
	ID          uuid.UUID  `db:"id"`
	Role        string     `db:"role"`
	JoiningDate *time.Time `db:"joining_date"`
}

func (r *Repository) GetAllActiveEmployeesWithRoles(tx *sqlx.Tx) ([]ActiveEmployeeRole, error) {
	var employees []ActiveEmployeeRole
	err := tx.Select(&employees, `
		SELECT e.id, r.type AS role, e.joining_date
		FROM Tbl_Employee e
		JOIN Tbl_Role r ON e.role_id = r.id
		WHERE e.status = 'active'
	`)
	return employees, err
}
