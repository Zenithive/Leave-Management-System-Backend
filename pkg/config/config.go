package config

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/joho/godotenv"
)

// ENV holds all application environment variables in a structured way
type ENV struct {
	DB_URL          string
	APP_PORT        string
	SERACT_KEY      string
	FRONTEND_SERVER string
	RESEND_API_KEY  string
	RESEND_FROM     string
	SLACK_WEBHOOK   string
}

var (
	cfg  *ENV      // Singleton instance of ENV
	once sync.Once // Ensures LoadENV is executed only once (thread-safe)
)

// LoadENV loads environment variables from .env file (if exists)
// and system environment variables. It returns a singleton ENV instance.
//
// Usage:
//
//	env := config.LoadENV()
//	dbURL := env.DB.DB_URL
//	port  := env.PORT.PORT
func LoadENV() *ENV {
	once.Do(func() {
		// Try loading .env from the project root regardless of working directory
		// Walk up from this file's location to find .env
		_, filename, _, _ := runtime.Caller(0)
		projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
		envPath := filepath.Join(projectRoot, ".env")

		if err := godotenv.Load(envPath); err != nil {
			// Fallback: try current working directory
			if err2 := godotenv.Load(); err2 != nil {
				log.Println("⚠ No .env file found, using system environment variables")
			}
		}

		// Populate ENV struct with environment variables
		cfg = &ENV{
			DB_URL:          os.Getenv("DB_URL"),
			APP_PORT:        os.Getenv("APP_PORT"),
			SERACT_KEY:      os.Getenv("SECRATE_KEY"),
			FRONTEND_SERVER: os.Getenv("F_SERVER"),
			RESEND_API_KEY:  os.Getenv("RESEND_API_KEY"),
			RESEND_FROM:     os.Getenv("RESEND_FROM"),
			SLACK_WEBHOOK:   os.Getenv("SLACK_WEBHOOK_URL"),
		}
	})
	log.Println(" Environment variables loaded successfully")
	return cfg
}
