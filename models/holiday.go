package models

import "time"

type Holiday struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name" binding:"required"`
	Date      time.Time `json:"date" db:"date" binding:"required"`
	Day       string    `json:"day" db:"day"`
	Type      string    `json:"type" db:"type"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

var HolidayInput struct {
	Name string    `json:"name" binding:"required,min=2,max=100"`
	Date time.Time `json:"date" binding:"required"`
	Type string    `json:"type" binding:"omitempty,oneof=HOLIDAY OPTIONAL"`
}
