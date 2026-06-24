package service

// LeaveAccrualService runs the monthly paid-leave accrual job.
//
// Business rules implemented here:
//
//  1. Only paid, non-early leave types are accrued.
//  2. Monthly credit:
//     - INTERN      → 1.0 day / month
//     - All others  → 1.5 days / month
//  3. Carry-over is automatic: the closing balance already contains unused days
//     from previous months, so adding the monthly credit on top of it gives the
//     employee their accumulated total.  No separate carry-over step is needed.
//  4. Idempotent: the accrual log table (Tbl_Leave_accrual_log) records every
//     (employee, leave_type, month, year) that has been credited.  If the job
//     runs more than once in the same month, the second run is a no-op.
//  5. Each employee+leave_type pair is processed in its own transaction so a
//     single failure does not roll back the entire batch.

import (
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/access_role"
)

const (
	// Monthly accrual rates (in days)
	accrualRateIntern  float64 = 1.0
	accrualRateDefault float64 = 1.5
)

// LeaveAccrualService holds the dependencies for the monthly accrual cron.
type LeaveAccrualService struct {
	repo *repositories.Repository
	cron *cron.Cron
}

// NewLeaveAccrualService creates a new LeaveAccrualService.
func NewLeaveAccrualService(repo *repositories.Repository) *LeaveAccrualService {
	return &LeaveAccrualService{
		repo: repo,
		cron: cron.New(cron.WithSeconds()),
	}
}

// Start registers the monthly accrual job and starts the scheduler.
// Schedule: 1st of every month at 00:05:00.
func (s *LeaveAccrualService) Start() {
	// sec min hour day-of-month month day-of-week
	// "0 5 0 1 * *" = 00:05:00 on the 1st of every month
	schedule := "0 5 0 1 * *"

	_, err := s.cron.AddFunc(schedule, func() {
		now := time.Now()
		log.Printf("[LeaveAccrual] Running monthly accrual for %s %d...\n",
			now.Month().String(), now.Year())

		credited, skipped, failed := s.RunAccrual(now.Month(), now.Year())

		log.Printf("[LeaveAccrual] Done — credited: %d, skipped (already run): %d, failed: %d\n",
			credited, skipped, failed)
	})
	if err != nil {
		log.Fatalf("[LeaveAccrual] Failed to register cron job: %v", err)
	}

	s.cron.Start()
	log.Println("[LeaveAccrual] Scheduled: 1st of every month at 00:05")
}

// Stop gracefully shuts down the accrual scheduler.
func (s *LeaveAccrualService) Stop() {
	s.cron.Stop()
	log.Println("[LeaveAccrual] Scheduler stopped.")
}

// RunAccrual executes the accrual logic for the given month and year.
// It is exported so it can be called manually (e.g. from an admin endpoint
// or the seed script) without waiting for the cron trigger.
//
// Returns the number of (employee, leave_type) pairs that were:
//   - credited  — accrual was applied
//   - skipped   — already credited this month (idempotency guard)
//   - failed    — a DB error occurred (logged, does not abort the batch)
func (s *LeaveAccrualService) RunAccrual(month time.Month, year int) (credited, skipped, failed int) {
	// 1. Fetch all paid, non-early leave types
	leaveTypes, err := s.repo.GetAccruableLeaveTypes()
	if err != nil {
		log.Printf("[LeaveAccrual] Failed to fetch leave types: %v\n", err)
		return
	}
	if len(leaveTypes) == 0 {
		log.Println("[LeaveAccrual] No accrual-eligible leave types found.")
		return
	}

	// 2. Fetch all active employees
	employees, err := s.repo.GetAllActiveEmployeesForAccrual()
	if err != nil {
		log.Printf("[LeaveAccrual] Failed to fetch employees: %v\n", err)
		return
	}
	if len(employees) == 0 {
		log.Println("[LeaveAccrual] No active employees found.")
		return
	}

	monthInt := int(month)

	// 3. Process each employee × leave_type pair
	for _, emp := range employees {
		for _, lt := range leaveTypes {

			// Each pair gets its own transaction — one failure doesn't block others
			tx, err := s.repo.DB.Beginx()
			if err != nil {
				log.Printf("[LeaveAccrual] Failed to begin tx for emp=%s lt=%d: %v\n",
					emp.ID, lt.ID, err)
				failed++
				continue
			}

			// Idempotency check
			alreadyRun, err := s.repo.IsAccrualAlreadyRun(tx, emp.ID, lt.ID, monthInt, year)
			if err != nil {
				log.Printf("[LeaveAccrual] Idempotency check failed emp=%s lt=%d: %v\n",
					emp.ID, lt.ID, err)
				_ = tx.Rollback()
				failed++
				continue
			}
			if alreadyRun {
				_ = tx.Rollback()
				skipped++
				continue
			}

			// Determine accrual rate based on role
			days := accrualRateDefault
			if emp.Role == access_role.ROLE_INTERN {
				days = accrualRateIntern
			}

			// Credit the accrual
			if err := s.repo.CreditMonthlyAccrual(tx, emp.ID, lt.ID, year, monthInt, days); err != nil {
				log.Printf("[LeaveAccrual] Credit failed emp=%s lt=%d: %v\n",
					emp.ID, lt.ID, err)
				_ = tx.Rollback()
				failed++
				continue
			}

			if err := tx.Commit(); err != nil {
				log.Printf("[LeaveAccrual] Commit failed emp=%s lt=%d: %v\n",
					emp.ID, lt.ID, err)
				failed++
				continue
			}

			log.Printf("[LeaveAccrual] Credited %.1f day(s) → emp=%s role=%s lt=%d month=%d/%d\n",
				days, emp.ID, emp.Role, lt.ID, monthInt, year)
			credited++
		}
	}

	return
}

// AccrualSummary is returned by RunAccrualWithSummary for use in API responses.
type AccrualSummary struct {
	Month    int    `json:"month"`
	Year     int    `json:"year"`
	Credited int    `json:"credited"`
	Skipped  int    `json:"skipped"`
	Failed   int    `json:"failed"`
	Message  string `json:"message"`
}

// RunAccrualWithSummary wraps RunAccrual and returns a structured summary.
// Useful for the manual trigger endpoint.
func (s *LeaveAccrualService) RunAccrualWithSummary(month time.Month, year int) AccrualSummary {
	credited, skipped, failed := s.RunAccrual(month, year)
	return AccrualSummary{
		Month:    int(month),
		Year:     year,
		Credited: credited,
		Skipped:  skipped,
		Failed:   failed,
		Message:  fmt.Sprintf("Accrual complete for %s %d", month.String(), year),
	}
}
