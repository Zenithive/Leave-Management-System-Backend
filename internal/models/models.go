package models

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// ----------------- ROLE -----------------
type RoleInput struct {
	Type string `json:"type" validate:"required"`
}

// Role is the response model for a single role entry.
type Role struct {
	ID   int    `json:"id" db:"id"`
	Type string `json:"type" db:"type"`
}

// ----------------- EMPLOYEE -----------------

// EmployeeInput is used for create employee (API input + validation).
type EmployeeInput struct {
	ID              *uuid.UUID `json:"id,omitempty"` // optional UUID
	FullName        string     `json:"full_name" validate:"required"`
	Email           string     `json:"email" validate:"required,email"`
	Role            string     `json:"role" validate:"required"`
	Password        string     `json:"password,omitempty"`       // optional - auto-generated if not provided
	ManagerID       *uuid.UUID `json:"manager_id,omitempty"`     // optional UUID
	DesignationID   *uuid.UUID `json:"designation_id,omitempty"` // optional UUID
	Salary          *float64   `json:"salary,omitempty"`         // optional
	JoiningDate     *time.Time `json:"joining_date,omitempty"`   // optional
	BirthDate       *time.Time `json:"birth_date,omitempty"`     // optional
	EndingDate      *time.Time `json:"ending_date,omitempty"`    // optional
	Status          *string    `json:"status,omitempty"`         // optional, new field
	CreatedAt       *time.Time `json:"created_at,omitempty"`     // optional
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`     // optional
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	ManagerName     *string    `json:"manager_name,omitempty"`     // optional
	DesignationName *string    `json:"designation_name,omitempty"` // optional
}

// ----------------- PAYROLL RUN -----------------
type PayrollRunInput struct {
	Month  int     `json:"month" validate:"required"`
	Year   int     `json:"year" validate:"required"`
	Status *string `json:"status,omitempty"`
}

// ----------------- PAYSLIP -----------------
type PayslipInput struct {
	PayrollRunID    uuid.UUID `json:"payroll_run_id" validate:"required"`
	EmployeeID      uuid.UUID `json:"employee_id" validate:"required"`
	BasicSalary     *float64  `json:"basic_salary,omitempty"`
	WorkingDays     *int      `json:"working_days,omitempty"`
	UnpaidLeaves    *float64  `json:"unpaid_leaves,omitempty"`
	PaidLeaves      *float64  `json:"paid_leaves,omitempty"`
	DeductionAmount *float64  `json:"deduction_amount,omitempty"`
	NetSalary       *float64  `json:"net_salary,omitempty"`
	PdfPath         *string   `json:"pdf_path,omitempty"`
}
type PayrollEmployeeResponse struct {
	EmployeeName string  `json:"employee_name"`
	BasicSalary  float64 `json:"basic_salary"`
	WorkingDays  float64 `json:"working_days"` // float64 expected
	UnpaidLeaves float64 `json:"unpaid_leaves"`
	PaidLeaves   float64 `json:"paid_leaves"`
	Deductions   float64 `json:"deductions"`
	NetSalary    float64 `json:"net_salary"`
}

// -------------------Loing input-----------------------
type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// ----------------- AUDIT -----------------
type AuditInput struct {
	ActorID  uuid.UUID  `json:"actor_id" validate:"required"`
	Action   *string    `json:"action,omitempty"`
	Entity   *string    `json:"entity,omitempty"`
	EntityID *uuid.UUID `json:"entity_id,omitempty"`
	Metadata *string    `json:"metadata,omitempty"` // JSON as string
}

type FullPayslipResponse struct {
	PayslipID       uuid.UUID `json:"payslip_id"`
	EmployeeID      uuid.UUID `json:"employee_id"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	Month           int       `json:"month"` // from Payroll_Run
	Year            int       `json:"year"`
	BasicSalary     float64   `json:"basic_salary"`
	WorkingDays     int       `json:"working_days"`
	PaidLeaves      float64   `json:"paid_leaves"`
	UnpaidLeaves    float64   `json:"unpaid_leaves"`
	DeductionAmount float64   `json:"deduction_amount"`
	NetSalary       float64   `json:"net_salary"`
	PDFPath         string    `json:"pdf_path"`
	CalculationText string    `json:"calculation"`
	CreatedAt       string    `json:"created_at"`
}

var Validate *validator.Validate

func InitValidator() *validator.Validate {
	Validate = validator.New()
	return Validate
}

// CompanySettings struct mapping the DB table
type CompanySettings struct {
	ID                   uuid.UUID `db:"id" json:"id"`
	WorkingDaysPerMonth  int       `db:"working_days_per_month" json:"working_days_per_month"`
	AllowManagerAddLeave bool      `db:"allow_manager_add_leave" json:"allow_manager_add_leave"`
	CreatedAt            string    `db:"created_at" json:"created_at"`
	UpdatedAt            string    `db:"updated_at" json:"updated_at"`

	CompanyName    string `db:"company_name" json:"company_name"`
	LogoPath       string `db:"logo_path" json:"logo_path"`
	PrimaryColor   string `db:"primary_color" json:"primary_color"`
	SecondaryColor string `db:"secondary_color" json:"secondary_color"`

	// Birthday message template — supports {name}, {date}, {age} placeholders
	BirthdayMessageTemplate string `db:"birthday_message_template" json:"birthday_message_template"`
}

type CompanyField struct {
	WorkingDaysPerMonth     int    `form:"WorkingDaysPerMonth" json:"working_days_per_month"`
	AllowManagerAddLeave    bool   `form:"AllowManagerAddLeave" json:"allow_manager_add_leave"`
	CompanyName             string `form:"CompanyName" json:"company_name"`
	PrimaryColor            string `form:"PrimaryColor" json:"primary_color"`
	SecondaryColor          string `form:"SecondaryColor" json:"secondary_color"`
	LogoPath                string `json:"logo_path"`
	BirthdayMessageTemplate string `form:"BirthdayMessageTemplate" json:"birthday_message_template"`
}
