package repositories

import (
	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/jmoiron/sqlx"
)

func (r *Repository) GetCompanySettings(settings *models.CompanySettings) error {
	return r.DB.Get(settings, `
		SELECT id, working_days_per_month, allow_manager_add_leave,
		       company_name, primary_color, secondary_color,
		       birthday_message_template, created_at, updated_at
		FROM Tbl_Company_Settings LIMIT 1`)
}

func (r *Repository) UpdateCompanySettings(tx *sqlx.Tx, input models.CompanyField) error {
	_, err := tx.Exec(`
		UPDATE Tbl_Company_Settings
		SET working_days_per_month        = $1,
		    allow_manager_add_leave       = $2,
		    company_name                  = $3,
		    primary_color                 = $4,
		    secondary_color               = $5,
		    birthday_message_template     = CASE WHEN $6 = '' THEN birthday_message_template ELSE $6 END,
		    updated_at                    = NOW()
	`,
		input.WorkingDaysPerMonth,
		input.AllowManagerAddLeave,
		input.CompanyName,
		input.PrimaryColor,
		input.SecondaryColor,
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
