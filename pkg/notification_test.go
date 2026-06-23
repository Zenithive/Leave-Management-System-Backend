package pkg

import (
	"os"
	"testing"
)

// setTestDomain sets COMPANY_EMAIL_DOMAIN for the duration of a test
// and restores it via t.Cleanup.
func setTestDomain(t *testing.T, domain string) {
	t.Helper()
	prev := os.Getenv("COMPANY_EMAIL_DOMAIN")
	os.Setenv("COMPANY_EMAIL_DOMAIN", domain)
	t.Cleanup(func() { os.Setenv("COMPANY_EMAIL_DOMAIN", prev) })
}

// ─── IsDemoEmail ─────────────────────────────────────────────────────────────

// func TestIsDemoEmail(t *testing.T) {
// 	setTestDomain(t, "example.com")

// 	tests := []struct {
// 		name  string
// 		email string
// 		want  bool
// 	}{
// 		// demo accounts — must be suppressed
// 		{"superadmin demo", "demo.superadmin@example.com", true},
// 		{"admin demo", "demo.admin@example.com", true},
// 		{"hr demo", "demo.hr@example.com", true},
// 		{"manager demo", "demo.manager@example.com", true},
// 		{"employee demo", "demo.employee@example.com", true},
// 		{"intern demo", "demo.intern@example.com", true},
// 		// case-insensitive
// 		{"uppercase prefix", "DEMO.admin@example.com", true},
// 		{"mixed case", "Demo.Manager@Example.Com", true},
// 		// real accounts — must NOT be suppressed
// 		{"real employee", "john.doe@example.com", false},
// 		{"real admin", "admin@example.com", false},
// 		{"empty string", "", false},
// 		// edge: demo prefix but different domain
// 		{"demo wrong domain", "demo.admin@gmail.com", false},
// 		// edge: company domain but no demo prefix
// 		{"no demo prefix", "notdemo.admin@example.com", false},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := IsDemoEmail(tt.email)
// 			if got != tt.want {
// 				t.Errorf("IsDemoEmail(%q) = %v, want %v", tt.email, got, tt.want)
// 			}
// 		})
// 	}
// }

// // ─── FilterDemoRecipients ────────────────────────────────────────────────────

// func TestFilterDemoRecipients(t *testing.T) {
// 	setTestDomain(t, "example.com")

// 	tests := []struct {
// 		name       string
// 		recipients []string
// 		want       []string
// 	}{
// 		{
// 			name:       "all demo — returns empty slice",
// 			recipients: []string{"demo.admin@example.com", "demo.hr@example.com"},
// 			want:       []string{},
// 		},
// 		{
// 			name:       "no demo — returns all",
// 			recipients: []string{"alice@example.com", "bob@example.com"},
// 			want:       []string{"alice@example.com", "bob@example.com"},
// 		},
// 		{
// 			name: "mixed — strips only demo addresses",
// 			recipients: []string{
// 				"demo.manager@example.com",
// 				"real.manager@example.com",
// 				"demo.admin@example.com",
// 				"hr@example.com",
// 			},
// 			want: []string{"real.manager@example.com", "hr@example.com"},
// 		},
// 		{
// 			name:       "empty input — returns empty slice",
// 			recipients: []string{},
// 			want:       []string{},
// 		},
// 		{
// 			name:       "nil input — returns empty slice",
// 			recipients: nil,
// 			want:       []string{},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := FilterDemoRecipients(tt.recipients)
// 			if len(got) != len(tt.want) {
// 				t.Fatalf("FilterDemoRecipients() len = %d, want %d\ngot:  %v\nwant: %v",
// 					len(got), len(tt.want), got, tt.want)
// 			}
// 			for i := range got {
// 				if got[i] != tt.want[i] {
// 					t.Errorf("FilterDemoRecipients()[%d] = %q, want %q", i, got[i], tt.want[i])
// 				}
// 			}
// 		})
// 	}
// }

// // ─── SendEmail skips demo addresses ──────────────────────────────────────────

// func TestSendEmail_SkipsDemoAccount(t *testing.T) {
// 	setTestDomain(t, "example.com")
// 	err := SendEmail("demo.employee@example.com", "Test Subject", "Test Body")
// 	if err != nil {
// 		t.Errorf("SendEmail to demo account should return nil, got: %v", err)
// 	}
// }

// // ─── SendEmailToMultiple skips when all recipients are demo ──────────────────

// func TestSendEmailToMultiple_AllDemoReturnsNil(t *testing.T) {
// 	setTestDomain(t, "example.com")
// 	recipients := []string{
// 		"demo.admin@example.com",
// 		"demo.superadmin@example.com",
// 	}
// 	err := SendEmailToMultiple(recipients, "Subject", "Body")
// 	if err != nil {
// 		t.Errorf("SendEmailToMultiple with all-demo recipients should return nil, got: %v", err)
// 	}
// }
