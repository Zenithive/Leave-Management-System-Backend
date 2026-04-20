package repositories

import (
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
