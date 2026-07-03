package repositories

import (
	"time"

	"github.com/google/uuid"
)

type EmpMonthlyData struct {
	ID          uuid.UUID `db:"id" json:"id"`
	FullName    string    `db:"full_name" json:"full_name"`
	Salary      *float64  `db:"salary" json:"salary"`
	Status      string    `db:"status" json:"status"`
	JoiningDate time.Time `db:"joining_date" json:"joining_date"`
}

type ExistingRun struct {
	ID     uuid.UUID `db:"id"`
	Status string    `db:"status"`
}

func (r *Repository) GetExitstingpayload(input struct {
	Month int `json:"month" validate:"required"`
	Year  int `json:"year" validate:"required"`
}) (ExistingRun, error) {
	var data ExistingRun
	query := `SELECT id, status 
        FROM Tbl_Payroll_run 
        WHERE month=$1 AND year=$2`
	err := r.DB.Get(&data, query, input.Month, input.Year)
	return data, err
}

// Get Working Days
func (r *Repository) GetCompanyCurrWorkingDays() int {
	var workingDays int
	query := `
		SELECT working_days_per_month 
		FROM Tbl_Company_Settings 
		ORDER BY created_at DESC 
		LIMIT 1
	`
	err := r.DB.Get(&workingDays, query)
	if err != nil {
		workingDays = 22 // fallback default
	}
	return workingDays
}

// Get All Employee Base on MONTH and YEARS also JOININIG DATES

func (r *Repository) GetEmployeeByMonthAndYear(input struct {
	Month int `json:"month" validate:"required"`
	Year  int `json:"year" validate:"required"`
}) ([]EmpMonthlyData, error) {

	var employees []EmpMonthlyData

	query := `
		SELECT id, full_name, salary, status, joining_date
		FROM tbl_employee
		WHERE status = 'active'
		AND (
			EXTRACT(YEAR FROM joining_date) < $1
			OR (EXTRACT(YEAR FROM joining_date) = $1 
			    AND EXTRACT(MONTH FROM joining_date) <= $2)
		)
	`

	err := r.DB.Select(&employees, query, input.Year, input.Month)
	if err != nil {
		return nil, err
	}
	return employees, nil
}
