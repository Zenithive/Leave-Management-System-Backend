package config

import (
	"log"
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

// ENV holds all application environment variables
type ENV struct {
	DB_URL               string
	APP_PORT             string
	SERACT_KEY           string
	FRONTEND_SERVER      string
	ALLOWED_ORIGINS      []string
	RESEND_API_KEY       string
	RESEND_FROM          string
	SLACK_WEBHOOK        string
	EXTERNAL_API_URL     string
	CRON_SECRET          string
	COMPANY_EMAIL_DOMAIN string // e.g. "yourdomain.com" — used for email validation and demo seeder
}

var (
	cfg  *ENV
	once sync.Once
)

func LoadENV() *ENV {
	once.Do(func() {

		if err := godotenv.Overload(".env"); err != nil {
			log.Println("⚠ No .env file found, using system environment variables")
		}

		// Read origins
		rawOrigins := os.Getenv("ALLOWED_ORIGINS")

		var origins []string

		for _, origin := range strings.Split(rawOrigins, ",") {
			origin = strings.TrimSpace(origin)

			if origin != "" {
				origins = append(origins, origin)
			}
		}

		cfg = &ENV{
			DB_URL:               os.Getenv("DB_URL"),
			APP_PORT:             os.Getenv("APP_PORT"),
			SERACT_KEY:           os.Getenv("SECRATE_KEY"),
			FRONTEND_SERVER:      os.Getenv("F_SERVER"),
			ALLOWED_ORIGINS:      origins,
			RESEND_API_KEY:       os.Getenv("RESEND_API_KEY"),
			RESEND_FROM:          os.Getenv("RESEND_FROM"),
			SLACK_WEBHOOK:        os.Getenv("SLACK_WEBHOOK_URL"),
			EXTERNAL_API_URL:     os.Getenv("EXTERNAL_API_URL"),
			CRON_SECRET:          os.Getenv("CRON_SECRET"),
			COMPANY_EMAIL_DOMAIN: os.Getenv("COMPANY_EMAIL_DOMAIN"),
		}
	})

	log.Println(" Environment variables loaded successfully")

	return cfg
}
