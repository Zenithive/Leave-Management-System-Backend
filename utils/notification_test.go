package utils

import (
	"testing"
)

// ─── IsDemoEmail ─────────────────────────────────────────────────────────────

func TestIsDemoEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		// demo accounts — must be suppressed
		{"superadmin demo", "demo.superadmin@zenithive.com", true},
		{"admin demo", "demo.admin@zenithive.com", true},
		{"hr demo", "demo.hr@zenithive.com", true},
		{"manager demo", "demo.manager@zenithive.com", true},
		{"employee demo", "demo.employee@zenithive.com", true},
		{"intern demo", "demo.intern@zenithive.com", true},
		// case-insensitive
		{"uppercase prefix", "DEMO.admin@zenithive.com", true},
		{"mixed case", "Demo.Manager@Zenithive.Com", true},
		// real accounts — must NOT be suppressed
		{"real employee", "john.doe@zenithive.com", false},
		{"real admin", "admin@zenithive.com", false},
		{"empty string", "", false},
		// edge: demo prefix but different domain
		{"demo wrong domain", "demo.admin@gmail.com", false},
		// edge: zenithive domain but no demo prefix
		{"no demo prefix", "notdemo.admin@zenithive.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDemoEmail(tt.email)
			if got != tt.want {
				t.Errorf("IsDemoEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

// ─── FilterDemoRecipients ─────────────────────────────────────────────────────

func TestFilterDemoRecipients(t *testing.T) {
	tests := []struct {
		name       string
		recipients []string
		want       []string
	}{
		{
			name:       "all demo — returns empty slice",
			recipients: []string{"demo.admin@zenithive.com", "demo.hr@zenithive.com"},
			want:       []string{},
		},
		{
			name:       "no demo — returns all",
			recipients: []string{"alice@zenithive.com", "bob@zenithive.com"},
			want:       []string{"alice@zenithive.com", "bob@zenithive.com"},
		},
		{
			name: "mixed — strips only demo addresses",
			recipients: []string{
				"demo.manager@zenithive.com",
				"real.manager@zenithive.com",
				"demo.admin@zenithive.com",
				"hr@zenithive.com",
			},
			want: []string{"real.manager@zenithive.com", "hr@zenithive.com"},
		},
		{
			name:       "empty input — returns empty slice",
			recipients: []string{},
			want:       []string{},
		},
		{
			name:       "nil input — returns empty slice",
			recipients: nil,
			want:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterDemoRecipients(tt.recipients)

			if len(got) != len(tt.want) {
				t.Fatalf("FilterDemoRecipients() len = %d, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("FilterDemoRecipients()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ─── SendEmail skips demo addresses ──────────────────────────────────────────
// SendEmail returns nil (no error) when the recipient is a demo account,
// even without a valid Resend config in the environment.

func TestSendEmail_SkipsDemoAccount(t *testing.T) {
	err := SendEmail("demo.employee@zenithive.com", "Test Subject", "Test Body")
	if err != nil {
		t.Errorf("SendEmail to demo account should return nil, got: %v", err)
	}
}

// ─── SendEmailToMultiple skips when all recipients are demo ──────────────────

func TestSendEmailToMultiple_AllDemoReturnsNil(t *testing.T) {
	recipients := []string{
		"demo.admin@zenithive.com",
		"demo.superadmin@zenithive.com",
	}
	err := SendEmailToMultiple(recipients, "Subject", "Body")
	if err != nil {
		t.Errorf("SendEmailToMultiple with all-demo recipients should return nil, got: %v", err)
	}
}
