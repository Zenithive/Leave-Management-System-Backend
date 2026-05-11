package access_role

import (
	"errors"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

func Admin_SuperAdmin_Hr(role string, message string) error {
	if role != constant.ROLE_SUPER_ADMIN && role != constant.ROLE_ADMIN && role != constant.ROLE_HR {
		return errors.New(message)
	}
	return nil
}

func Admin_SuperAdmin(role string, message string) error {
	if role != constant.ROLE_SUPER_ADMIN && role != constant.ROLE_ADMIN {
		return errors.New(message)
	}
	return nil
}

func SuperAdmin(role string, message string) error {
	if role != constant.ROLE_SUPER_ADMIN {
		return errors.New(message)
	}
	return nil
}

// IsEmployeeLike returns true for roles that have employee-level access (EMPLOYEE and INTERN).
func IsEmployeeLike(role string) bool {
	return role == constant.ROLE_EMPLOYEE || role == constant.ROLE_INTERN
}
