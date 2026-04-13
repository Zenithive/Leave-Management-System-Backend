package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
)

func CalculateWorkingDays(Query *repositories.Repository, tx *sqlx.Tx, start, end time.Time, leaveTiming time.Time) (float64, error) {
	// 1️ Validate date range
	if end.Before(start) {
		return 0, fmt.Errorf("end date cannot be before start date")
	}

	// Normalize dates to midnight UTC to avoid timezone issues
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)

	fmt.Println("1s")

	// 2️ Fetch holidays within range
	holidays, err := Query.GetByFilterHolidayBetwweenTwoDates(tx, start, end)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch holidays: %v", err)
	}
	fmt.Println("2s")

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

// CalculateWorkingDaysWithTiming calculates working days based on timing type
// timingID: 1 = First Half (0.5 days), 2 = Second Half (0.5 days), 3 = Full Day (1.0 days)
func CalculateWorkingDaysWithTiming(Query *repositories.Repository, tx *sqlx.Tx, start, end time.Time, timingID int, leaveTiming time.Time) (float64, error) {
	// First calculate the base working days
	baseDays, err := CalculateWorkingDays(Query, tx, start, end, leaveTiming)
	fmt.Println("Base Days:", baseDays)
	if err != nil {
		return 0, err
	}

	if !leaveTiming.IsZero() {
		return baseDays, nil
	}

	switch timingID {
	case 1, 2: // First Half or Second Half
		return baseDays * 0.5, nil
	case 3: // Full Day
		return baseDays, nil
	default:
		return 0, fmt.Errorf("invalid timing ID: %d. Must be 1 (First Half), 2 (Second Half), or 3 (Full Day)", timingID)
	}
}

type LeaveSummary struct {
	PaidDays   float64
	UnpaidDays float64
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
    `, employeeID, lastDay, firstDay)

	if err != nil {
		fmt.Printf("Error fetching leaves: %v\n", err)
		return LeaveSummary{}
	}

	summary := LeaveSummary{}

	// 4. Calculate days
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

// CalculateLeaveBalances calculates leave balances using map-based approach
// This function takes leave types and balance records, then calculates the final balances
func CalculateLeaveBalances(leaveTypes []LeaveTypeData, balanceRecords []LeaveBalanceData) []CalculatedBalance {
	// Create a map of leave_type_id -> balance for O(1) lookup
	balanceMap := make(map[int]LeaveBalanceData)
	for _, balance := range balanceRecords {
		balanceMap[balance.LeaveTypeID] = balance
	}

	var calculatedBalances []CalculatedBalance

	// Calculate balances for each leave type
	for _, lt := range leaveTypes {
		balance, exists := balanceMap[lt.LeaveTypeID]

		var opening, accrued, used, adjusted, total, available float64

		if exists {
			// Balance record exists - use actual values from database
			opening = balance.Opening
			accrued = balance.Accrued
			used = balance.Used
			adjusted = balance.Adjusted
			// Total = Opening + Accrued
			total = opening + accrued
			// Available = Closing (which is calculated as: opening + accrued - used + adjusted)
			available = balance.Closing
		} else {
			// No balance record exists - use default entitlement
			opening = lt.DefaultEntitlement
			accrued = 0
			used = 0
			adjusted = 0
			// Total = Default Entitlement (treated as opening)
			total = lt.DefaultEntitlement
			// Available = Default Entitlement (since nothing used yet)
			available = lt.DefaultEntitlement
		}

		calculatedBalances = append(calculatedBalances, CalculatedBalance{
			LeaveTypeID: lt.LeaveTypeID,
			LeaveType:   lt.LeaveTypeName,
			Opening:     opening,
			Accrued:     accrued,
			Used:        used,
			Adjusted:    adjusted,
			Total:       total,
			Available:   available,
		})
	}

	return calculatedBalances
}

// validateLeaveTiming
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
