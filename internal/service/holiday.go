package service

import (
	"context"
	"net/http"
	"time"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/internal/repositories"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/jmoiron/sqlx"
)

type HolidayService interface {
	AddHoliday(ctx context.Context, input *models.Holiday) (string, error)
	GetAllHolidays(ctx context.Context) ([]models.Holiday, error)
	DeleteHoliday(ctx context.Context, id string) error
}
type holidayService struct {
	DB   *sqlx.DB
	Repo repositories.HolidayRepository
}

func NewHolidayService(repo repositories.HolidayRepository) HolidayService {
	return &holidayService{
		Repo: repo,
	}
}

func (s *holidayService) AddHoliday(ctx context.Context, input *models.Holiday) (string, error) {

	var holidayID string

	// normalize date
	normalizedDate := time.Date(
		input.Date.Year(),
		input.Date.Month(),
		input.Date.Day(),
		0, 0, 0, 0,
		time.UTC,
	)

	id, err := s.Repo.AddHoliday(ctx, input.Name, normalizedDate, input.Type)
	if err != nil {
		return "", err
	}

	holidayID = id

	return holidayID, nil
}

func (s *holidayService) GetAllHolidays(ctx context.Context) ([]models.Holiday, error) {
	return s.Repo.GetAllHolidays(ctx)
}

func (s *holidayService) DeleteHoliday(ctx context.Context, id string) error {

	if err := s.Repo.DeleteHoliday(ctx, id); err != nil {
		return errors.CustomErr(http.StatusInternalServerError, err.Error())
	}
	return nil
}
