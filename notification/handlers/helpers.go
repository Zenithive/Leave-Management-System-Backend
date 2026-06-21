package handlers

import "github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"

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
