package constant

const (
	LEAVE_REJECT    = "REJECT"
	LEAVE_APPLOVED  = "APPROVED"
	LEAVE_REJECTED  = "REJECTED"
	LEAVE_CANCELLED = "CANCELLED"
	LEAVE_WITHDRAWN = "WITHDRAWN"
	LEAVE_PENDING   = "Pending"

	// Intermediate statuses
	LEAVE_MANAGER_APPROVED   = "MANAGER_APPROVED"
	LEAVE_MANAGER_REJECTED   = "MANAGER_REJECTED"
	LEAVE_WITHDRAWAL_PENDING = "WITHDRAWAL_PENDING"
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
