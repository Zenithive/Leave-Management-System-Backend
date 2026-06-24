package service

import (
	"testing"
)

// ─── ValidateLeaveTiming ─────────────────────────────────────────────────────

func TestValidateLeaveTiming(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantHour int
		wantMin  int
	}{
		// valid times
		{"valid 10:00", "10:00", false, 10, 0},
		{"valid noon", "12:30", false, 12, 30},
		{"valid 18:59", "18:59", false, 18, 59},
		{"valid boundary 19:00", "19:00", false, 19, 0},
		// invalid — too early
		{"before 10:00", "09:59", true, 0, 0},
		{"midnight", "00:00", true, 0, 0},
		// invalid — too late
		{"after 19:00", "19:01", true, 0, 0},
		{"23:00", "23:00", true, 0, 0},
		// invalid format
		{"bad format", "9am", true, 0, 0},
		{"empty string", "", true, 0, 0},
		{"seconds included", "10:00:00", true, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateLeaveTiming(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateLeaveTiming(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateLeaveTiming(%q) unexpected error: %v", tt.input, err)
				return
			}

			if got.Hour() != tt.wantHour || got.Minute() != tt.wantMin {
				t.Errorf("ValidateLeaveTiming(%q) = %02d:%02d, want %02d:%02d",
					tt.input, got.Hour(), got.Minute(), tt.wantHour, tt.wantMin)
			}
		})
	}
}

// ─── CalculateLeaveBalances ───────────────────────────────────────────────────

func TestCalculateLeaveBalances(t *testing.T) {
	leaveTypes := []LeaveTypeData{
		{LeaveTypeID: 1, LeaveTypeName: "Annual Leave"},
		{LeaveTypeID: 2, LeaveTypeName: "Sick Leave"},
		{LeaveTypeID: 3, LeaveTypeName: "Casual Leave"},
	}

	t.Run("basic calculation", func(t *testing.T) {
		balances := []LeaveBalanceData{
			{LeaveTypeID: 1, Opening: 18, Accrued: 0, Used: 3, Adjusted: 0, Closing: 15},
			{LeaveTypeID: 2, Opening: 6, Accrued: 0, Used: 1, Adjusted: 0, Closing: 5},
		}

		result := CalculateLeaveBalances(leaveTypes, balances)

		if len(result) != 2 {
			t.Fatalf("expected 2 results, got %d", len(result))
		}

		// Annual Leave
		if result[0].LeaveType != "Annual Leave" {
			t.Errorf("result[0].LeaveType = %q, want %q", result[0].LeaveType, "Annual Leave")
		}
		if result[0].Total != 18 { // Opening + Accrued
			t.Errorf("result[0].Total = %.1f, want 18", result[0].Total)
		}
		if result[0].Available != 15 {
			t.Errorf("result[0].Available = %.1f, want 15", result[0].Available)
		}

		// Sick Leave
		if result[1].LeaveType != "Sick Leave" {
			t.Errorf("result[1].LeaveType = %q, want %q", result[1].LeaveType, "Sick Leave")
		}
		if result[1].Available != 5 {
			t.Errorf("result[1].Available = %.1f, want 5", result[1].Available)
		}
	})

	t.Run("empty balance records returns nil slice", func(t *testing.T) {
		result := CalculateLeaveBalances(leaveTypes, []LeaveBalanceData{})
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})

	t.Run("leave type not in name map gets empty name", func(t *testing.T) {
		balances := []LeaveBalanceData{
			{LeaveTypeID: 99, Opening: 10, Accrued: 0, Used: 0, Adjusted: 0, Closing: 10},
		}
		result := CalculateLeaveBalances(leaveTypes, balances)
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		if result[0].LeaveType != "" {
			t.Errorf("unknown leave type should have empty name, got %q", result[0].LeaveType)
		}
	})

	t.Run("total = opening + accrued", func(t *testing.T) {
		balances := []LeaveBalanceData{
			{LeaveTypeID: 1, Opening: 12, Accrued: 3, Used: 2, Adjusted: 1, Closing: 14},
		}
		result := CalculateLeaveBalances(leaveTypes, balances)
		wantTotal := 12.0 + 3.0 // Opening + Accrued
		if result[0].Total != wantTotal {
			t.Errorf("Total = %.1f, want %.1f", result[0].Total, wantTotal)
		}
	})

	t.Run("adjusted field is preserved", func(t *testing.T) {
		balances := []LeaveBalanceData{
			{LeaveTypeID: 2, Opening: 6, Accrued: 0, Used: 0, Adjusted: 2, Closing: 8},
		}
		result := CalculateLeaveBalances(leaveTypes, balances)
		if result[0].Adjusted != 2 {
			t.Errorf("Adjusted = %.1f, want 2", result[0].Adjusted)
		}
	})
}

// ─── CalculateWorkingDaysWithTiming (pure timing multiplier logic) ────────────
// We test the timing multiplier in isolation by verifying the 0.5 / 1.0 factor
// applied to a known base-day count. The DB-dependent holiday fetch is bypassed
// by using a Monday→Monday range with no holidays (requires no DB call for the
// multiplier path itself — we test the switch logic directly here).

func TestTimingMultiplier(t *testing.T) {
	tests := []struct {
		name     string
		baseDays float64
		timingID int
		wantDays float64
		wantErr  bool
	}{
		{"full day", 5, 3, 5.0, false},
		{"first half", 4, 1, 2.0, false},
		{"second half", 4, 2, 2.0, false},
		{"invalid timing id", 5, 0, 0, true},
		{"invalid timing id 4", 5, 4, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyTimingMultiplier(tt.baseDays, tt.timingID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("applyTimingMultiplier(%v, %d) expected error, got nil", tt.baseDays, tt.timingID)
				}
				return
			}
			if err != nil {
				t.Errorf("applyTimingMultiplier(%v, %d) unexpected error: %v", tt.baseDays, tt.timingID, err)
				return
			}
			if got != tt.wantDays {
				t.Errorf("applyTimingMultiplier(%v, %d) = %.1f, want %.1f", tt.baseDays, tt.timingID, got, tt.wantDays)
			}
		})
	}
}

// ─── ValidateLeaveTiming boundary precision ──────────────────────────────────

func TestValidateLeaveTiming_ReturnedTimeIsZeroDate(t *testing.T) {
	// time.Parse("15:04", ...) always returns year 0000-01-01.
	// Callers rely on Hour()/Minute() only — verify that contract.
	got, err := ValidateLeaveTiming("14:30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// time.Parse with no date component returns year 0 (not Go's zero Time year of 1).
	if got.Year() != 0 {
		t.Errorf("expected parsed year 0, got %d", got.Year())
	}
	if got.Hour() != 14 || got.Minute() != 30 {
		t.Errorf("expected 14:30, got %02d:%02d", got.Hour(), got.Minute())
	}
}
