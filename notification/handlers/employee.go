package handlers

import (
	"fmt"
	"log/slog"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/providers"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"
)

// EmployeeNotificationHandler handles employee lifecycle notification events.
type EmployeeNotificationHandler struct {
	email  providers.EmailProvider
	logger *slog.Logger
	cfg    *config.ENV
}

func NewEmployeeNotificationHandler(email providers.EmailProvider, logger *slog.Logger, cfg *config.ENV) *EmployeeNotificationHandler {
	return &EmployeeNotificationHandler{email: email, logger: logger, cfg: cfg}
}

// OnEmployeeCreated sends the welcome email with auto-generated credentials.
func (h *EmployeeNotificationHandler) OnEmployeeCreated(d *models.EmployeeNotificationData) {
	name := appName(h.cfg)
	subject := fmt.Sprintf("Welcome to %s — Your Account Has Been Created", name)
	body := fmt.Sprintf(`Dear %s,

Welcome to %s!

Your employee account has been created. Below are your login credentials:

Email   : %s
Password: %s

Please log in and change your password immediately.
%s
If you have questions, contact your HR department.

%s HR Team`,
		d.EmployeeName, name,
		d.EmployeeEmail, d.GeneratedPassword,
		loginURL(h.cfg), name,
	)
	if err := h.email.Send(d.EmployeeEmail, subject, body); err != nil {
		h.logger.Error("employee created notification failed", "email", d.EmployeeEmail, "err", err)
	}
}

// OnPasswordChanged sends the new credentials to the employee.
func (h *EmployeeNotificationHandler) OnPasswordChanged(d *models.EmployeeNotificationData) {
	name := appName(h.cfg)
	subject := "Your Password Has Been Updated"
	body := fmt.Sprintf(`Dear %s,

Your account password has been updated by %s (%s).

New credentials:
Email   : %s
Password: %s

If you did not request this change, contact HR immediately.
%s
%s HR Team`,
		d.EmployeeName, d.ActorEmail, d.ActorRole,
		d.EmployeeEmail, d.NewPassword,
		loginURL(h.cfg), name,
	)
	if err := h.email.Send(d.EmployeeEmail, subject, body); err != nil {
		h.logger.Error("password changed notification failed", "email", d.EmployeeEmail, "err", err)
	}
}
