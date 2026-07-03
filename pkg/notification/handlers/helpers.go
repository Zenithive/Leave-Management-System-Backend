package handlers

import (
	"github.com/Zenithive/LeaveManagementSystem/internal/config"
)

// appName returns the application name from config.
// Falls back to a default if not set.
func appName(cfg *config.ENV) string {
	if cfg.APP_NAME != "" {
		return cfg.APP_NAME
	}
	return "Leave Management System"
}
