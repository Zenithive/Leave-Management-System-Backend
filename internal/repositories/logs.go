package repositories

import (
	"errors"
	"time"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/jmoiron/sqlx"
)

func (r *Repository) AddLog(data *models.Common, q *sqlx.Tx) error {
	_, err := q.Exec("INSERT INTO tbl_log (from_user_id, action, component) VALUES ($1, $2, $3)", data.FromUserID, data.Action, data.Component)
	return err
}

func (r *Repository) GetLogs(dateThreshold time.Time) ([]models.LogResponse, error) {
	query := `
		SELECT 
			l.id,
			e.full_name as user_name,
			l.action,
			l.component,
			l.created_at
		FROM tbl_log l
		JOIN Tbl_Employee e ON l.from_user_id = e.id
		WHERE l.created_at >= $1
		ORDER BY l.created_at DESC
	`

	rows, err := r.DB.Query(query, dateThreshold)
	if err != nil {
		return nil, errors.New("failed to get logs" + err.Error())
	}
	defer rows.Close()

	var logs []models.LogResponse
	for rows.Next() {
		var log models.LogResponse
		err := rows.Scan(
			&log.ID,
			&log.UserName,
			&log.Action,
			&log.Component,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, errors.New("Failed to scan log data" + err.Error())
		}
		logs = append(logs, log)
	}
	return logs, nil
}
