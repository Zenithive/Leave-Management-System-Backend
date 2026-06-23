package models

import (
	"time"

	"github.com/google/uuid"
)

// ----------------- LOG -----------------
type LogResponse struct {
	ID        int       `json:"id" db:"id"`
	UserName  string    `json:"user_name" db:"user_name"`
	Action    string    `json:"action" db:"action"`
	Component string    `json:"component" db:"component"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Common struct {
	Component  string
	Action     string
	FromUserID uuid.UUID
}

func NewCommon(component, action string, fromUserID uuid.UUID) *Common {
	return &Common{
		Component:  component,
		Action:     action,
		FromUserID: fromUserID,
	}
}
