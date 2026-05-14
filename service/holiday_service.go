package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

type HolidayService struct {
	repo *repositories.Repository
}

func NewHolidayService(repo *repositories.Repository) *HolidayService {
	return &HolidayService{repo: repo}
}

// AddResult is returned by Add.
type AddHolidayResult struct {
	ID   string
	Date string
}

// Add inserts a new holiday. Normalizes the date to UTC midnight and logs the action.
func (s *HolidayService) Add(tx *sqlx.Tx, input models.HolidayInput, callerID uuid.UUID) (*AddHolidayResult, error) {
	// Normalize to UTC midnight — avoids timezone drift in DB storage
	normalizedDate := time.Date(input.Date.Year(), input.Date.Month(), input.Date.Day(), 0, 0, 0, 0, time.UTC)

	id, err := s.repo.AddHoliday(tx, input.Name, normalizedDate, input.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to add holiday: %w", err)
	}

	if err := s.repo.AddLog(models.NewCommon(constant.ComponentHoliday, constant.ActionCreate, callerID), tx); err != nil {
		return nil, fmt.Errorf("failed to log action: %w", err)
	}

	return &AddHolidayResult{
		ID:   id,
		Date: normalizedDate.Format("2006-01-02"),
	}, nil
}

// GetAll returns all holidays ordered by date.
func (s *HolidayService) GetAll() ([]models.Holiday, error) {
	holidays, err := s.repo.GetAllHolidays()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch holidays: %w", err)
	}
	return holidays, nil
}

// Delete removes a holiday by ID and logs the action.
func (s *HolidayService) Delete(tx *sqlx.Tx, id string, callerID uuid.UUID) error {
	if err := s.repo.DeleteHoliday(id, tx); err != nil {
		return fmt.Errorf("failed to delete holiday: %w", err)
	}
	if err := s.repo.AddLog(models.NewCommon(constant.ComponentHoliday, constant.ActionDelete, callerID), tx); err != nil {
		return fmt.Errorf("failed to log action: %w", err)
	}
	return nil
}
