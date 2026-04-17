package models

import (
	"time"

	"github.com/google/uuid"
)

type Employee struct {
}

// EmployeeFilterParams - Query parameters for filtering, sorting, and pagination
type EmployeeFilterParams struct {
	// Pagination
	Page     int `form:"page"`
	PageSize int `form:"page_size"`

	// Filters
	Search      string `form:"search"`      // searches name, email, manager name
	Role        string `form:"role"`        // exact: EMPLOYEE, MANAGER, HR, ADMIN, SUPERADMIN
	Designation string `form:"designation"` // exact match on designation_name
	Status      string `form:"status"`      // active / deactive
	Manager     string `form:"manager"`     // exact match on manager full_name

	// Sorting
	SortBy    string `form:"sort_by"`    // name|email|joining_date|ending_date|salary|birth_date|manager_name|role|status
	SortOrder string `form:"sort_order"` // asc / desc
}

// TeamFilterParams - Query parameters for manager's team list
type TeamFilterParams struct {
	SortBy    string `form:"sort_by"`    // name|birth_date|joining_date|ending_date|email
	SortOrder string `form:"sort_order"` // asc / desc
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
	BirthDate       *time.Time `json:"birth_date"`
	EndingDate      *time.Time `json:"ending_date,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type BirthdayEntry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

// BirthdayEmployee holds minimal employee data for birthday processing.
type BirthdayEmployee struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Email            string     `json:"email"`
	BirthDate        *time.Time `json:"birth_date,omitempty"`
	Age              int        `json:"age"`
	Status           string     `json:"status"`
	RemainingDays    int        `json:"remaining_days"`
	RemainingHours   int        `json:"remaining_hours,omitempty"`
	RemainingMinutes int        `json:"remaining_minutes,omitempty"`
}
