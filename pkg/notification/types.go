package notification

// Type identifies which kind of notification event occurred.
type Type string

const (
	// Leave events
	LeaveApplied          Type = "LEAVE_APPLIED"
	LeaveApproved         Type = "LEAVE_APPROVED"
	LeaveRejected         Type = "LEAVE_REJECTED"
	LeaveWithdrawn        Type = "LEAVE_WITHDRAWN"
	LeaveWithdrawalPending Type = "LEAVE_WITHDRAWAL_PENDING"
	LeaveCancelled        Type = "LEAVE_CANCELLED"

	// Employee events
	EmployeeCreated Type = "EMPLOYEE_CREATED"
	EmployeeUpdated Type = "EMPLOYEE_UPDATED"

	// Auth events
	PasswordReset   Type = "PASSWORD_RESET"
	PasswordChanged Type = "PASSWORD_CHANGED"
)

// Event is the generic envelope published onto the notification bus.
// Data must be one of the typed payloads defined in notification/models/.
type Event struct {
	Type Type
	Data any
}
