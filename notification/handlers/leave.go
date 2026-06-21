package handlers

import (
	"fmt"
	"log/slog"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/providers"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"
)

// LeaveNotificationHandler handles all leave-related notification events.
// It owns the email body composition for leave events and delegates delivery
// to the injected EmailProvider — keeping transport and content decoupled.
type LeaveNotificationHandler struct {
	email  providers.EmailProvider
	logger *slog.Logger
	cfg    *config.ENV
}

func NewLeaveNotificationHandler(email providers.EmailProvider, logger *slog.Logger, cfg *config.ENV) *LeaveNotificationHandler {
	return &LeaveNotificationHandler{email: email, logger: logger, cfg: cfg}
}

// ─────────────────────────────────────────────────────────────────────────────
// Public event handlers — one per notification type
// ─────────────────────────────────────────────────────────────────────────────

// OnLeaveApplied notifies admins/HR that a new leave application is pending.
func (h *LeaveNotificationHandler) OnLeaveApplied(d *models.LeaveNotificationData) {
	app := appName(h.cfg)
	subject := fmt.Sprintf("Leave Application - %s", d.EmployeeName)
	body := fmt.Sprintf(`Dear Manager/Admin,

A new leave application has been submitted and requires your review.

Employee  : %s
Email     : %s
Leave Type: %s
From      : %s
To        : %s
Duration  : %.1f day(s)
Reason    : %s
Status    : Pending Approval

Please log in to review this request.

%s`,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02"),
		d.Days, d.Reason, app,
	)
	h.sendBulk(append(d.AdminEmails, d.HREmails...), subject, body)
}

// OnLeaveApproved notifies the employee and HR that the leave is fully approved.
func (h *LeaveNotificationHandler) OnLeaveApproved(d *models.LeaveNotificationData) {
	app := appName(h.cfg)
	subject := "Your Leave Has Been Approved"

	empBody := fmt.Sprintf(`Dear %s,

Your leave application has been APPROVED by %s (%s).

Leave Type: %s
From      : %s
To        : %s
Duration  : %.1f day(s)
Status    : APPROVED

Enjoy your time off!

%s`,
		d.EmployeeName, d.ActorName, d.ActorRole,
		d.LeaveType, d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02"),
		d.Days, app,
	)
	h.send(d.EmployeeEmail, subject, empBody)

	hrBody := fmt.Sprintf(`[HR Record] Leave Approved

Employee  : %s (%s)
Leave Type: %s
From      : %s
To        : %s
Duration  : %.1f day(s)
Approved By: %s (%s)
Status    : APPROVED

%s`,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02"),
		d.Days, d.ActorName, d.ActorRole, app,
	)
	h.sendBulk(d.HREmails, "[HR] "+subject, hrBody)
}

// OnLeaveRejected notifies the employee and HR that the leave is rejected.
func (h *LeaveNotificationHandler) OnLeaveRejected(d *models.LeaveNotificationData) {
	app := appName(h.cfg)
	subject := "Your Leave Request Has Been Rejected"

	empBody := fmt.Sprintf(`Dear %s,

Your leave request has been REJECTED by %s (%s).

Leave Type: %s
From      : %s
To        : %s
Duration  : %.1f day(s)
Status    : REJECTED

Please contact your manager if you have questions.

%s`,
		d.EmployeeName, d.ActorName, d.ActorRole,
		d.LeaveType, d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02"),
		d.Days, app,
	)
	h.send(d.EmployeeEmail, subject, empBody)

	hrBody := fmt.Sprintf(`[HR Record] Leave Rejected

Employee  : %s (%s)
Leave Type: %s
From      : %s
To        : %s
Duration  : %.1f day(s)
Rejected By: %s (%s)
Status    : REJECTED

%s`,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02"),
		d.Days, d.ActorName, d.ActorRole, app,
	)
	h.sendBulk(d.HREmails, "[HR] Leave Rejected - "+d.EmployeeName, hrBody)
}

// OnLeaveWithdrawalPending notifies admins that a withdrawal has been initiated
// and is awaiting higher-stage confirmation.
func (h *LeaveNotificationHandler) OnLeaveWithdrawalPending(d *models.LeaveNotificationData) {
	app := appName(h.cfg)
	subject := fmt.Sprintf("Leave Withdrawal Pending - %s", d.EmployeeName)
	body := fmt.Sprintf(`Dear Admin,

A leave withdrawal has been initiated by %s (%s) and is awaiting further confirmation.

Employee  : %s
Leave Type: %s
From      : %s
To        : %s
Duration  : %.1f day(s)
Status    : WITHDRAWAL_PENDING

Please log in to review.

%s`,
		d.ActorName, d.ActorRole,
		d.EmployeeName, d.LeaveType,
		d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02"),
		d.Days, app,
	)
	h.sendBulk(append(d.AdminEmails, d.HREmails...), subject, body)
}

// OnLeaveWithdrawn notifies the employee and HR that the leave is fully withdrawn
// and the balance has been restored.
func (h *LeaveNotificationHandler) OnLeaveWithdrawn(d *models.LeaveNotificationData) {
	app := appName(h.cfg)
	subject := "Your Leave Has Been Withdrawn"

	empBody := fmt.Sprintf(`Dear %s,

Your approved leave has been WITHDRAWN by %s (%s).

Leave Type: %s
From      : %s
To        : %s
Duration  : %.1f day(s)
Status    : WITHDRAWN

Your leave balance has been restored.

%s`,
		d.EmployeeName, d.ActorName, d.ActorRole,
		d.LeaveType, d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02"),
		d.Days, app,
	)
	h.send(d.EmployeeEmail, subject, empBody)

	hrBody := fmt.Sprintf(`[HR Record] Leave Withdrawn

Employee  : %s (%s)
Leave Type: %s
From      : %s
To        : %s
Duration  : %.1f day(s)
Withdrawn By: %s (%s)
Status    : WITHDRAWN

Balance has been restored.

%s`,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02"),
		d.Days, d.ActorName, d.ActorRole, app,
	)
	h.sendBulk(d.HREmails, "[HR] Leave Withdrawn - "+d.EmployeeName, hrBody)
}

// OnLeaveCancelled notifies the employee that their leave is cancelled.
func (h *LeaveNotificationHandler) OnLeaveCancelled(d *models.LeaveNotificationData) {
	app := appName(h.cfg)
	subject := "Leave Request Cancelled"
	body := fmt.Sprintf(`Dear %s,

Your leave request has been CANCELLED.

Leave Type: %s
From      : %s
To        : %s
Duration  : %.1f day(s)
Status    : CANCELLED

%s`,
		d.EmployeeName, d.LeaveType,
		d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02"),
		d.Days, app,
	)
	h.send(d.EmployeeEmail, subject, body)
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

func (h *LeaveNotificationHandler) send(to, subject, body string) {
	if err := h.email.Send(to, subject, body); err != nil {
		h.logger.Error("leave notification send failed", "to", to, "subject", subject, "err", err)
	}
}

func (h *LeaveNotificationHandler) sendBulk(recipients []string, subject, body string) {
	if len(recipients) == 0 {
		return
	}
	if err := h.email.SendBulk(recipients, subject, body); err != nil {
		h.logger.Error("leave notification bulk send failed", "subject", subject, "err", err)
	}
}
