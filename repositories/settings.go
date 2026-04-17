package repositories

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

func (r *Repository) GetCompanySettings(settings *models.CompanySettings) error {
	return r.DB.Get(settings, `SELECT * FROM Tbl_Company_Settings LIMIT 1`)
}

func (r *Repository) UpdateCompanySettings(tx *sqlx.Tx, input models.CompanyField, logoPath string) error {
	_, err := tx.Exec(`
		UPDATE Tbl_Company_Settings
		SET working_days_per_month        = $1,
		    allow_manager_add_leave       = $2,
		    company_name                  = $3,
		    primary_color                 = $4,
		    secondary_color               = $5,
		    logo_path                     = COALESCE(NULLIF($6, ''), logo_path),
		    birthday_message_template     = CASE WHEN $7 = '' THEN birthday_message_template ELSE $7 END,
		    updated_at                    = NOW()
	`,
		input.WorkingDaysPerMonth,
		input.AllowManagerAddLeave,
		input.CompanyName,
		input.PrimaryColor,
		input.SecondaryColor,
		logoPath,
		input.BirthdayMessageTemplate,
	)
	return err
}

// GetBirthdayMessageTemplate returns the raw template string from settings.
func (r *Repository) GetBirthdayMessageTemplate() (string, error) {
	var tmpl string
	err := r.DB.Get(&tmpl, `SELECT birthday_message_template FROM Tbl_Company_Settings LIMIT 1`)
	return tmpl, err
}

// RenderBirthdayMessage replaces supported placeholders in the template:
//
//	{name}  → employee full name
//	{date}  → birth date formatted as "2 January"
//	{age}   → calculated age in years (requires birth_date)
func RenderBirthdayMessage(template, name string, birthDate *time.Time) string {
	msg := template
	msg = strings.ReplaceAll(msg, "{name}", name)

	if birthDate != nil {
		msg = strings.ReplaceAll(msg, "{date}", birthDate.Format("2 January"))
		age := calculateAge(*birthDate)
		msg = strings.ReplaceAll(msg, "{age}", fmt.Sprintf("%d", age))
	} else {
		// Remove placeholders gracefully if no birth date
		msg = strings.ReplaceAll(msg, "{date}", "")
		msg = strings.ReplaceAll(msg, "{age}", "")
	}

	return msg
}

func calculateAge(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	// Adjust if birthday hasn't occurred yet this year
	if now.YearDay() < birthDate.YearDay() {
		age--
	}
	return age
}

// BirthdayEmployee holds minimal employee data for birthday processing.
type BirthdayEmployee struct {
	ID        string
	Name      string
	Email     string
	BirthDate *time.Time
}

// GetTodayBirthdays returns all active employees whose birth_date month+day matches today.
func (r *Repository) GetTodayBirthdays() ([]BirthdayEmployee, error) {
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

	var result []BirthdayEmployee
	for rows.Next() {
		var emp BirthdayEmployee
		if err := rows.Scan(&emp.ID, &emp.Name, &emp.Email, &emp.BirthDate); err != nil {
			return nil, err
		}
		result = append(result, emp)
	}
	return result, nil
}
