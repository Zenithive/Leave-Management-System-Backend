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
