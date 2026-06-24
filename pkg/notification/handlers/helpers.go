package handlers

import (
	"github.com/Zenithive/LeaveManagementSystem/internal/config"
	"github.com/Zenithive/LeaveManagementSystem/internal/models"
)

// appName returns the application name from config.
// Falls back to a default if not set.
func appName(cfg *config.ENV) string {
	if cfg.APP_NAME != "" {
		return cfg.APP_NAME
	}
	return "Leave Management System"
}

// loginURL returns the frontend URL appended to credential emails.
func loginURL(cfg *config.ENV) string {
	if cfg.APP_URL != "" {
		return "\nLogin: " + cfg.APP_URL
	}
	return ""
}

func recipientEmails(recipients []models.Recipient) []string {
	emails := make([]string, 0, len(recipients))

	for _, r := range recipients {
		if r.Email != "" {
			emails = append(emails, r.Email)
		}
	}

	return emails
}
