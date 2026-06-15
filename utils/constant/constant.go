package constant

const (
	LEAVE_APPROVE   = "APPROVE"
	LEAVE_REJECT    = "REJECT"
	LEAVE_APPLOVED  = "APPROVED"
	LEAVE_REJECTED  = "REJECTED"
	LEAVE_CANCELLED = "CANCELLED"

	// Manager first-stage (IsEarly / WFH two-stage via manager)
	LEAVE_MANAGER_REJECTED = "MANAGER_REJECTED"
	LEAVE_MANAGER_APPROVED = "MANAGER_APPROVED"

	// Admin/HR first-stage for Default and WFH leave types
	// (pending final approval/rejection by SuperAdmin)
	LEAVE_ADMIN_APPROVED = "ADMIN_APPROVED"
	LEAVE_ADMIN_REJECTED = "ADMIN_REJECTED"

	LEAVE_WITHDRAWAL_PENDING = "WITHDRAWAL_PENDING"
	LEAVE_WITHDRAWN          = "WITHDRAWN"
)

const (
	ROLE_SUPER_ADMIN = "SUPERADMIN"
	ROLE_ADMIN       = "ADMIN"
	ROLE_EMPLOYEE    = "EMPLOYEE"
	ROLE_MANAGER     = "MANAGER"
	ROLE_HR          = "HR"
	ROLE_INTERN      = "INTERN"
)

type BirthdayStatus string

const (
	StatusToday    BirthdayStatus = "TODAY"
	StatusUpcoming BirthdayStatus = "UPCOMING"
	StatusPast     BirthdayStatus = "PAST"
)
