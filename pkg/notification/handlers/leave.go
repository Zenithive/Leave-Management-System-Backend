package handlers

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Zenithive/LeaveManagementSystem/internal/config"
	models2 "github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/notification/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/notification/providers"
	"github.com/Zenithive/LeaveManagementSystem/pkg/notification/templates"
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
// Public event handlers
// ─────────────────────────────────────────────────────────────────────────────

// OnLeaveApplied notifies every recipient (manager, admin, HR, etc.) that a
// new leave request has been submitted.
//
// FIXED: previously used `_ = h.email.Send(...)`, which discarded the error
// returned by Send. This masked a real bug — see the throttling fix below.
//
// FIXED: previously never set vm.RecipientName before rendering, unlike
// every other On* method in this file.
//
// FIXED: Resend's API allows only 2 requests/second. Looping over more than
// 2 recipients with no delay caused the 3rd+ Send() calls to fail with
// HTTP 429 "rate_limit_exceeded". A small delay between sends keeps this
// loop under that limit regardless of recipient count.
func (h *LeaveNotificationHandler) OnLeaveApplied(d *models.LeaveNotificationData) {

	for i, recipient := range d.Recipients {

		// Resend allows 2 requests/second. Sleeping ~1000ms between sends
		// (after the first) keeps us safely under that limit even with
		// network jitter, without slowing down the common case of 1-2
		// recipients at all.
		if i > 0 {
			time.Sleep(1000 * time.Millisecond)
		}

		vm := templates.LeaveAppliedVM(
			appName(h.cfg), h.cfg.APP_URL,
			d.EmployeeName, d.EmployeeEmail, d.LeaveType,
			d.StartDate, d.EndDate, d.Days, d.Reason,
		)

		vm.RecipientName = recipient.FullName

		body, err := templates.Render("leave.html", vm)
		if err != nil {
			h.logger.Error("leave template render failed",
				"template", "leave.html",
				"recipient", recipient.Email,
				"err", err,
			)
			continue
		}

		if err := h.email.Send(
			recipient.Email,
			fmt.Sprintf("Leave Application - %s", d.EmployeeName),
			body,
		); err != nil {
			h.logger.Error("leave notification send failed",
				"event", "LEAVE_APPLIED",
				"to", recipient.Email,
				"err", err,
			)
		}
	}
}

func (h *LeaveNotificationHandler) OnLeaveApproved(d *models.LeaveNotificationData) {
	empVM := templates.LeaveApprovedEmployeeVM(
		appName(h.cfg), h.cfg.APP_URL,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate, d.EndDate, d.Days,
		d.ActorName, d.ActorEmail, d.ActorRole,
	)

	h.renderAndSend(
		empVM,
		"Your Leave Has Been Approved",
		d.EmployeeEmail,
	)

	hrVM := templates.LeaveApprovedHRVM(
		appName(h.cfg), h.cfg.APP_URL,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate, d.EndDate, d.Days,
		d.ActorName, d.ActorEmail, d.ActorRole,
	)

	h.renderAndSendRecipients(
		hrVM,
		"Leave Approved - "+d.EmployeeName,
		d.Recipients,
	)

}

func (h *LeaveNotificationHandler) OnLeaveRejected(d *models.LeaveNotificationData) {
	empVM := templates.LeaveRejectedEmployeeVM(
		appName(h.cfg), h.cfg.APP_URL,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate, d.EndDate, d.Days,
		d.ActorName, d.ActorEmail, d.ActorRole,
	)

	h.renderAndSend(
		empVM,
		"Your Leave Request Has Been Rejected",
		d.EmployeeEmail,
	)

	hrVM := templates.LeaveRejectedHRVM(
		appName(h.cfg), h.cfg.APP_URL,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate, d.EndDate, d.Days,
		d.ActorName, d.ActorEmail, d.ActorRole,
	)

	h.renderAndSendRecipients(
		hrVM,
		"Leave Rejected - "+d.EmployeeName,
		d.Recipients,
	)

}

func (h *LeaveNotificationHandler) OnLeaveWithdrawalPending(d *models.LeaveNotificationData) {
	vm := templates.LeaveWithdrawalPendingVM(
		appName(h.cfg), h.cfg.APP_URL,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate, d.EndDate, d.Days,
		d.ActorName, d.ActorEmail, d.ActorRole,
	)

	h.renderAndSendRecipients(
		vm,
		fmt.Sprintf("Leave Withdrawal Pending - %s", d.EmployeeName),
		d.Recipients,
	)

}

func (h *LeaveNotificationHandler) OnLeaveWithdrawn(d *models.LeaveNotificationData) {
	empVM := templates.LeaveWithdrawnEmployeeVM(
		appName(h.cfg), h.cfg.APP_URL,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate, d.EndDate, d.Days,
		d.ActorName, d.ActorEmail, d.ActorRole,
	)

	h.renderAndSendRecipients(
		empVM,
		"Your Leave Has Been Withdrawn",
		d.Recipients,
	)

	hrVM := templates.LeaveWithdrawnHRVM(
		appName(h.cfg), h.cfg.APP_URL,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate, d.EndDate, d.Days,
		d.ActorName, d.ActorEmail, d.ActorRole,
	)

	h.renderAndSendRecipients(
		hrVM,
		"Leave Withdrawn - "+d.EmployeeName,
		d.Recipients,
	)

}

func (h *LeaveNotificationHandler) OnLeaveCancelled(d *models.LeaveNotificationData) {
	vm := templates.LeaveCancelledVM(
		appName(h.cfg), h.cfg.APP_URL,
		d.EmployeeName, d.EmployeeEmail, d.LeaveType,
		d.StartDate, d.EndDate, d.Days,
	)
	h.renderAndSend(vm, "Leave Request Cancelled", d.EmployeeEmail)
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// renderAndSend renders leave.html with vm and sends to a single recipient.
func (h *LeaveNotificationHandler) renderAndSend(vm templates.LeaveVM, subject, to string) {
	body, err := templates.Render("leave.html", vm)
	if err != nil {
		h.logger.Error("leave template render failed", "template", "leave.html", "subject", subject, "err", err)
		return
	}
	if err := h.email.Send(to, subject, body); err != nil {
		h.logger.Error("leave notification send failed", "to", to, "subject", subject, "err", err)
	}
}

// renderAndSendBulk renders leave.html with vm and sends to multiple recipients.
func (h *LeaveNotificationHandler) renderAndSendBulk(vm templates.LeaveVM, subject string, recipients []string) {
	if len(recipients) == 0 {
		return
	}
	body, err := templates.Render("leave.html", vm)
	if err != nil {
		h.logger.Error("leave template render failed", "template", "leave.html", "subject", subject, "err", err)
		return
	}
	if err := h.email.SendBulk(recipients, subject, body); err != nil {
		h.logger.Error("leave notification bulk send failed", "subject", subject, "err", err)
	}
}

// renderAndSendRecipients renders vm for each recipient (personalizing
// RecipientName) and sends individually.
//
// FIXED: Resend's API allows only 2 requests/second. This helper is used by
// OnLeaveApproved, OnLeaveRejected, and OnLeaveWithdrawalPending — any of
// them can have 3+ recipients (manager, admin, HR, superadmin), which
// previously caused the 3rd+ Send() calls to fail with HTTP 429
// "rate_limit_exceeded". A small delay between sends keeps this loop under
// that limit regardless of recipient count.
func (h *LeaveNotificationHandler) renderAndSendRecipients(
	vm templates.LeaveVM,
	subject string,
	recipients []models2.Recipient,
) {
	for i, recipient := range recipients {

		if i > 0 {
			time.Sleep(1000 * time.Millisecond)
		}

		vm.RecipientName = recipient.FullName

		body, err := templates.Render("leave.html", vm)
		if err != nil {
			h.logger.Error("leave template render failed",
				"template", "leave.html",
				"subject", subject,
				"recipient", recipient.Email,
				"err", err,
			)
			continue
		}

		if err := h.email.Send(
			recipient.Email,
			subject,
			body,
		); err != nil {
			h.logger.Error(
				"leave notification send failed",
				"email", recipient.Email,
				"subject", subject,
				"err", err,
			)
		}
	}
}
