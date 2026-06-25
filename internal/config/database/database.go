package database

import (
	"log"
	"sync"

	"github.com/Zenithive/LeaveManagementSystem/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

var (
	DB   *sqlx.DB
	once sync.Once
)

func RunMigrations(migrationsDir string) {
	if DB == nil {
		log.Fatal("Database not initialized. Call Connection() first.")
	}

	log.Println("Running database migrations...")
	if err := goose.Up(DB.DB, migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations ran successfully!")
}

// Connection establishes a PostgreSQL connection using the ENV configuration.
// It sets the global DB variable and ensures it is initialized only once.
func Connection(env *config.ENV) *sqlx.DB {
	once.Do(func() {
		db, err := sqlx.Connect("postgres", env.DB_URL) // Supabase Postgres URL (with sslmode=require)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Configure connection pool to prevent prepared statement caching issues
		db.SetMaxOpenConns(25)   // Maximum number of open connections
		db.SetMaxIdleConns(5)    // Maximum number of idle connections
		db.SetConnMaxLifetime(0) // Connections never expire (0 = unlimited)
		db.SetConnMaxIdleTime(0) // Idle connections never expire

		// Optional: ping to ensure DB is reachable
		if err := db.Ping(); err != nil {
			log.Fatalf("Database ping failed: %v", err)
		}

		DB = db
		log.Println(" Database connection established successfully")
		RunMigrations("../../migration")
	})
	return DB
}
