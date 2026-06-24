package models

// EmployeeNotificationData carries fields for employee lifecycle notifications.
type EmployeeNotificationData struct {
	EmployeeID    string
	EmployeeName  string
	EmployeeEmail string

	// Populated for EmployeeCreated — the auto-generated password to deliver.
	GeneratedPassword string

	// Populated for PasswordChanged — the new plain-text password.
	NewPassword string

	// Who performed the action (admin, HR, superadmin).
	ActorEmail string
	ActorRole  string
}
