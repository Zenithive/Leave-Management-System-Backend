package access_role

import (
	"errors"

	"github.com/gin-gonic/gin"
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

var AdminAccessRoles = []string{
	constant.ROLE_SUPER_ADMIN,
	constant.ROLE_ADMIN,
	constant.ROLE_HR,
}

var EmployeeAccessRoles = []string{
	constant.ROLE_EMPLOYEE,
	constant.ROLE_INTERN,
}

var SuperAdminOnly = []string{
	constant.ROLE_SUPER_ADMIN,
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}

		role, ok := roleValue.(string)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid role"})
			return
		}

		// check role in allowed list
		for _, r := range allowedRoles {
			if role == r {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
	}
}
