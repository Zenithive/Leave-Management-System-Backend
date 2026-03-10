package repositories

import (
	"github.com/google/uuid"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

// ------------------ DESIGNATION OPERATIONS ------------------

// CreateDesignation inserts a new designation
func (r *Repository) CreateDesignation(input *models.DesignationInput) (string, error) {
	var id string
	query := `
		INSERT INTO Tbl_Designation (designation_name, description)
		VALUES ($1, $2)
		RETURNING id
	`
	err := r.DB.QueryRow(query, input.DesignationName, input.Description).Scan(&id)
	return id, err
}

// GetAllDesignations fetches all designations
func (r *Repository) GetAllDesignations() ([]models.Designation, error) {
	var designations []models.Designation
	query := `
		SELECT id, designation_name, description
		FROM Tbl_Designation
		ORDER BY designation_name
	`
	err := r.DB.Select(&designations, query)
	return designations, err
}

// GetDesignationByID fetches a single designation by ID
func (r *Repository) GetDesignationByID(id uuid.UUID) (*models.Designation, error) {
	var designation models.Designation
	query := `
		SELECT id, designation_name, description
		FROM Tbl_Designation
		WHERE id = $1
	`
	err := r.DB.Get(&designation, query, id)
	if err != nil {
		return nil, err
	}
	return &designation, nil
}

// UpdateDesignation updates an existing designation
func (r *Repository) UpdateDesignation(id uuid.UUID, input *models.DesignationInput) error {
	query := `
		UPDATE Tbl_Designation
		SET designation_name = $1, description = $2
		WHERE id = $3
	`
	_, err := r.DB.Exec(query, input.DesignationName, input.Description, id)
	return err
}

// DeleteDesignation deletes a designation by ID
// Due to ON DELETE SET NULL constraint, employee designation_id will be set to NULL automatically
func (r *Repository) DeleteDesignation(id uuid.UUID) error {
	query := `DELETE FROM Tbl_Designation WHERE id = $1`
	_, err := r.DB.Exec(query, id)
	return err
}
