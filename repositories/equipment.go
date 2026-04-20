package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

// ======================
// SORT FIELD MAPS
// ======================

// categorySortMap maps API sort keys to safe SQL column expressions for categories
var categorySortMap = map[string]string{
	"name":       "name",
	"created_at": "created_at",
}

// equipmentSortMap maps API sort keys to safe SQL column expressions for equipment
var equipmentSortMap = map[string]string{
	"name":               "e.name",
	"category":           "COALESCE(c.name, '')",
	"price":              "e.price",
	"total_quantity":     "e.total_quantity",
	"remaining_quantity": "e.remaining_quantity",
	"is_shared":          "e.is_shared",
	"purchase_date":      "COALESCE(e.purchase_date, '1970-01-01')",
	"created_at":         "e.created_at",
}

// assignmentSortMap maps API sort keys to safe SQL column expressions for assignments
var assignmentSortMap = map[string]string{
	"employee_name":  "e.full_name",
	"equipment_name": "eq.name",
	"quantity":       "ea.quantity",
	"assigned_at":    "ea.assigned_at",
}

// resolveSortField returns the SQL expression for a given sort key, falling back to defaultField
func resolveSortField(sortMap map[string]string, key, defaultField string) string {
	if col, ok := sortMap[key]; ok {
		return col
	}
	return defaultField
}

// ======================
// CATEGORY REPOSITORIES
// ======================

func (r *Repository) CreateCategory(tx *sqlx.Tx, data models.EquipmentCategoryRequest) error {
	_, err := tx.Exec(`
		INSERT INTO tbl_equipment_category (name, description)
		VALUES ($1,$2)
	`, data.Name, data.Description)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

// GetAllCategory supports optional pagination, search, and sorting.
// limit=0 returns all records.
func (r *Repository) GetAllCategory(limit, offset int, search, sortBy, sortDir string) ([]models.EquipmentCategoryRes, int64, error) {
	var res []models.EquipmentCategoryRes
	var total int64
	var args []interface{}
	argIndex := 1

	whereClause := ""
	if search != "" {
		whereClause = fmt.Sprintf(" WHERE name ILIKE $%d", argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	countQuery := "SELECT COUNT(*) FROM tbl_equipment_category" + whereClause
	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	sortCol := resolveSortField(categorySortMap, sortBy, "name")
	safeSortDir := "ASC"
	if sortDir == "desc" {
		safeSortDir = "DESC"
	}
	orderClause := fmt.Sprintf(" ORDER BY %s %s, id ASC", sortCol, safeSortDir)

	query := "SELECT id, name, description, created_at, updated_at FROM tbl_equipment_category" + whereClause + orderClause

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, limit, offset)
	}

	if err := r.DB.Select(&res, query, args...); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

func (r *Repository) UpdateCategory(tx *sqlx.Tx, id uuid.UUID, data models.EquipmentCategoryRequest) error {
	result, err := tx.Exec(`
		UPDATE tbl_equipment_category
		SET name=$1, description=$2, updated_at=now()
		WHERE id=$3
	`, data.Name, data.Description, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}

func (r *Repository) DeleteCategory(tx *sqlx.Tx, id uuid.UUID) error {
	result, err := tx.Exec(`DELETE FROM tbl_equipment_category WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("category not found")
	}
	return nil
}

// ======================
// EQUIPMENT REPOSITORIES
// ======================

func (r *Repository) CreateEquipment(tx *sqlx.Tx, data models.EquipmentRequest) error {
	_, err := tx.Exec(`
		INSERT INTO tbl_equipment
		(name, category_id, is_shared, price, total_quantity, remaining_quantity, purchase_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		data.Name,
		data.CategoryID,
		data.IsShared,
		data.Price,
		data.TotalQuantity,
		data.TotalQuantity, // remaining = total on creation
		data.PurchaseDate,
	)
	if err != nil {
		return fmt.Errorf("failed to create equipment: %w", err)
	}
	return nil
}

// GetAllEquipment supports optional pagination, search (name or category), and sorting.
// limit=0 returns all records.
func (r *Repository) GetAllEquipment(limit, offset int, search, sortBy, sortDir string) ([]models.EquipmentRes, int64, error) {
	res := []models.EquipmentRes{}
	var total int64
	var args []interface{}
	argIndex := 1

	whereClause := ""
	if search != "" {
		whereClause = fmt.Sprintf(` WHERE (e.name ILIKE $%d OR c.name ILIKE $%d)`, argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	countQuery := `
		SELECT COUNT(*)
		FROM tbl_equipment e
		LEFT JOIN tbl_equipment_category c ON e.category_id = c.id
	` + whereClause

	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	sortCol := resolveSortField(equipmentSortMap, sortBy, "e.name")
	orderClause := fmt.Sprintf(" ORDER BY %s %s, e.id ASC", sortCol, sortDir)

	query := `
		SELECT e.id, e.name, e.category_id, e.is_shared, e.price,
		       e.total_quantity, e.remaining_quantity,
		       e.purchase_date, e.created_at, e.updated_at
		FROM tbl_equipment e
		LEFT JOIN tbl_equipment_category c ON e.category_id = c.id
	` + whereClause + orderClause

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, limit, offset)
	}

	if err := r.DB.Select(&res, query, args...); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

// GetEquipmentByCategory supports optional pagination, search, and sorting.
// limit=0 returns all records.
func (r *Repository) GetEquipmentByCategory(categoryID uuid.UUID, limit, offset int, search, sortBy, sortDir string) ([]models.EquipmentRes, int64, error) {
	res := []models.EquipmentRes{}
	var total int64
	var args []interface{}
	argIndex := 1

	// Base WHERE for category
	args = append(args, categoryID)
	argIndex++

	whereClause := " WHERE e.category_id = $1"
	if search != "" {
		whereClause += fmt.Sprintf(" AND e.name ILIKE $%d", argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	countQuery := `SELECT COUNT(*) FROM tbl_equipment e` + whereClause
	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	sortCol := resolveSortField(equipmentSortMap, sortBy, "e.name")
	orderClause := fmt.Sprintf(" ORDER BY %s %s, e.id ASC", sortCol, sortDir)

	query := `
		SELECT e.id, e.name, e.category_id, e.is_shared, e.price,
		       e.total_quantity, e.remaining_quantity,
		       e.purchase_date, e.created_at, e.updated_at
		FROM tbl_equipment e
		LEFT JOIN tbl_equipment_category c ON e.category_id = c.id
	` + whereClause + orderClause

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, limit, offset)
	}

	if err := r.DB.Select(&res, query, args...); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

func (r *Repository) UpdateEquipment(tx *sqlx.Tx, id uuid.UUID, data models.EquipmentRequest) error {
	var currentRemaining, currentTotal int
	var currentIsShared bool
	err := tx.QueryRow(`
		SELECT remaining_quantity, total_quantity, is_shared FROM tbl_equipment WHERE id = $1
	`, id).Scan(&currentRemaining, &currentTotal, &currentIsShared)
	if err != nil {
		return fmt.Errorf("equipment not found")
	}

	newIsShared := currentIsShared
	if data.IsShared != nil {
		newIsShared = *data.IsShared
	}

	// Recalculate remaining_quantity only for non-shared equipment.
	// Shared equipment always keeps remaining = total (no deduction tracking).
	var newRemaining int
	if newIsShared {
		newRemaining = data.TotalQuantity
	} else {
		newRemaining = currentRemaining + (data.TotalQuantity - currentTotal)
		if newRemaining < 0 {
			newRemaining = 0
		}
	}

	result, err := tx.Exec(`
		UPDATE tbl_equipment
		SET name=$1, category_id=$2, is_shared=$3, price=$4,
		    total_quantity=$5, remaining_quantity=$6, purchase_date=$7, updated_at=now()
		WHERE id=$8
	`, data.Name, data.CategoryID, newIsShared, data.Price,
		data.TotalQuantity, newRemaining, data.PurchaseDate, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("equipment not found")
	}
	return nil
}

func (r *Repository) DeleteEquipment(tx *sqlx.Tx, id uuid.UUID) error {
	result, err := tx.Exec(`DELETE FROM tbl_equipment WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("equipment not found")
	}
	return nil
}

// ======================
// ASSIGNMENT REPOSITORIES
// ======================

func (r *Repository) AssignEquipment(tx *sqlx.Tx, req models.AssignEquipmentRequest) error {
	var remaining int
	var isShared bool

	err := tx.QueryRow(`
		SELECT remaining_quantity, is_shared FROM tbl_equipment WHERE id=$1 FOR UPDATE
	`, req.EquipmentID).Scan(&remaining, &isShared)
	if err != nil {
		return fmt.Errorf("equipment not found")
	}

	// Shared equipment: no quantity limit — multiple employees can use simultaneously.
	// Non-shared equipment: must have enough remaining units.
	if !isShared && remaining < req.Quantity {
		return fmt.Errorf("not enough equipment available: need %d, have %d", req.Quantity, remaining)
	}

	_, err = tx.Exec(`
		INSERT INTO tbl_equipment_assignment (equipment_id, employee_id, assigned_by, quantity)
		VALUES ($1,$2,$3,$4)
	`, req.EquipmentID, req.EmployeeID, req.AssignedBy, req.Quantity)
	if err != nil {
		return err
	}

	// Only reduce remaining_quantity for non-shared equipment.
	if !isShared {
		_, err = tx.Exec(`
			UPDATE tbl_equipment SET remaining_quantity = remaining_quantity - $1 WHERE id=$2
		`, req.Quantity, req.EquipmentID)
		return err
	}

	return nil
}

// assignmentSelectQuery is the shared SELECT for assignment responses
const assignmentSelectQuery = `
	SELECT
	    ea.id          AS assignment_id,
	    ea.employee_id,
	    e.full_name    AS employee_name,
	    e.email        AS employee_email,
	    ea.equipment_id,
	    eq.name        AS equipment_name,
	    eq.purchase_date,
	    ea.quantity,
	    ea.assigned_at,
	    ab.full_name   AS approved_by_name
	FROM tbl_equipment_assignment ea
	JOIN tbl_employee e   ON e.id  = ea.employee_id
	JOIN tbl_equipment eq ON eq.id = ea.equipment_id
	JOIN tbl_employee ab  ON ab.id = ea.assigned_by
`

// GetAllAssignedEquipment supports optional pagination, search, and sorting.
// limit=0 returns all records.
func (r *Repository) GetAllAssignedEquipment(limit, offset int, search, sortBy, sortDir string) ([]models.AssignEquipmentResponse, int64, error) {
	res := []models.AssignEquipmentResponse{}
	var total int64
	var args []interface{}
	argIndex := 1

	whereClause := ""
	if search != "" {
		whereClause = fmt.Sprintf(` WHERE (e.full_name ILIKE $%d OR eq.name ILIKE $%d)`, argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	countQuery := `
		SELECT COUNT(*)
		FROM tbl_equipment_assignment ea
		JOIN tbl_employee e   ON e.id  = ea.employee_id
		JOIN tbl_equipment eq ON eq.id = ea.equipment_id
	` + whereClause

	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	sortCol := resolveSortField(assignmentSortMap, sortBy, "ea.assigned_at")
	orderClause := fmt.Sprintf(" ORDER BY %s %s, ea.id ASC", sortCol, sortDir)

	query := assignmentSelectQuery + whereClause + orderClause

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, limit, offset)
	}

	if err := r.DB.Select(&res, query, args...); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

// GetAssignedEquipmentByEmployee supports optional pagination, search, and sorting.
// limit=0 returns all records.
func (r *Repository) GetAssignedEquipmentByEmployee(employeeID uuid.UUID, limit, offset int, search, sortBy, sortDir string) ([]models.AssignEquipmentResponse, int64, error) {
	res := []models.AssignEquipmentResponse{}
	var total int64
	var args []interface{}
	argIndex := 1

	args = append(args, employeeID)
	argIndex++

	whereClause := " WHERE ea.employee_id = $1"
	if search != "" {
		whereClause += fmt.Sprintf(" AND eq.name ILIKE $%d", argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	countQuery := `
		SELECT COUNT(*)
		FROM tbl_equipment_assignment ea
		JOIN tbl_equipment eq ON eq.id = ea.equipment_id
	` + whereClause

	if err := r.DB.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	sortCol := resolveSortField(assignmentSortMap, sortBy, "ea.assigned_at")
	orderClause := fmt.Sprintf(" ORDER BY %s %s, ea.id ASC", sortCol, sortDir)

	query := assignmentSelectQuery + whereClause + orderClause

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, limit, offset)
	}

	if err := r.DB.Select(&res, query, args...); err != nil {
		return nil, 0, err
	}
	return res, total, nil
}

// RemoveEquipment hard-deletes the most recent assignment.
// For non-shared equipment, restores remaining_quantity.
// For shared equipment, quantity tracking is skipped.
func (r *Repository) RemoveEquipment(tx *sqlx.Tx, req models.RemoveEquipmentRequest) error {
	var assignmentID uuid.UUID
	var qty int

	err := tx.QueryRow(`
		SELECT id, quantity FROM tbl_equipment_assignment
		WHERE equipment_id = $1 AND employee_id = $2
		ORDER BY assigned_at DESC LIMIT 1
	`, req.EquipmentID, req.EmployeeID).Scan(&assignmentID, &qty)
	if err != nil {
		return fmt.Errorf("assignment not found for equipment_id=%s, employee_id=%s: %w",
			req.EquipmentID, req.EmployeeID, err)
	}

	result, err := tx.Exec(`DELETE FROM tbl_equipment_assignment WHERE id = $1`, assignmentID)
	if err != nil {
		return fmt.Errorf("failed to delete assignment id=%s: %w", assignmentID, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("assignment id=%s not found", assignmentID)
	}

	// Only restore remaining_quantity for non-shared equipment.
	var isShared bool
	if err := tx.QueryRow(`SELECT is_shared FROM tbl_equipment WHERE id = $1`, req.EquipmentID).Scan(&isShared); err != nil {
		return fmt.Errorf("equipment id=%s not found", req.EquipmentID)
	}

	if !isShared {
		result2, err := tx.Exec(`
			UPDATE tbl_equipment SET remaining_quantity = remaining_quantity + $1 WHERE id = $2
		`, qty, req.EquipmentID)
		if err != nil {
			return fmt.Errorf("failed to update equipment quantity: %w", err)
		}
		if rows, _ := result2.RowsAffected(); rows == 0 {
			return fmt.Errorf("equipment id=%s not found", req.EquipmentID)
		}
	}

	return nil
}

// UpdateAssignment handles quantity updates and reassignments.
// For shared equipment, assignment records are updated but remaining_quantity is never touched.
func (r *Repository) UpdateAssignment(tx *sqlx.Tx, req models.UpdateAssignmentRequest) error {
	var assignmentID uuid.UUID
	var currentQty int

	err := tx.QueryRow(`
		SELECT id, quantity FROM tbl_equipment_assignment
		WHERE equipment_id = $1 AND employee_id = $2
		ORDER BY assigned_at DESC LIMIT 1
	`, req.EquipmentID, req.FromEmployeeID).Scan(&assignmentID, &currentQty)
	if err != nil {
		return fmt.Errorf("assignment not found for equipment_id=%s, employee_id=%s: %w",
			req.EquipmentID, req.FromEmployeeID, err)
	}

	// Fetch is_shared once — drives all quantity tracking decisions below.
	var isShared bool
	if err := tx.QueryRow(`SELECT is_shared FROM tbl_equipment WHERE id = $1 FOR UPDATE`, req.EquipmentID).Scan(&isShared); err != nil {
		return fmt.Errorf("equipment not found: %w", err)
	}

	// ── Reassignment to another employee ──────────────────────────────────────
	if req.ToEmployeeID != nil {
		if req.Quantity > currentQty {
			return fmt.Errorf("quantity %d exceeds assigned amount %d", req.Quantity, currentQty)
		}

		// Step 1: Remove from current employee (full or partial)
		if req.Quantity == currentQty {
			if _, err := tx.Exec(`DELETE FROM tbl_equipment_assignment WHERE id = $1`, assignmentID); err != nil {
				return fmt.Errorf("failed to delete current assignment: %w", err)
			}
		} else {
			if _, err := tx.Exec(`
				UPDATE tbl_equipment_assignment SET quantity = quantity - $1 WHERE id = $2
			`, req.Quantity, assignmentID); err != nil {
				return fmt.Errorf("failed to reduce current assignment quantity: %w", err)
			}
		}

		// Step 2 & 3 only apply to non-shared equipment.
		// Shared equipment: just move the assignment record, no pool changes.
		if !isShared {
			// Add back to pool (will be re-consumed when assigning to new employee)
			if _, err := tx.Exec(`
				UPDATE tbl_equipment SET remaining_quantity = remaining_quantity + $1 WHERE id = $2
			`, req.Quantity, req.EquipmentID); err != nil {
				return fmt.Errorf("failed to restore equipment quantity: %w", err)
			}
		}

		// Step 3: Upsert assignment for new employee
		var newAssignID uuid.UUID
		var existingQty int
		err = tx.QueryRow(`
			SELECT id, quantity FROM tbl_equipment_assignment
			WHERE equipment_id = $1 AND employee_id = $2
			ORDER BY assigned_at DESC LIMIT 1
		`, req.EquipmentID, *req.ToEmployeeID).Scan(&newAssignID, &existingQty)

		if err == nil {
			// New employee already has an assignment — merge into it
			if _, err := tx.Exec(`
				UPDATE tbl_equipment_assignment SET quantity = quantity + $1, assigned_by = $2 WHERE id = $3
			`, req.Quantity, req.AssignedBy, newAssignID); err != nil {
				return fmt.Errorf("failed to update new employee assignment: %w", err)
			}
			// Non-shared: consume from pool (we added it back above)
			if !isShared {
				if _, err := tx.Exec(`
					UPDATE tbl_equipment SET remaining_quantity = remaining_quantity - $1 WHERE id = $2
				`, req.Quantity, req.EquipmentID); err != nil {
					return fmt.Errorf("failed to consume equipment quantity: %w", err)
				}
			}
		} else {
			// New employee has no assignment — create one
			if _, err := tx.Exec(`
				INSERT INTO tbl_equipment_assignment (equipment_id, employee_id, assigned_by, quantity)
				VALUES ($1, $2, $3, $4)
			`, req.EquipmentID, *req.ToEmployeeID, req.AssignedBy, req.Quantity); err != nil {
				return fmt.Errorf("failed to create new employee assignment: %w", err)
			}
			// Non-shared: consume from pool
			if !isShared {
				if _, err := tx.Exec(`
					UPDATE tbl_equipment SET remaining_quantity = remaining_quantity - $1 WHERE id = $2
				`, req.Quantity, req.EquipmentID); err != nil {
					return fmt.Errorf("failed to consume equipment quantity: %w", err)
				}
			}
		}
		return nil
	}

	// ── Quantity update for same employee ─────────────────────────────────────
	diff := req.Quantity - currentQty

	if !isShared {
		// Only touch remaining_quantity for non-shared equipment
		if diff > 0 {
			var remaining int
			if err := tx.QueryRow(`SELECT remaining_quantity FROM tbl_equipment WHERE id = $1`, req.EquipmentID).Scan(&remaining); err != nil {
				return fmt.Errorf("equipment not found: %w", err)
			}
			if remaining < diff {
				return fmt.Errorf("not enough quantity available: need %d, have %d", diff, remaining)
			}
			if _, err := tx.Exec(`
				UPDATE tbl_equipment SET remaining_quantity = remaining_quantity - $1 WHERE id = $2
			`, diff, req.EquipmentID); err != nil {
				return fmt.Errorf("failed to reduce equipment quantity: %w", err)
			}
		} else if diff < 0 {
			if _, err := tx.Exec(`
				UPDATE tbl_equipment SET remaining_quantity = remaining_quantity + $1 WHERE id = $2
			`, -diff, req.EquipmentID); err != nil {
				return fmt.Errorf("failed to increase equipment quantity: %w", err)
			}
		}
	}

	result, err := tx.Exec(`
		UPDATE tbl_equipment_assignment SET quantity = $1 WHERE id = $2
	`, req.Quantity, assignmentID)
	if err != nil {
		return fmt.Errorf("failed to update assignment quantity: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("assignment id=%s not found", assignmentID)
	}
	return nil
}
