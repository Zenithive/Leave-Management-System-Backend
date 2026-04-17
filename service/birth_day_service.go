package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

// RenderBirthdayMessage replaces supported placeholders in the template:
//
//	{name}  → employee full name
//	{date}  → birth date formatted as "2 January"
//	{age}   → calculated age in years (requires birth_date)
func RenderBirthdayMessage(template, name string, birthDate *time.Time) string {
	msg := template
	msg = strings.ReplaceAll(msg, "{name}", name)

	if birthDate != nil {
		msg = strings.ReplaceAll(msg, "{date}", birthDate.Format("2 January"))
		age := calculateAge(*birthDate)
		msg = strings.ReplaceAll(msg, "{age}", fmt.Sprintf("%d", age))
	} else {
		// Remove placeholders gracefully if no birth date
		msg = strings.ReplaceAll(msg, "{date}", "")
		msg = strings.ReplaceAll(msg, "{age}", "")
	}

	return msg
}

func calculateAge(birthDate time.Time) int {
	now := time.Now()
	age := now.Year() - birthDate.Year()
	// Adjust if birthday hasn't occurred yet this year
	// if now.YearDay() < birthDate.YearDay() {
	// 	age--
	// }
	return age
}

// Calculation classifies each employee's birthday as PAST, TODAY, or UPCOMING
// relative to today's actual date, within the requested calendar scope.
//
// month > 0 && year > 0  → specific month/year view (e.g. April 2027)
// year > 0 only          → full year view (e.g. all of 2027)
// default (0,0)          → upcoming 30 days from today
//
// Age is calculated as of the birthday in the target year (not current year).
func Calculation(list []models.BirthdayEmployee, month, year int) []models.BirthdayEmployee {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Determine the reference year for birthday placement
	refYear := now.Year()
	if year > 0 {
		refYear = year
	}

	for i := range list {
		emp := &list[i]
		if emp.BirthDate == nil {
			continue
		}

		// Place the birthday in the reference year
		birthdayInRefYear := time.Date(
			refYear,
			emp.BirthDate.Month(),
			emp.BirthDate.Day(),
			0, 0, 0, 0,
			now.Location(),
		)

		// Age turning on this birthday (birth_year → refYear)
		emp.Age = refYear - emp.BirthDate.Year()

		var targetTime time.Time

		if year > 0 {
			// Calendar mode (year only, or month+year): classify relative to today
			switch {
			case birthdayInRefYear.Before(today):
				emp.Status = string(constant.StatusPast)
				targetTime = birthdayInRefYear
			case birthdayInRefYear.Equal(today):
				emp.Status = string(constant.StatusToday)
				targetTime = today.AddDate(0, 0, 1)
			default:
				emp.Status = string(constant.StatusUpcoming)
				targetTime = birthdayInRefYear
			}
		} else {
			// Default: upcoming 30-day window — wrap to next year if already passed this year
			targetTime = birthdayInRefYear // refYear == now.Year() here
			if targetTime.Before(today) {
				targetTime = targetTime.AddDate(1, 0, 0)
				emp.Age++ // birthday is next year, so age is one more
			}
			if targetTime.Equal(today) {
				emp.Status = string(constant.StatusToday)
				targetTime = today.AddDate(0, 0, 1)
			} else {
				emp.Status = string(constant.StatusUpcoming)
			}
		}

		// Time diff: past = how long ago, future = how long until
		var diff time.Duration
		if now.After(targetTime) {
			diff = now.Sub(targetTime)
		} else {
			diff = targetTime.Sub(now)
		}

		totalHours := int(diff.Hours())
		emp.RemainingDays = totalHours / 24
		emp.RemainingHours = totalHours % 24
		emp.RemainingMinutes = int(diff.Minutes()) % 60
	}

	return list
}
