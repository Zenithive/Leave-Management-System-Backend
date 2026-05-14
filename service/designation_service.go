package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

type DesignationService struct {
	repo *repositories.Repository
}

func NewDesignationService(repo *repositories.Repository) *DesignationService {
	return &DesignationService{repo: repo}
}

// Create inserts a new designation and logs the action.
func (s *DesignationService) Create(tx *sqlx.Tx, input *models.DesignationInput, callerID uuid.UUID) (string, error) {
	id, err := s.repo.CreateDesignation(tx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create designation: %w", err)
	}
	if err := s.repo.AddLog(models.NewCommon(constant.ComponentDesignation, constant.ActionCreate, callerID), tx); err != nil {
		return "", fmt.Errorf("failed to log action: %w", err)
	}
	return id, nil
}

// GetAll returns all designations.
func (s *DesignationService) GetAll() ([]models.Designation, error) {
	designations, err := s.repo.GetAllDesignations()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch designations: %w", err)
	}
	return designations, nil
}

// GetByID returns a single designation by UUID.
func (s *DesignationService) GetByID(id uuid.UUID) (*models.Designation, error) {
	designation, err := s.repo.GetDesignationByID(id)
	if err != nil {
		return nil, fmt.Errorf("designation not found")
	}
	return designation, nil
}

// Update modifies an existing designation and logs the action.
func (s *DesignationService) Update(tx *sqlx.Tx, id uuid.UUID, input *models.DesignationInput, callerID uuid.UUID) error {
	if err := s.repo.UpdateDesignation(tx, id, input); err != nil {
		return fmt.Errorf("failed to update designation: %w", err)
	}
	if err := s.repo.AddLog(models.NewCommon(constant.ComponentDesignation, constant.ActionUpdate, callerID), tx); err != nil {
		return fmt.Errorf("failed to log action: %w", err)
	}
	return nil
}

// Delete removes a designation and logs the action.
// Due to ON DELETE SET NULL, employee designation_id is set to NULL automatically.
func (s *DesignationService) Delete(tx *sqlx.Tx, id uuid.UUID, callerID uuid.UUID) error {
	if err := s.repo.DeleteDesignation(tx, id); err != nil {
		return fmt.Errorf("failed to delete designation: %w", err)
	}
	if err := s.repo.AddLog(models.NewCommon(constant.ComponentDesignation, constant.ActionDelete, callerID), tx); err != nil {
		return fmt.Errorf("failed to log action: %w", err)
	}
	return nil
}
