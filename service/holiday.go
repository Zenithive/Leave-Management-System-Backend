package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
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
		fmt.Println(err.Error())
		return "", err
	}

	holidayID = id

	return holidayID, nil
}

func (s *holidayService) GetAllHolidays(ctx context.Context) ([]models.Holiday, error) {
	return s.Repo.GetAllHolidays(ctx)
}

func (s *holidayService) DeleteHoliday(ctx context.Context, id string) error {

	return common.ExecuteTransaction(ctx, s.DB, func(tx *sqlx.Tx) error {

		if err := s.Repo.DeleteHoliday(ctx, id); err != nil {
			return err
		}
		return nil
	})
}
