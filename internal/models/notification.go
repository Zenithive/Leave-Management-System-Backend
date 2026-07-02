package models

import "github.com/google/uuid"

type Recipient struct {
	ID       uuid.UUID `db:"id"`
	FullName string    `db:"full_name"`
	Email    string    `db:"email"`
	Role     string    `db:"role"`
}
