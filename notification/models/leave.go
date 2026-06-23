package models

import (
	"time"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

// LeaveNotificationData carries all fields needed for any leave-related notification.
// A single struct covers apply, approve, reject, withdraw, cancel —
// handlers only read the fields relevant to their event type.
type LeaveNotificationData struct {
	// Leave details
	LeaveID   string
	LeaveType string
	StartDate time.Time
	EndDate   time.Time
	Days      float64
	Reason    string

	// Employee (applicant)
	EmployeeID    string
	EmployeeName  string
	EmployeeEmail string

	// Actor (who triggered the action — approver, rejecter, withdrawer)
	ActorName  string
	ActorEmail string
	ActorRole  string

	// Unified recipients (replaces AdminEmails + HREmails)
	Recipients []models.Recipient
}
