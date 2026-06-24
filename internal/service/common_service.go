package service

import (
	"fmt"
	"math"
	"time"

	"github.com/Zenithive/LeaveManagementSystem/internal/repositories"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ValidateUnpaidLeaveApplication checks if an employee can apply for unpaid leave.
// Business Rule: Employees cannot apply for unpaid leave if they have:
//  1. Any paid leave balance > 0, OR
//  2. Any pending/manager-approved paid leaves
//
// Returns an error if validation fails, nil if validation passes.
func ValidateUnpaidLeaveApplication(repo *repositories.Repository, tx *sqlx.Tx, employeeID uuid.UUID, leaveTypeID int) error {
	// First, check if the leave type being applied is unpaid or early leave
	var result struct {
		IsPaid  bool  `db:"is_paid"`
		IsEarly *bool `db:"is_early"`
	}
	err := tx.Get(&result, `SELECT is_paid, is_early FROM Tbl_Leave_Type WHERE id=$1`, leaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to fetch leave type: %w", err)
	}

	// If the leave type is paid, no validation needed
	if result.IsPaid {
		return nil
	}

	// If the leave type is early leave, skip unpaid leave validation
	if result.IsEarly != nil && *result.IsEarly {
		return nil
	}

	// If applying for unpaid leave, check if employee has any paid leave balance
	totalPaidBalance, err := repo.GetTotalPaidLeaveBalance(tx, employeeID)
	if err != nil {
		return fmt.Errorf("failed to fetch paid leave balance: %w", err)
	}

	// Check if employee has any pending paid leaves
	totalPendingPaidDays, err := repo.GetTotalPendingPaidLeaveDays(tx, employeeID)
	if err != nil {
		return fmt.Errorf("failed to fetch pending paid leave days: %w", err)
	}

	// Validation logic with detailed error messages
	if totalPaidBalance > 0 && totalPendingPaidDays > 0 {

		return fmt.Errorf("cannot apply for unpaid leave. You have %.1f days of paid leave balance remaining and %.1f days of pending paid leaves. Please use paid leave first", totalPaidBalance, totalPendingPaidDays)
	}

	if totalPaidBalance > 0 {
		return fmt.Errorf("cannot apply for unpaid leave. You have %.1f days of paid leave balance remaining. Please use paid leave first", totalPaidBalance)
	}

	if totalPendingPaidDays > 0 {
		return fmt.Errorf("cannot apply for unpaid leave. You have %.1f days of pending paid leave applications. Please wait for approval or use those paid leaves first", totalPendingPaidDays)
	}
	return nil
}

func CalculateWorkingDays(Query *repositories.Repository, tx *sqlx.Tx, start, end time.Time, leaveTiming time.Time) (float64, error) {
	// 1️ Validate date range
	if end.Before(start) {
		return 0, fmt.Errorf("end date cannot be before start date")
	}

	// Normalize dates to midnight UTC to avoid timezone issues
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)

	// 2️ Fetch holidays within range
	holidays, err := Query.GetByFilterHolidayBetweenTwoDates(tx, start, end)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch holidays: %v", err)
	}

	// Convert slice to a map for O(1) lookup
	holidayMap := make(map[string]bool)
	for _, h := range holidays {
		holidayMap[h.Format("2006-01-02")] = true
	}

	// 3️ Count working days
	workingDays := 0
	var workingDaysList []string // For debugging

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dayStr := d.Format("2006-01-02")
		weekday := d.Weekday()

		// Skip Saturday and Sunday
		if weekday == time.Saturday || weekday == time.Sunday {
			fmt.Printf("DEBUG: Skipping weekend: %s (%s)\n", dayStr, weekday)
			continue
		}

		// Skip holidays
		if holidayMap[dayStr] {
			fmt.Printf("DEBUG: Skipping holiday: %s\n", dayStr)
			continue
		}

		workingDays++
		workingDaysList = append(workingDaysList, fmt.Sprintf("%s (%s)", dayStr, weekday))
	}

	fmt.Printf("DEBUG: Working days calculated: %d - Days: %v\n", workingDays, workingDaysList)
	return float64(workingDays), nil
}

// applyTimingMultiplier applies the half/full-day multiplier to a pre-calculated
// base working-day count. Extracted so it can be unit-tested without a DB.
// timingID: 1 = First Half (×0.5), 2 = Second Half (×0.5), 3 = Full Day (×1.0)
func applyTimingMultiplier(baseDays float64, timingID int) (float64, error) {
	fmt.Println("===============", timingID)
	switch timingID {
	case 1, 2:
		return baseDays * 0.5, nil
	case 3:
		return baseDays, nil
	default:
		return 0, fmt.Errorf("invalid timing ID: %d. Must be 1 (First Half), 2 (Second Half), or 3 (Full Day)", timingID)
	}
}

// CalculateWorkingDaysWithTiming calculates working days based on timing type
// timingID: 1 = First Half (0.5 days), 2 = Second Half (0.5 days), 3 = Full Day (1.0 days)
func CalculateWorkingDaysWithTiming(Query *repositories.Repository, tx *sqlx.Tx, start, end time.Time, timingID int, leaveTiming time.Time) (float64, error) {
	// First calculate the base working days
	baseDays, err := CalculateWorkingDays(Query, tx, start, end, leaveTiming)
	if err != nil {
		return 0, err
	}

	// Early-leave timing string overrides the half-day multiplier
	if !leaveTiming.IsZero() {
		return baseDays, nil
	}

	return applyTimingMultiplier(baseDays, timingID)
}

type LeaveSummary struct {
	PaidDays    float64
	UnpaidDays  float64
	EarlyLeaves float64
}

func CalculateAbsentDaysForMonth(db *sqlx.DB, employeeID uuid.UUID, month, year int) LeaveSummary {
	// 1. Define time boundaries
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	// 2. Optimization: Fetch holidays once for the period
	var holidays []time.Time
	_ = db.Select(&holidays, `SELECT date FROM Tbl_Holiday WHERE date >= $1 AND date <= $2`, firstDay, lastDay)

	holidayMap := make(map[string]bool)
	for _, h := range holidays {
		holidayMap[h.Format("2006-01-02")] = true
	}

	// 3. Updated SQL: Removed "is_paid = false" to get ALL approved leaves
	type LeaveRecord struct {
		StartDate  time.Time `db:"start_date"`
		EndDate    time.Time `db:"end_date"`
		Days       float64   `db:"days"`
		IsPaid     bool      `db:"is_paid"` // Now fetching this field
		TimingType *string   `db:"timing_type"`
	}

	var leaves []LeaveRecord
	err := db.Select(&leaves, `
        SELECT l.start_date, l.end_date, l.days, lt.is_paid, h.type as timing_type
        FROM Tbl_Leave l
        JOIN Tbl_Leave_type lt ON l.leave_type_id = lt.id
        LEFT JOIN Tbl_Half h ON l.half_id = h.id
        WHERE l.employee_id=$1 
        AND l.status='APPROVED'
        AND l.start_date <= $2
        AND l.end_date >= $3
        AND (lt.is_early IS NULL OR lt.is_early IS NOT TRUE)
    `, employeeID, lastDay, firstDay)
	if err != nil {
		fmt.Printf("Error fetching leaves: %v\n", err)
		return LeaveSummary{}
	}

	summary := LeaveSummary{}

	// 4a. Count early leaves separately (for display only — no deduction)
	var earlyLeaveCount float64
	_ = db.Get(&earlyLeaveCount, `
		SELECT COALESCE(SUM(l.days), 0)
		FROM Tbl_Leave l
		JOIN Tbl_Leave_type lt ON l.leave_type_id = lt.id
		WHERE l.employee_id = $1
		AND l.status = 'APPROVED'
		AND lt.is_early = TRUE
		AND l.start_date <= $2
		AND l.end_date >= $3
	`, employeeID, lastDay, firstDay)
	summary.EarlyLeaves = earlyLeaveCount

	// 4b. Calculate absent days (paid/unpaid) — early leaves already excluded from `leaves` slice
	for _, leave := range leaves {
		overlapStart := leave.StartDate
		if overlapStart.Before(firstDay) {
			overlapStart = firstDay
		}

		overlapEnd := leave.EndDate
		if overlapEnd.After(lastDay) {
			overlapEnd = lastDay
		}

		actualDaysInMonth := 0.0
		for d := overlapStart; !d.After(overlapEnd); d = d.AddDate(0, 0, 1) {
			// Skip weekends and holidays
			if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday || holidayMap[d.Format("2006-01-02")] {
				continue
			}

			if leave.StartDate.Equal(leave.EndDate) && leave.Days < 1.0 {
				actualDaysInMonth += leave.Days
			} else {
				actualDaysInMonth += 1.0
			}
		}

		// 5. Categorize based on IsPaid flag
		if leave.IsPaid {
			summary.PaidDays += actualDaysInMonth
		} else {
			summary.UnpaidDays += actualDaysInMonth
		}
	}

	return summary
}

// LeaveBalanceData represents raw balance data from database
type LeaveBalanceData struct {
	LeaveTypeID int
	Opening     float64
	Accrued     float64
	Used        float64
	Adjusted    float64
	Closing     float64
}

// LeaveTypeData represents leave type information
type LeaveTypeData struct {
	LeaveTypeID        int
	LeaveTypeName      string
	DefaultEntitlement float64
	InternEntitlement  *float64
}

// CalculatedBalance represents the calculated leave balance result
type CalculatedBalance struct {
	LeaveTypeID int     `json:"leave_type_id"`
	LeaveType   string  `json:"leave_type"`
	Opening     float64 `json:"opening"`
	Accrued     float64 `json:"accrued"`
	Used        float64 `json:"used"`
	Adjusted    float64 `json:"adjusted"`
	Total       float64 `json:"total"`
	Available   float64 `json:"available"`
}

// CalculateLeaveBalances calculates leave balances using map-based approach.
// Only leave types that have an actual balance row in the DB are returned.
func CalculateLeaveBalances(leaveTypes []LeaveTypeData, balanceRecords []LeaveBalanceData) []CalculatedBalance {
	// Build a name lookup: leave_type_id -> leave_type_name
	nameMap := make(map[int]string, len(leaveTypes))
	for _, lt := range leaveTypes {
		nameMap[lt.LeaveTypeID] = lt.LeaveTypeName
	}

	var calculatedBalances []CalculatedBalance

	// Only iterate over actual DB balance records — no synthetic entries
	for _, balance := range balanceRecords {
		name := nameMap[balance.LeaveTypeID]
		total := balance.Opening

		calculatedBalances = append(calculatedBalances, CalculatedBalance{
			LeaveTypeID: balance.LeaveTypeID,
			LeaveType:   name,
			Opening:     balance.Opening,
			Accrued:     balance.Accrued,
			Used:        balance.Used,
			Adjusted:    balance.Adjusted,
			Total:       total,
			Available:   balance.Closing,
		})
	}

	return calculatedBalances
}

// validateLeaveTiming

func CalculateProratedLeave(yearlyLeave int, joinMonth int) int {
	if joinMonth < 1 || joinMonth > 12 {
		return 0
	}

	// Remaining months including joining month
	remainingMonths := 12 - joinMonth + 1

	// Prorated calculation
	prorated := (float64(yearlyLeave) * float64(remainingMonths)) / 12

	// Round down
	return int(math.Floor(prorated))
}

func ValidateLeaveTiming(leaveTiming string) (time.Time, error) {

	// Expected format: "18:02"
	t, err := time.Parse("15:04", leaveTiming)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid leave_timing format, expected HH:MM")
	}

	minTime := 10*60 + 0 // 10:00 AM
	maxTime := 19*60 + 0 // 7:00 PM

	current := t.Hour()*60 + t.Minute()

	if current < minTime {
		return time.Time{}, fmt.Errorf("leave_timing must be after 10:00 AM")
	}

	if current > maxTime {
		return time.Time{}, fmt.Errorf("leave_timing must be before 7:00 PM")
	}

	return t, nil
}
