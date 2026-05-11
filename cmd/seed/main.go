// cmd/seed/main.go
//
// Demo account seeder for local / staging environments.
//
// Usage:
//
//	go run ./cmd/seed            → create demo accounts for all 6 roles
//	go run ./cmd/seed --teardown → remove all demo accounts and their data
//
// All demo accounts use the email pattern:  demo.<role>@zenithive.com
// Fixed password for every account:         Demo@1234
//
// Org structure created:
//
//	SUPERADMIN  demo.superadmin@zenithive.com
//	ADMIN       demo.admin@zenithive.com
//	HR          demo.hr@zenithive.com
//	MANAGER     demo.manager@zenithive.com
//	  └─ EMPLOYEE  demo.employee@zenithive.com   (reports to demo manager)
//	  └─ INTERN    demo.intern@zenithive.com     (reports to demo manager)

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// ─── constants ───────────────────────────────────────────────────────────────

const (
	demoPassword  = "Demo@1234"
	demoEmailMark = "demo." // prefix that identifies all demo accounts
)

// demoAccounts defines the accounts to create, in insertion order.
// managerEmail is resolved at runtime after the MANAGER row is inserted.
var demoAccounts = []struct {
	fullName     string
	email        string
	role         string
	salary       float64
	managerEmail string // filled in at runtime for EMPLOYEE and INTERN
}{
	{"Demo SuperAdmin", "demo.superadmin@zenithive.com", "SUPERADMIN", 0, ""},
	{"Demo Admin", "demo.admin@zenithive.com", "ADMIN", 0, ""},
	{"Demo HR", "demo.hr@zenithive.com", "HR", 0, ""},
	{"Demo Manager", "demo.manager@zenithive.com", "MANAGER", 50000, ""},
	{"Demo Employee", "demo.employee@zenithive.com", "EMPLOYEE", 30000, "demo.manager@zenithive.com"},
	{"Demo Intern", "demo.intern@zenithive.com", "INTERN", 15000, "demo.manager@zenithive.com"},
}

// ─── entry point ─────────────────────────────────────────────────────────────

func main() {
	teardown := flag.Bool("teardown", false, "Remove all demo accounts and their associated data")
	flag.Parse()

	db := connectDB()
	defer db.Close()

	if *teardown {
		runTeardown(db)
	} else {
		runSeed(db)
	}
}

// ─── database connection ──────────────────────────────────────────────────────

func connectDB() *sqlx.DB {
	// Load .env from project root (two levels up from cmd/seed)
	_ = godotenv.Overload(".env")

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("✓ Connected to database")
	return db
}

// ─── seed ─────────────────────────────────────────────────────────────────────

func runSeed(db *sqlx.DB) {
	log.Println("Starting demo account seeding...")

	hash, err := bcrypt.GenerateFromPassword([]byte(demoPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	hashedPassword := string(hash)

	// Fetch all leave types once — needed for leave balance allocation
	leaveTypes, err := fetchLeaveTypes(db)
	if err != nil {
		log.Fatalf("Failed to fetch leave types: %v", err)
	}
	if len(leaveTypes) == 0 {
		log.Println("⚠  No leave types found in the database. Leave balances will not be allocated.")
		log.Println("   Run the application at least once so migrations create the leave types, then re-run the seeder.")
	}

	// Track manager UUID so EMPLOYEE and INTERN can reference it
	managerIDs := map[string]string{} // email → uuid string

	created := 0
	skipped := 0

	for _, acc := range demoAccounts {
		// Idempotency check — skip if already exists
		if emailExists(db, acc.email) {
			log.Printf("  SKIP  %s (already exists)", acc.email)
			skipped++
			continue
		}

		// Resolve role ID
		roleID, err := getRoleID(db, acc.role)
		if err != nil {
			log.Fatalf("Role %q not found in Tbl_Role. Make sure migrations have run: %v", acc.role, err)
		}

		// Resolve manager UUID (only for EMPLOYEE and INTERN)
		var managerID *string
		if acc.managerEmail != "" {
			if mid, ok := managerIDs[acc.managerEmail]; ok {
				managerID = &mid
			} else {
				// Manager might already exist in DB from a previous partial seed
				mid, err := getEmployeeIDByEmail(db, acc.managerEmail)
				if err != nil {
					log.Fatalf("Manager %q not found. Seed the MANAGER account first: %v", acc.managerEmail, err)
				}
				managerID = &mid
			}
		}

		joiningDate := time.Now()

		// Insert employee inside a transaction
		tx, err := db.Beginx()
		if err != nil {
			log.Fatalf("Failed to begin transaction: %v", err)
		}

		empID, err := insertEmployee(tx, acc.fullName, acc.email, roleID, hashedPassword, acc.salary, joiningDate, managerID)
		if err != nil {
			_ = tx.Rollback()
			log.Fatalf("Failed to insert %s: %v", acc.email, err)
		}

		// Allocate leave balances for every non-early leave type
		for _, lt := range leaveTypes {
			entitlement := lt.DefaultEntitlement
			if acc.role == "INTERN" && lt.InternEntitlement != nil {
				entitlement = *lt.InternEntitlement
			}
			if err := allocateLeaveBalance(tx, empID, lt.ID, entitlement); err != nil {
				_ = tx.Rollback()
				log.Fatalf("Failed to allocate leave balance for %s (leave_type %d): %v", acc.email, lt.ID, err)
			}
		}

		if err := tx.Commit(); err != nil {
			log.Fatalf("Failed to commit transaction for %s: %v", acc.email, err)
		}

		managerIDs[acc.email] = empID
		log.Printf("  CREATE %s  [%s]  id=%s", acc.email, acc.role, empID)
		created++
	}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    DEMO ACCOUNTS READY                          ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  %-12s  %-38s  %s\n", "ROLE", "EMAIL", "PASSWORD ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
	for _, acc := range demoAccounts {
		fmt.Printf("║  %-12s  %-38s  %s ║\n", acc.role, acc.email, demoPassword)
	}
	fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Created: %-3d   Skipped (already existed): %-3d                  ║\n", created, skipped)
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Org structure:")
	fmt.Println("  SUPERADMIN  demo.superadmin@zenithive.com")
	fmt.Println("  ADMIN       demo.admin@zenithive.com")
	fmt.Println("  HR          demo.hr@zenithive.com")
	fmt.Println("  MANAGER     demo.manager@zenithive.com")
	fmt.Println("    └─ EMPLOYEE  demo.employee@zenithive.com")
	fmt.Println("    └─ INTERN    demo.intern@zenithive.com")
	fmt.Println()
	fmt.Println("To remove all demo accounts run:")
	fmt.Println("  go run ./cmd/seed --teardown")
}

// ─── teardown ─────────────────────────────────────────────────────────────────

func runTeardown(db *sqlx.DB) {
	log.Println("Starting demo account teardown...")

	// Collect all demo employee IDs
	rows, err := db.Query(`SELECT id, email FROM Tbl_Employee WHERE email LIKE $1`, demoEmailMark+"%@zenithive.com")
	if err != nil {
		log.Fatalf("Failed to query demo accounts: %v", err)
	}

	type empRow struct {
		ID    string
		Email string
	}
	var employees []empRow
	for rows.Next() {
		var e empRow
		if err := rows.Scan(&e.ID, &e.Email); err != nil {
			rows.Close()
			log.Fatalf("Failed to scan row: %v", err)
		}
		employees = append(employees, e)
	}
	rows.Close()

	if len(employees) == 0 {
		log.Println("No demo accounts found. Nothing to remove.")
		return
	}

	ids := make([]string, len(employees))
	for i, e := range employees {
		ids[i] = "'" + e.ID + "'"
		log.Printf("  Found  %s  id=%s", e.Email, e.ID)
	}
	idList := strings.Join(ids, ",")

	tx, err := db.Beginx()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	steps := []struct {
		label string
		query string
	}{
		// Remove manager references first to avoid FK violations
		{"Clear manager_id references", fmt.Sprintf(
			`UPDATE Tbl_Employee SET manager_id = NULL WHERE manager_id IN (%s)`, idList)},
		// Dependent tables — order matters (children before parents)
		{"Delete leave adjustments", fmt.Sprintf(
			`DELETE FROM Tbl_Leave_adjustment WHERE employee_id IN (%s) OR created_by IN (%s)`, idList, idList)},
		{"Delete leave balances", fmt.Sprintf(
			`DELETE FROM Tbl_Leave_balance WHERE employee_id IN (%s)`, idList)},
		{"Delete leaves (applied_by / approved_by)", fmt.Sprintf(
			`UPDATE Tbl_Leave SET applied_by = NULL WHERE applied_by IN (%s)`, idList)},
		{"Nullify approved_by on leaves", fmt.Sprintf(
			`UPDATE Tbl_Leave SET approved_by = NULL WHERE approved_by IN (%s)`, idList)},
		{"Delete leaves", fmt.Sprintf(
			`DELETE FROM Tbl_Leave WHERE employee_id IN (%s)`, idList)},
		{"Delete payslips", fmt.Sprintf(
			`DELETE FROM Tbl_Payslip WHERE employee_id IN (%s)`, idList)},
		{"Delete audit records", fmt.Sprintf(
			`DELETE FROM Tbl_Audit WHERE actor_id IN (%s)`, idList)},
		{"Delete equipment assignments", fmt.Sprintf(
			`DELETE FROM tbl_equipment_assignment WHERE employee_id IN (%s)`, idList)},
		{"Delete employees", fmt.Sprintf(
			`DELETE FROM Tbl_Employee WHERE id IN (%s)`, idList)},
	}

	for _, step := range steps {
		if _, err := tx.Exec(step.query); err != nil {
			_ = tx.Rollback()
			log.Fatalf("Step %q failed: %v", step.label, err)
		}
		log.Printf("  ✓  %s", step.label)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit teardown transaction: %v", err)
	}

	fmt.Println()
	fmt.Printf("✓ Removed %d demo account(s) and all associated data.\n", len(employees))
}

// ─── helpers ──────────────────────────────────────────────────────────────────

type leaveTypeRow struct {
	ID                 int  `db:"id"`
	DefaultEntitlement int  `db:"default_entitlement"`
	InternEntitlement  *int `db:"intern_entitlement"`
}

// fetchLeaveTypes returns all non-early leave types.
func fetchLeaveTypes(db *sqlx.DB) ([]leaveTypeRow, error) {
	var rows []leaveTypeRow
	err := db.Select(&rows, `
		SELECT id, default_entitlement, intern_entitlement
		FROM Tbl_Leave_type
		WHERE is_early IS NULL OR is_early = FALSE
		ORDER BY id
	`)
	return rows, err
}

// emailExists returns true if an employee with that email already exists.
func emailExists(db *sqlx.DB, email string) bool {
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM Tbl_Employee WHERE email = $1`, email).Scan(&count)
	return count > 0
}

// getRoleID returns the integer ID for a role type string.
func getRoleID(db *sqlx.DB, roleType string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM Tbl_Role WHERE type = $1`, roleType).Scan(&id)
	return id, err
}

// getEmployeeIDByEmail returns the UUID string for an employee by email.
func getEmployeeIDByEmail(db *sqlx.DB, email string) (string, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM Tbl_Employee WHERE email = $1`, email).Scan(&id)
	return id, err
}

// insertEmployee inserts a new employee row and returns the generated UUID.
func insertEmployee(
	tx *sqlx.Tx,
	fullName, email, roleID, password string,
	salary float64,
	joiningDate time.Time,
	managerID *string,
) (string, error) {
	var empID string
	err := tx.QueryRow(`
		INSERT INTO Tbl_Employee
			(full_name, email, role_id, password, salary, joining_date, manager_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
		RETURNING id
	`, fullName, email, roleID, password, salary, joiningDate, managerID).Scan(&empID)
	return empID, err
}

// allocateLeaveBalance creates a leave balance row for the given employee + leave type.
func allocateLeaveBalance(tx *sqlx.Tx, employeeID string, leaveTypeID, entitlement int) error {
	_, err := tx.Exec(`
		INSERT INTO Tbl_Leave_balance
			(employee_id, leave_type_id, year, opening, accrued, used, adjusted, closing)
		VALUES ($1, $2, EXTRACT(YEAR FROM CURRENT_DATE), $3, 0, 0, 0, $3)
		ON CONFLICT DO NOTHING
	`, employeeID, leaveTypeID, entitlement)
	return err
}
