package models

import (
	"fmt"
	"strings"
)

const (
	SortEmail          = "email"
	SortRole           = "role"
	SortTotalLeaves    = "total_leaves"
	SortPaidLeaves     = "paid_leaves"
	SortUnpaidLeaves   = "unpaid_leaves"
	SortEarlyLeaves    = "early_leaves"
	SortAccruedLeaves  = "accrued_leaves"
	SortBalanceLeaves  = "balance_leaves"
	SortUsedLeaves     = "used_leaves"
	DefaultOrderColumn = "e.full_name"
)

type LeaveReportFilter struct {
	FromMonth int
	FromYear  int

	ToMonth int
	ToYear  int

	Search string
	Role   string

	SortBy    string
	SortOrder string
}

func BuildLeaveReportOrder(sortBy, sortOrder string) string {

	orderCol := DefaultOrderColumn

	switch sortBy {
	case SortEmail:
		orderCol = "e.email"

	case SortRole:
		orderCol = "role"

	case SortTotalLeaves:
		orderCol = "total_leaves"

	case SortPaidLeaves:
		orderCol = "paid_leaves"

	case SortUnpaidLeaves:
		orderCol = "unpaid_leaves"

	case SortEarlyLeaves:
		orderCol = "early_leaves"

	case SortAccruedLeaves:
		orderCol = "accrued_leaves"

	case SortBalanceLeaves:
		orderCol = "balance_leaves"

	case SortUsedLeaves:
		orderCol = "used_leaves"
	}

	dir := "ASC"

	if strings.ToUpper(sortOrder) == "DESC" {
		dir = "DESC"
	}

	return fmt.Sprintf(" ORDER BY %s %s", orderCol, dir)
}

// LeaveReportRecord is a single employee row in any leave report response.
type LeaveReportRecord struct {
	EmployeeID    string  `json:"employee_id"    db:"employee_id"`
	EmployeeName  string  `json:"employee_name"  db:"employee_name"`
	Email         string  `json:"email"          db:"email"`
	Role          string  `json:"role"           db:"role"`
	AccruedLeaves float64 `json:"accrued_leaves" db:"accrued_leaves"`
	UsedLeaves    float64 `json:"used_leaves"    db:"used_leaves"`
	BalanceLeaves float64 `json:"balance_leaves" db:"balance_leaves"`
	TotalLeaves   float64 `json:"total_leaves"   db:"total_leaves"`
	PaidLeaves    float64 `json:"paid_leaves"    db:"paid_leaves"`
	UnpaidLeaves  float64 `json:"unpaid_leaves"  db:"unpaid_leaves"`
	EarlyLeaves   float64 `json:"early_leaves"   db:"early_leaves"`
}

// LeaveReportResponse is the unified API response for all leave report types.
type LeaveReportResponse struct {
	ReportType string              `json:"report_type"`
	FromMonth  int                 `json:"from_month"`
	FromYear   int                 `json:"from_year"`
	ToMonth    int                 `json:"to_month"`
	ToYear     int                 `json:"to_year"`
	Total      int                 `json:"total"`
	Records    []LeaveReportRecord `json:"records"`
}

type LeaveReportRequest struct {
	ReportType string // "monthly" | "yearly" | "range"
	Month      int    // used by monthly
	Year       int    // used by monthly and yearly
	FromMonth  int    // used by range
	FromYear   int    // used by range
	ToMonth    int    // used by range
	ToYear     int    // used by range

	// Filters & sorting
	Search    string // search by name or email (partial, case-insensitive)
	Role      string // filter by role: EMPLOYEE, INTERN, HR, ADMIN, SUPERADMIN, MANAGER
	SortBy    string // name | email | role | total_leaves | paid_leaves | unpaid_leaves | early_leaves
	SortOrder string // asc | desc
}
