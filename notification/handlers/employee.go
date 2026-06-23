package handlers

import (
	"log/slog"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/providers"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/notification/templates"
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
	vm := templates.EmployeeCreatedVM{
		AppMeta:           templates.AppMeta{AppName: name, AppURL: h.cfg.APP_URL},
		EmployeeName:      d.EmployeeName,
		EmployeeEmail:     d.EmployeeEmail,
		GeneratedPassword: d.GeneratedPassword,
	}
	body, err := templates.Render("employee_created.html", vm)
	if err != nil {
		h.logger.Error("employee_created template render failed", "err", err)
		return
	}
	subject := "Welcome to " + name + " — Your Account Has Been Created"
	if err := h.email.Send(d.EmployeeEmail, subject, body); err != nil {
		h.logger.Error("employee created notification failed", "email", d.EmployeeEmail, "err", err)
	}
}

// OnPasswordChanged sends the new credentials to the employee.
func (h *EmployeeNotificationHandler) OnPasswordChanged(d *models.EmployeeNotificationData) {
	vm := templates.PasswordChangedVM{
		AppMeta:       templates.AppMeta{AppName: appName(h.cfg), AppURL: h.cfg.APP_URL},
		EmployeeName:  d.EmployeeName,
		EmployeeEmail: d.EmployeeEmail,
		NewPassword:   d.NewPassword,
		ActorEmail:    d.ActorEmail,
		ActorRole:     d.ActorRole,
	}
	body, err := templates.Render("password_changed.html", vm)
	if err != nil {
		h.logger.Error("password_changed template render failed", "err", err)
		return
	}
	if err := h.email.Send(d.EmployeeEmail, "Your Password Has Been Updated", body); err != nil {
		h.logger.Error("password changed notification failed", "email", d.EmployeeEmail, "err", err)
	}
}
