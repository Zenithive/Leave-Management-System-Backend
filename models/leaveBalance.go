package models

import "github.com/google/uuid"

// Balance is the API response shape for a single leave type balance.
type Balance struct {
	LeaveTypeID int     `json:"leave_type_id"`
	LeaveType   string  `json:"leave_type"`
	Opening     float64 `json:"opening"`
	Accrued     float64 `json:"accrued"`
	Used        float64 `json:"used"`
	Adjusted    float64 `json:"adjusted"`
	Total       float64 `json:"total"`
	Available   float64 `json:"available"`
}

// LeaveTypeData is used by the repo to return leave type + entitlement info.
type LeaveTypeData struct {
	LeaveTypeID        int      `db:"leave_type_id"`
	LeaveTypeName      string   `db:"leave_type_name"`
	DefaultEntitlement float64  `db:"default_entitlement"`
	InternEntitlement  *float64 `db:"intern_entitlement"`
}

// BalanceData is the raw DB row from Tbl_Leave_balance.
type BalanceData struct {
	LeaveTypeID int     `db:"leave_type_id"`
	Opening     float64 `db:"opening"`
	Accrued     float64 `db:"accrued"`
	Used        float64 `db:"used"`
	Adjusted    float64 `db:"adjusted"`
	Closing     float64 `db:"closing"`
}

// LeaveBalanceForAdjustment is the locked row fetched during an adjustment transaction.
type LeaveBalanceForAdjustment struct {
	ID          uuid.UUID `db:"id"`
	Opening     float64   `db:"opening"`
	Accrued     float64   `db:"accrued"`
	Used        float64   `db:"used"`
	Adjusted    float64   `db:"adjusted"`
	Closing     float64   `db:"closing"`
	EmployeeID  uuid.UUID `db:"employee_id"`
	LeaveTypeID int       `db:"leave_type_id"`
	Year        int       `db:"year"`
}

// AdjustLeaveBalanceInput is the request body for POST /api/leave-balances/:id/adjust
type AdjustLeaveBalanceInput struct {
	LeaveTypeID int     `json:"leave_type_id" validate:"required"`
	Quantity    float64 `json:"quantity" validate:"required"` // positive = add, negative = deduct
	Reason      string  `json:"reason" validate:"required"`
}
