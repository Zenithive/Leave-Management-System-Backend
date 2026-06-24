package service

import (
	"fmt"
	"time"
)

// HolidayChecker is the minimal interface the cron guard needs from the repository.
// This keeps the utility decoupled from the concrete Repository type.
type HolidayChecker interface {
	IsHolidayDate(date time.Time) (bool, error)
}

// CronSkipReason is returned when the cron should not run today.
type CronSkipReason struct {
	Reason string // e.g. "Saturday", "Sunday", "Holiday"
	Date   string // formatted as "2006-01-02"
}

func (s *CronSkipReason) Error() string {
	return fmt.Sprintf("cron skipped on %s (%s)", s.Date, s.Reason)
}

// ShouldSkipCronToday returns a non-nil *CronSkipReason when the cron job
// should NOT run today — i.e. when today is a weekend or a configured holiday.
//
// Pass a HolidayChecker (your *Repository satisfies this) to enable the
// holiday check.  Pass nil to skip the DB lookup (weekends are always checked).
//
//	skip, err := pkg.ShouldSkipCronToday(time.Now(), h.Query)
//	if err != nil { /* handle DB error */ }
//	if skip != nil { /* return early */ }
func ShouldSkipCronToday(now time.Time, checker HolidayChecker) (*CronSkipReason, error) {
	today := now.Format("2006-01-02")

	// 1. Weekend check — never needs a DB round-trip
	wd := now.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return &CronSkipReason{Reason: wd.String(), Date: today}, nil
	}

	// 2. Holiday check — only when a checker is provided
	if checker != nil {
		isHoliday, err := checker.IsHolidayDate(now)
		if err != nil {
			return nil, fmt.Errorf("holiday check failed: %w", err)
		}
		if isHoliday {
			return &CronSkipReason{Reason: "Holiday", Date: today}, nil
		}
	}

	return nil, nil
}

// IsWeekend reports whether the given time falls on a Saturday or Sunday.
func IsWeekend(t time.Time) bool {
	wd := t.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}
