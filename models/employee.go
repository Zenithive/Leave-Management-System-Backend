package models

import (
	"time"

	"github.com/google/uuid"
)

type Employee struct {
}

// EmployeeFilterParams - Query parameters for filtering and pagination
type EmployeeFilterParams struct {
	// Pagination
	Page     int `form:"page"`
	PageSize int `form:"page_size"`

	// Filters
	Search      string `form:"search"` // Unified search: searches employee name, email, and manager name
	Role        string `form:"role"`
	Designation string `form:"designation"`
	Status      string `form:"status"` // active/deactive

	// Sorting
	SortBy    string `form:"sort_by"`    // joining_date, salary, name, manager_name
	SortOrder string `form:"sort_order"` // asc/desc
}

// PaginatedEmployeeResponse - Response with pagination metadata
type PaginatedEmployeeResponse struct {
	Employees  []EmployeeResponse `json:"employees"`
	TotalCount int                `json:"total_count"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// EmployeeResponse is used for GET list and GET by ID (no password, safe for API response).
// Use this for GetAllEmployees, GetEmployeeByID, GetEmployeesByManagerID.
type EmployeeResponse struct {
	ID              uuid.UUID  `json:"id"`
	FullName        string     `json:"full_name"`
	Email           string     `json:"email"`
	Status          string     `json:"status"`
	Role            string     `json:"role"`
	ManagerID       *uuid.UUID `json:"manager_id,omitempty"`
	ManagerName     *string    `json:"manager_name,omitempty"`
	DesignationID   *uuid.UUID `json:"designation_id,omitempty"`
	DesignationName *string    `json:"designation_name,omitempty"`
	Salary          *float64   `json:"salary,omitempty"` // omitted for HR in list; present for admin/detail
	JoiningDate     *time.Time `json:"joining_date,omitempty"`
	EndingDate      *time.Time `json:"ending_date,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}
