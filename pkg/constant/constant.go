package constant

const (
	LEAVE_REJECT             = "REJECT"
	LEAVE_APPLOVED           = "APPROVED"
	LEAVE_REJECTED           = "REJECTED"
	LEAVE_CANCELLED          = "CANCELLED"
	LEAVE_WITHDRAWN          = "WITHDRAWN"
	LEAVE_PENDING            = "Pending"
	LEAVE_WITHDRAWAL_PENDING = "WITHDRAWAL_PENDING"
)

type BirthdayStatus string

const (
	StatusToday    BirthdayStatus = "TODAY"
	StatusUpcoming BirthdayStatus = "UPCOMING"
	StatusPast     BirthdayStatus = "PAST"
)
