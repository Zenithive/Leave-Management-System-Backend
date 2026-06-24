package repositories

import (
	"context"
	"time"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/jmoiron/sqlx"
)

type HolidayRepository interface {
	AddHoliday(ctx context.Context, name string, date time.Time, typ string) (string, error)
	GetAllHolidays(ctx context.Context) ([]models.Holiday, error)
	DeleteHoliday(ctx context.Context, id string) error
}

type holidayRepo struct {
	DB *sqlx.DB
}

func NewHolidayRepository(db *sqlx.DB) HolidayRepository {
	return &holidayRepo{DB: db}
}

func (r *holidayRepo) AddHoliday(ctx context.Context, name string, date time.Time, typ string) (string, error) {
	if typ == "" {
		typ = "HOLIDAY"
	}

	day := date.Weekday().String()
	var id string

	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO Tbl_Holiday (name, date, day, type, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id
	`, name, date, day, typ).Scan(&id)

	return id, err
}
func (r *holidayRepo) GetAllHolidays(ctx context.Context) ([]models.Holiday, error) {

	rows, err := r.DB.QueryxContext(ctx, `
		SELECT id, name, date, day, type, created_at, updated_at
		FROM Tbl_Holiday
		ORDER BY date
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holidays []models.Holiday

	for rows.Next() {
		var h models.Holiday
		if err := rows.StructScan(&h); err != nil {
			return nil, err
		}
		holidays = append(holidays, h)
	}

	return holidays, nil
}
func (r *holidayRepo) DeleteHoliday(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx, `
		DELETE FROM Tbl_Holiday
		WHERE id = $1
	`, id)

	return err
}
