package service

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
)

type LeaveTimingService struct {
	repo *repositories.Repository
}

func NewLeaveTimingService(repo *repositories.Repository) *LeaveTimingService {
	return &LeaveTimingService{repo: repo}
}

// GetAll returns all leave timing records.
func (s *LeaveTimingService) GetAll() ([]models.LeaveTimingResponse, error) {
	data, err := s.repo.GetLeaveTiming()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leave timing: %w", err)
	}
	if data == nil {
		data = []models.LeaveTimingResponse{}
	}
	return data, nil
}

// GetByID returns a single leave timing record by ID.
func (s *LeaveTimingService) GetByID(id int) (*models.LeaveTimingResponse, error) {
	data, err := s.repo.GetLeaveTimingByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("leave timing not found")
		}
		return nil, fmt.Errorf("failed to fetch leave timing: %w", err)
	}
	return data, nil
}

// Update updates the timing value for a leave timing record.
func (s *LeaveTimingService) Update(tx *sqlx.Tx, id int, timing string) error {
	err := s.repo.UpdateLeaveTiming(tx, id, timing)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("leave timing not found")
		}
		return fmt.Errorf("failed to update leave timing: %w", err)
	}
	return nil
}
