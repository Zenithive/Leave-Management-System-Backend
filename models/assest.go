package models

import (
	"time"

	"github.com/google/uuid"
)

// EQUIPMENT

type EquipmentCategoryRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=50"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=255"`
}
type EquipmentCategoryRes struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name" validate:"required,min=2,max=50"`
	Description string    `db:"description" json:"description,omitempty" validate:"omitempty,max=255"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type EquipmentRequest struct {
	ID                *uuid.UUID `json:"id,omitempty" validate:"omitempty,uuid4"`
	Name              string     `json:"name" validate:"required,min=2,max=100"`
	CategoryID        uuid.UUID  `json:"category_id" validate:"required,uuid4"`
	IsShared          *bool      `json:"is_shared,omitempty"`
	Price             float64    `json:"price" validate:"min=0"`
	TotalQuantity     int        `json:"total_quantity" validate:"required,min=0"`
	RemainingQuantity *int       `json:"remaining_quantity"`
	PurchaseDate      *time.Time `json:"purchase_date,omitempty"` // Optional
}

type EquipmentRes struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	Name              string     `db:"name" json:"name"`
	CategoryID        uuid.UUID  `db:"category_id" json:"category_id"`
	IsShared          bool       `db:"is_shared" json:"is_shared"`
	Price             float64    `db:"price" json:"price"`
	TotalQuantity     int        `db:"total_quantity" json:"total_quantity"`
	RemainingQuantity int        `db:"remaining_quantity" json:"remaining_quantity"`
	PurchaseDate      *time.Time `db:"purchase_date" json:"purchase_date,omitempty"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
}

// AssignEquipmentRequest - used when assigning equipment to an employee
type AssignEquipmentRequest struct {
	EmployeeID  uuid.UUID `json:"employee_id" validate:"required"`
	EquipmentID uuid.UUID `json:"equipment_id" validate:"required"`
	Quantity    int       `json:"quantity" validate:"required,min=1"`
	AssignedBy  uuid.UUID `json:"assigned_by" validate:"required"`
}
type AssignEquipmentResponse struct {
	AssignmentID   uuid.UUID  `db:"assignment_id" json:"assignment_id"`
	EmployeeID     uuid.UUID  `db:"employee_id" json:"employee_id"`
	EmployeeName   string     `db:"employee_name" json:"employee_name"`
	EmployeeEmail  string     `db:"employee_email" json:"employee_email"`
	EquipmentID    uuid.UUID  `db:"equipment_id" json:"equipment_id"`
	EquipmentName  string     `db:"equipment_name" json:"equipment_name"`
	PurchaseDate   *time.Time `db:"purchase_date" json:"purchase_date,omitempty"`
	Quantity       int        `db:"quantity" json:"quantity"`
	AssignedAt     time.Time  `db:"assigned_at" json:"assigned_at"`
	ApprovedByName string     `db:"approved_by_name" json:"approved_by_name"`
}

// RemoveEquipmentRequest - used when removing/returning equipment from an employee
type RemoveEquipmentRequest struct {
	EmployeeID  uuid.UUID `json:"employee_id" validate:"required"`  // Employee to remove equipment from
	EquipmentID uuid.UUID `json:"equipment_id" validate:"required"` // Equipment being removed
}

// UpdateAssignmentRequest - used for both reassigning equipment and updating quantity
type UpdateAssignmentRequest struct {
	FromEmployeeID uuid.UUID  `json:"from_employee_id" validate:"required"`
	ToEmployeeID   *uuid.UUID `json:"to_employee_id,omitempty"`
	EquipmentID    uuid.UUID  `json:"equipment_id" validate:"required"`
	Quantity       int        `json:"quantity" validate:"required,min=1"`
	AssignedBy     uuid.UUID  `json:"assigned_by" validate:"required"` // Add this
}
