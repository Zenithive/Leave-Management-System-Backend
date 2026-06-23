package handlers

import (
	"fmt"
	"log/slog"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/config"
	models2 "github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/providers"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/templates"
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

func (h *LeaveNotificationHandler) OnLeaveApplied(d *models.LeaveNotificationData) {

	for _, recipient := range d.Recipients {

		vm := templates.LeaveAppliedVM(
			appName(h.cfg), h.cfg.APP_URL,
			d.EmployeeName, d.EmployeeEmail, d.LeaveType,
			d.StartDate, d.EndDate, d.Days, d.Reason,
		)

		vm.RecipientName = recipient.FullName

		body, err := templates.Render("leave.html", vm)
		if err != nil {
			continue
		}

		_ = h.email.Send(
			recipient.Email,
			fmt.Sprintf("Leave Application - %s", d.EmployeeName),
			body,
		)
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

	h.renderAndSendBulk(
		hrVM,
		"Leave Withdrawn - "+d.EmployeeName,
		recipientEmails(d.Recipients),
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

func (h *LeaveNotificationHandler) renderAndSendRecipients(
	vm templates.LeaveVM,
	subject string,
	recipients []models2.Recipient,
) {
	for _, recipient := range recipients {

		vm.RecipientName = recipient.FullName

		body, err := templates.Render("leave.html", vm)
		if err != nil {
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
				"err", err,
			)
		}
	}
}
