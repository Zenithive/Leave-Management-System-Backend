package repositories

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

// Get All Leave Timing
func (r *Repository) GetLeaveTiming() ([]models.LeaveTimingResponse, error) {
	var data []models.LeaveTimingResponse
	query := `
		SELECT id, type, timing, created_at, updated_at
		FROM Tbl_Half
		ORDER BY id
	`
	err := r.DB.Select(&data, query)

	return data, err
}

// Get All Leave Timming

// Get Leave Timing By ID
func (r *Repository) GetLeaveTimingByID(id int) (*models.LeaveTimingResponse, error) {
	var data models.LeaveTimingResponse

	query := `
		SELECT *
		FROM Tbl_Half
		WHERE id = $1
	`

	err := r.DB.Get(&data, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return &data, nil
}

func (r *Repository) UpdateLeaveTiming(tx *sqlx.Tx, id int, timing string) error {
	query := `
		UPDATE Tbl_Half
		SET timing = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	res, err := tx.Exec(query, timing, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}
