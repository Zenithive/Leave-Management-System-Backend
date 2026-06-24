package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// EmailProvider is the abstraction layer for any email delivery mechanism.
// Swap ResendEmailProvider for SES, SendGrid, or SMTP without touching handlers.
type EmailProvider interface {
	// Send delivers a single email.
	Send(to, subject, body string) error
	// SendBulk delivers the same email to multiple recipients in one API call.
	SendBulk(recipients []string, subject, body string) error
}

// ─────────────────────────────────────────────────────────────────────────────
// ResendEmailProvider — sends via the Resend HTTP API
// ─────────────────────────────────────────────────────────────────────────────

// ResendEmailProvider implements EmailProvider using the Resend REST API.
type ResendEmailProvider struct {
	apiKey            string
	from              string
	companyDomain     string // used for demo-account filtering
	httpClient        *http.Client
	logger            *slog.Logger
}

// NewResendEmailProvider constructs a provider from environment variables.
// Panics on startup if the required config is missing — fail fast is correct here.
func NewResendEmailProvider(logger *slog.Logger) *ResendEmailProvider {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("RESEND_FROM")
	if apiKey == "" || from == "" {
		panic("notification: RESEND_API_KEY and RESEND_FROM must be set")
	}
	return &ResendEmailProvider{
		apiKey:        apiKey,
		from:          from,
		companyDomain: strings.ToLower(strings.TrimSpace(os.Getenv("COMPANY_EMAIL_DOMAIN"))),
		httpClient:    &http.Client{Timeout: 15 * time.Second},
		logger:        logger,
	}
}

func (p *ResendEmailProvider) Send(to, subject, body string) error {
	if p.isDemoEmail(to) {
		p.logger.Info("skipping email to demo account", "to", to)
		return nil
	}
	return p.send([]string{to}, subject, body)
}

func (p *ResendEmailProvider) SendBulk(recipients []string, subject, body string) error {
	filtered := p.filterDemoEmails(recipients)
	if len(filtered) == 0 {
		p.logger.Info("skipping bulk email — all recipients are demo accounts", "subject", subject)
		return nil
	}
	return p.send(filtered, subject, body)
}

// send is the shared HTTP call used by both Send and SendBulk.
func (p *ResendEmailProvider) send(to []string, subject, body string) error {
	payload := struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
	}{
		From:    p.from,
		To:      to,
		Subject: subject,
		HTML:    body,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("email: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("email: resend api: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("email: resend api status %d: %s", resp.StatusCode, string(respBody))
	}

	// Log the Resend email ID so you can trace delivery in the Resend dashboard.
	// A 200/201 from Resend means the API accepted the request, NOT that delivery succeeded.
	// On the free plan only your verified address receives mail — all others are silently dropped.
	p.logger.Info("email accepted by resend", "to", to, "subject", subject, "resend_response", string(respBody))
	return nil
}

func (p *ResendEmailProvider) isDemoEmail(email string) bool {
	if p.companyDomain == "" {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(email))
	return strings.HasPrefix(lower, "demo.") && strings.HasSuffix(lower, "@"+p.companyDomain)
}

func (p *ResendEmailProvider) filterDemoEmails(emails []string) []string {
	out := make([]string, 0, len(emails))
	for _, e := range emails {
		if !p.isDemoEmail(e) {
			out = append(out, e)
		}
	}
	return out
}
