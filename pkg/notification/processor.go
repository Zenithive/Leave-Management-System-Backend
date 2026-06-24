package notification

import (
	"fmt"
	"log/slog"

	"github.com/Zenithive/LeaveManagementSystem/pkg/notification/handlers"
	notifmodels "github.com/Zenithive/LeaveManagementSystem/pkg/notification/models"
)

// EventProcessor routes a notification Event to the correct domain handler.
// It is the single dispatch point — the worker pool calls Process() for every
// event dequeued from the channel.
//
// Design: adding a new event type never requires touching the worker pool or
// the service — only this file and the relevant handler. (OCP)
type EventProcessor struct {
	leaveHandler    *handlers.LeaveNotificationHandler
	employeeHandler *handlers.EmployeeNotificationHandler
	logger          *slog.Logger
}

func NewEventProcessor(
	leaveHandler *handlers.LeaveNotificationHandler,
	employeeHandler *handlers.EmployeeNotificationHandler,
	logger *slog.Logger,
) *EventProcessor {
	return &EventProcessor{
		leaveHandler:    leaveHandler,
		employeeHandler: employeeHandler,
		logger:          logger,
	}
}

// Process dispatches the event to the appropriate domain handler.
// Unknown event types are logged and dropped — workers never panic.
func (p *EventProcessor) Process(event Event) {
	switch event.Type {

	// ── Leave ─────────────────────────────────────────────────────────────────
	case LeaveApplied:
		if d, ok := castPayload[*notifmodels.LeaveNotificationData](event, p.logger); ok {
			p.leaveHandler.OnLeaveApplied(d)
		}
	case LeaveApproved:
		if d, ok := castPayload[*notifmodels.LeaveNotificationData](event, p.logger); ok {
			p.leaveHandler.OnLeaveApproved(d)
		}
	case LeaveRejected:
		if d, ok := castPayload[*notifmodels.LeaveNotificationData](event, p.logger); ok {
			p.leaveHandler.OnLeaveRejected(d)
		}
	case LeaveWithdrawalPending:
		if d, ok := castPayload[*notifmodels.LeaveNotificationData](event, p.logger); ok {
			p.leaveHandler.OnLeaveWithdrawalPending(d)
		}
	case LeaveWithdrawn:
		if d, ok := castPayload[*notifmodels.LeaveNotificationData](event, p.logger); ok {
			p.leaveHandler.OnLeaveWithdrawn(d)
		}
	case LeaveCancelled:
		if d, ok := castPayload[*notifmodels.LeaveNotificationData](event, p.logger); ok {
			p.leaveHandler.OnLeaveCancelled(d)
		}

	// ── Employee ──────────────────────────────────────────────────────────────
	case EmployeeCreated:
		if d, ok := castPayload[*notifmodels.EmployeeNotificationData](event, p.logger); ok {
			p.employeeHandler.OnEmployeeCreated(d)
		}
	case PasswordChanged:
		if d, ok := castPayload[*notifmodels.EmployeeNotificationData](event, p.logger); ok {
			p.employeeHandler.OnPasswordChanged(d)
		}

	default:
		p.logger.Warn("notification: unhandled event type dropped", "type", event.Type)
	}
}

// castPayload is a generic type-assertion helper.
// On mismatch it logs a structured error and returns false — never panics.
func castPayload[T any](event Event, logger *slog.Logger) (T, bool) {
	v, ok := event.Data.(T)
	if !ok {
		var zero T
		logger.Error("notification: payload type mismatch",
			"event_type", event.Type,
			"expected", fmt.Sprintf("%T", zero),
			"got", fmt.Sprintf("%T", event.Data),
		)
		return zero, false
	}
	return v, true
}
