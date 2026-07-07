package timezone

import (
	"log"
	"time"
)

// AppLocation is the canonical timezone for all business-logic time operations.
// It is loaded once at startup and shared across the application.
var AppLocation *time.Location

// Initialize loads the application timezone from the provided timezone string.
// Default is "Asia/Kolkata" if tzName is empty.
// Falls back to fixed UTC+5:30 if tzdata is unavailable (minimal Docker images).
func Initialize(tzName string) {
	if tzName == "" {
		tzName = "Asia/Kolkata"
	}

	var err error
	AppLocation, err = time.LoadLocation(tzName)
	if err != nil {
		// Fallback: construct IST manually as UTC+5:30 if tzdata is unavailable.
		// This can happen in minimal Docker images. Add tzdata to your Dockerfile to avoid this.
		log.Printf("[timezone] WARNING: could not load %s (%v), using fixed UTC+5:30 offset", tzName, err)
		AppLocation = time.FixedZone("IST", 5*60*60+30*60)
	}
	log.Printf("[timezone] Application timezone initialized: %s", AppLocation.String())
}

// Now returns the current time in the application's timezone.
func Now() time.Time {
	if AppLocation == nil {
		log.Println("[timezone] WARNING: AppLocation not initialized, using system local time")
		return time.Now()
	}
	return time.Now().In(AppLocation)
}

// TodayDate returns today's date (year, month, day) in the application's timezone.
func TodayDate() (year, month, day int) {
	now := Now()
	return now.Year(), int(now.Month()), now.Day()
}
