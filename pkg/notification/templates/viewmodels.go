package templates

import (
	"html/template"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Shared base
// ─────────────────────────────────────────────────────────────────────────────

// AppMeta carries the branding fields present in every template.
type AppMeta struct {
	AppName string
	AppURL  string
}

// ─────────────────────────────────────────────────────────────────────────────
// Leave — single reusable template driven by LeaveEmailConfig
// ─────────────────────────────────────────────────────────────────────────────

// LeaveEmailConfig controls every dynamic part of leave.html.
// Swap this struct to get a completely different email without touching the HTML.
type LeaveEmailConfig struct {
	// Header
	HeaderColor string // CSS colour value, e.g. "#059669"
	HeaderEmoji string // leading emoji in the <h1>
	HeaderTitle string // text after the emoji

	// Intro paragraph below the greeting (may contain safe HTML like <strong>)
	Intro template.HTML

	// Table row visibility
	ShowEmployeeInfo bool   // show Employee / Email rows (HR/admin-facing emails)
	ShowReason       bool   // show Reason row
	ShowActor        bool   // show "Approved By / Rejected By / …" row
	ActorLabel       string // label for the actor row, e.g. "Approved By"

	// Status badge
	BadgeText string // e.g. "APPROVED"
	BadgeBg   string // badge background colour, e.g. "#d1fae5"
	BadgeFg   string // badge text colour, e.g. "#065f46"

	// Optional extras
	FooterNote    string // paragraph after the table, e.g. "Your balance has been restored."
	CTALabel      string // button text; empty = no button
	IsAdminNotice bool   // true → greeting says "Dear Admin / HR"
}

// LeaveVM is the single data object passed to leave.html.
type LeaveVM struct {
	AppMeta
	Config LeaveEmailConfig

	EmployeeName  string
	EmployeeEmail string

	RecipientName string
	LeaveType     string
	StartDate     string
	EndDate       string
	Days          float64
	Reason        string
	ActorName     string
	ActorEmail    string
	ActorRole     string
}

// newLeaveVM is the internal constructor used by every preset builder below.
func newLeaveVM(
	appName, appURL string,
	cfg LeaveEmailConfig,
	employeeName, employeeEmail, leaveType string,
	startDate, endDate time.Time,
	days float64,
	reason, actorName, actorEmail, actorRole string,
) LeaveVM {
	return LeaveVM{
		AppMeta:       AppMeta{AppName: appName, AppURL: appURL},
		Config:        cfg,
		EmployeeName:  employeeName,
		EmployeeEmail: employeeEmail,
		LeaveType:     leaveType,
		StartDate:     startDate.Format("2006-01-02"),
		EndDate:       endDate.Format("2006-01-02"),
		Days:          days,
		Reason:        reason,
		ActorName:     actorName,
		ActorEmail:    actorEmail,
		ActorRole:     actorRole,
	}
}

// ── Preset builders — one per event × audience ────────────────────────────

// LeaveAppliedVM builds the VM for the admin/HR new-application email.
func LeaveAppliedVM(appName, appURL string, employeeName, employeeEmail, leaveType string, startDate, endDate time.Time, days float64, reason string) LeaveVM {
	return newLeaveVM(appName, appURL, LeaveEmailConfig{
		HeaderColor:      "#4f46e5",
		HeaderEmoji:      "📋",
		HeaderTitle:      "New Leave Application",
		Intro:            template.HTML("A new leave application has been submitted and requires your review."),
		ShowEmployeeInfo: true,
		ShowReason:       true,
		ShowActor:        false,
		BadgeText:        "PENDING APPROVAL",
		BadgeBg:          "#fef3c7",
		BadgeFg:          "#92400e",
		CTALabel:         "Review Request",
		IsAdminNotice:    true,
	}, employeeName, employeeEmail, leaveType, startDate, endDate, days, reason, "", "", "")
}

// LeaveApprovedEmployeeVM builds the VM for the employee approval email.
func LeaveApprovedEmployeeVM(appName, appURL string, employeeName, employeeEmail, leaveType string, startDate, endDate time.Time, days float64, actorName, actorEmail, actorRole string) LeaveVM {
	return newLeaveVM(appName, appURL, LeaveEmailConfig{
		HeaderColor: "#059669",
		HeaderEmoji: "✅",
		HeaderTitle: "Leave Approved",
		Intro:       template.HTML("Your leave application has been <strong>approved</strong>."),
		ShowActor:   true,
		ActorLabel:  "Approved By",
		BadgeText:   "APPROVED",
		BadgeBg:     "#d1fae5",
		BadgeFg:     "#065f46",
		FooterNote:  "Enjoy your time off! 🎉",
	}, employeeName, employeeEmail, leaveType, startDate, endDate, days, "", actorName, actorEmail, actorRole)
}

// LeaveApprovedHRVM builds the VM for the HR record copy of an approval.
func LeaveApprovedHRVM(appName, appURL string, employeeName, employeeEmail, leaveType string, startDate, endDate time.Time, days float64, actorName, actorEmail, actorRole string) LeaveVM {
	return newLeaveVM(appName, appURL, LeaveEmailConfig{
		HeaderColor:      "#0284c7",
		HeaderEmoji:      "📂",
		HeaderTitle:      "[HR Record] Leave Approved",
		Intro:            template.HTML("This is an HR record notification."),
		ShowEmployeeInfo: true,
		ShowActor:        true,
		ActorLabel:       "Approved By",
		BadgeText:        "APPROVED",
		BadgeBg:          "#d1fae5",
		BadgeFg:          "#065f46",
		IsAdminNotice:    true,
	}, employeeName, employeeEmail, leaveType, startDate, endDate, days, "", actorName, actorEmail, actorRole)
}

// LeaveRejectedEmployeeVM builds the VM for the employee rejection email.
func LeaveRejectedEmployeeVM(appName, appURL string, employeeName, employeeEmail, leaveType string, startDate, endDate time.Time, days float64, actorName, actorEmail, actorRole string) LeaveVM {
	return newLeaveVM(appName, appURL, LeaveEmailConfig{
		HeaderColor: "#dc2626",
		HeaderEmoji: "❌",
		HeaderTitle: "Leave Request Rejected",
		Intro:       template.HTML("Your leave request has been <strong>rejected</strong>."),
		ShowActor:   true,
		ActorLabel:  "Rejected By",
		BadgeText:   "REJECTED",
		BadgeBg:     "#fee2e2",
		BadgeFg:     "#991b1b",
		FooterNote:  "Please contact your manager if you have questions.",
	}, employeeName, employeeEmail, leaveType, startDate, endDate, days, "", actorName, actorEmail, actorRole)
}

// LeaveRejectedHRVM builds the VM for the HR record copy of a rejection.
func LeaveRejectedHRVM(appName, appURL string, employeeName, employeeEmail, leaveType string, startDate, endDate time.Time, days float64, actorName, actorEmail, actorRole string) LeaveVM {
	return newLeaveVM(appName, appURL, LeaveEmailConfig{
		HeaderColor:      "#0284c7",
		HeaderEmoji:      "📂",
		HeaderTitle:      "[HR Record] Leave Rejected",
		Intro:            template.HTML("This is an HR record notification."),
		ShowEmployeeInfo: true,
		ShowActor:        true,
		ActorLabel:       "Rejected By",
		BadgeText:        "REJECTED",
		BadgeBg:          "#fee2e2",
		BadgeFg:          "#991b1b",
		IsAdminNotice:    true,
	}, employeeName, employeeEmail, leaveType, startDate, endDate, days, "", actorName, actorEmail, actorRole)
}

// LeaveWithdrawalPendingVM builds the VM for the admin/HR withdrawal-pending email.
func LeaveWithdrawalPendingVM(appName, appURL string, employeeName, employeeEmail, leaveType string, startDate, endDate time.Time, days float64, actorName, actorEmail, actorRole string) LeaveVM {
	return newLeaveVM(appName, appURL, LeaveEmailConfig{
		HeaderColor:      "#d97706",
		HeaderEmoji:      "⏳",
		HeaderTitle:      "Leave Withdrawal Pending",
		Intro:            template.HTML("A leave withdrawal has been initiated and is awaiting further confirmation."),
		ShowEmployeeInfo: true,
		ShowActor:        true,
		ActorLabel:       "Initiated By",
		BadgeText:        "WITHDRAWAL PENDING",
		BadgeBg:          "#fef3c7",
		BadgeFg:          "#92400e",
		CTALabel:         "Review Withdrawal",
		IsAdminNotice:    true,
	}, employeeName, employeeEmail, leaveType, startDate, endDate, days, "", actorName, actorEmail, actorRole)
}

// LeaveWithdrawnEmployeeVM builds the VM for the employee withdrawal email.
func LeaveWithdrawnEmployeeVM(appName, appURL string, employeeName, employeeEmail, leaveType string, startDate, endDate time.Time, days float64, actorName, actorEmail, actorRole string) LeaveVM {
	return newLeaveVM(appName, appURL, LeaveEmailConfig{
		HeaderColor: "#7c3aed",
		HeaderEmoji: "↩️",
		HeaderTitle: "Leave Withdrawn",
		Intro:       template.HTML("Your approved leave has been <strong>withdrawn</strong>."),
		ShowActor:   true,
		ActorLabel:  "Withdrawn By",
		BadgeText:   "WITHDRAWN",
		BadgeBg:     "#ede9fe",
		BadgeFg:     "#5b21b6",
		FooterNote:  "Your leave balance has been restored.",
	}, employeeName, employeeEmail, leaveType, startDate, endDate, days, "", actorName, actorEmail, actorRole)
}

// LeaveWithdrawnHRVM builds the VM for the HR record copy of a withdrawal.
func LeaveWithdrawnHRVM(appName, appURL string, employeeName, employeeEmail, leaveType string, startDate, endDate time.Time, days float64, actorName, actorEmail, actorRole string) LeaveVM {
	return newLeaveVM(appName, appURL, LeaveEmailConfig{
		HeaderColor:      "#0284c7",
		HeaderEmoji:      "📂",
		HeaderTitle:      "[HR Record] Leave Withdrawn",
		Intro:            template.HTML("This is an HR record notification."),
		ShowEmployeeInfo: true,
		ShowActor:        true,
		ActorLabel:       "Withdrawn By",
		BadgeText:        "WITHDRAWN",
		BadgeBg:          "#ede9fe",
		BadgeFg:          "#5b21b6",
		FooterNote:       "Leave balance has been restored.",
		IsAdminNotice:    true,
	}, employeeName, employeeEmail, leaveType, startDate, endDate, days, "", actorName, actorEmail, actorRole)
}

// LeaveCancelledVM builds the VM for the employee cancellation email.
func LeaveCancelledVM(appName, appURL string, employeeName, employeeEmail, leaveType string, startDate, endDate time.Time, days float64) LeaveVM {
	return newLeaveVM(appName, appURL, LeaveEmailConfig{
		HeaderColor: "#6b7280",
		HeaderEmoji: "🚫",
		HeaderTitle: "Leave Request Cancelled",
		Intro:       template.HTML("Your leave request has been <strong>cancelled</strong>."),
		BadgeText:   "CANCELLED",
		BadgeBg:     "#f3f4f6",
		BadgeFg:     "#374151",
	}, employeeName, employeeEmail, leaveType, startDate, endDate, days, "", "", "", "")
}

// ─────────────────────────────────────────────────────────────────────────────
// Employee view-models (unchanged)
// ─────────────────────────────────────────────────────────────────────────────

// EmployeeCreatedVM is the data passed to employee_created.html.
type EmployeeCreatedVM struct {
	AppMeta
	EmployeeName      string
	EmployeeEmail     string
	GeneratedPassword string
}

// PasswordChangedVM is the data passed to password_changed.html.
type PasswordChangedVM struct {
	AppMeta
	EmployeeName  string
	EmployeeEmail string
	NewPassword   string
	ActorEmail    string
	ActorRole     string
}
