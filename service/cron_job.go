package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
)

// BirthdayCronService holds dependencies for the birthday cron job.
type BirthdayCronService struct {
	repo *repositories.Repository
	env  *config.ENV
	cron *cron.Cron
}

// NewBirthdayCronService creates and returns a new BirthdayCronService.
func NewBirthdayCronService(repo *repositories.Repository, env *config.ENV) *BirthdayCronService {
	return &BirthdayCronService{
		repo: repo,
		env:  env,
		// Use seconds-level precision so "0 1 0 * * *" = 00:01:00 every day
		cron: cron.New(cron.WithSeconds()),
	}
}

// Start registers the birthday job and starts the cron scheduler.
// Schedule: every day at 00:01:00 (midnight + 1 minute)
func (s *BirthdayCronService) Start() {
	schedule := "0 1 0 * * *" // sec min hour day month weekday

	_, err := s.cron.AddFunc(schedule, func() {
		log.Println("[BirthdayCron] Running birthday notification job...")
		if err := s.runBirthdayJob(); err != nil {
			log.Printf("[BirthdayCron] Job failed: %v\n", err)
		}
	})
	if err != nil {
		log.Fatalf("[BirthdayCron] Failed to register cron job: %v", err)
	}

	s.cron.Start()
	log.Printf("[BirthdayCron] Scheduled at '%s' (daily 00:01)\n", schedule)
}

// Stop gracefully shuts down the cron scheduler.
func (s *BirthdayCronService) Stop() {
	s.cron.Stop()
	log.Println("[BirthdayCron] Scheduler stopped.")
}

// runBirthdayJob fetches today's birthdays and dispatches notifications.
func (s *BirthdayCronService) runBirthdayJob() error {
	// 1. Fetch template from settings
	tmpl, err := s.repo.GetBirthdayMessageTemplate()
	if err != nil {
		return fmt.Errorf("fetch template: %w", err)
	}

	// 2. Fetch today's birthday employees
	employees, err := s.repo.GetTodayBirthdays()
	if err != nil {
		return fmt.Errorf("fetch birthdays: %w", err)
	}

	if len(employees) == 0 {
		log.Println("[BirthdayCron] No birthdays today.")
		return nil
	}

	log.Printf("[BirthdayCron] Found %d birthday(s) today.\n", len(employees))

	// 3. Notify each employee
	for _, emp := range employees {
		message := repositories.RenderBirthdayMessage(tmpl, emp.Name, emp.BirthDate)
		s.notify(emp.Name, emp.Email, message)
	}

	return nil
}

// notify dispatches both email and Slack for a single employee.
// Errors are logged but do not stop other notifications.
func (s *BirthdayCronService) notify(name, email, message string) {
	subject := fmt.Sprintf("🎂 Happy Birthday, %s!", name)

	// --- Email ---
	if err := utils.SendEmail(email, subject, message); err != nil {
		log.Printf("[BirthdayCron] Email failed for %s (%s): %v\n", name, email, err)
	} else {
		log.Printf("[BirthdayCron] Email sent to %s (%s)\n", name, email)
	}

	// --- Slack ---
	if s.env.SLACK_WEBHOOK != "" {
		if err := sendSlackMessage(s.env.SLACK_WEBHOOK, message); err != nil {
			log.Printf("[BirthdayCron] Slack failed for %s: %v\n", name, err)
		} else {
			log.Printf("[BirthdayCron] Slack notified for %s\n", name)
		}
	}
}

// sendSlackMessage posts a plain-text message to a Slack Incoming Webhook.
func sendSlackMessage(webhookURL, text string) error {
	payload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("slack POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}
	return nil
}
