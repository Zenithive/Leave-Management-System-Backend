package middleware

import (
	"net/http"
	"strings"

	"github.com/Zenithive/LeaveManagementSystem/internal/handler"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/Zenithive/LeaveManagementSystem/pkg/security"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware verifies Bearer JWT Token
func AuthMiddleware(h *handler.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {

		// 1. Read Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			errors.RespondWithError(c, http.StatusUnauthorized, "Missing Authorization header")
			c.Abort()
			return
		}

		// 2. Allow both:
		//    - "Bearer <token>"
		//    - "<token>"
		var tokenString string

		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		tokenString = strings.TrimSpace(tokenString)

		if tokenString == "" {
			errors.RespondWithError(c, http.StatusUnauthorized, "Token missing")
			c.Abort()
			return
		}

		// 3. Validate JWT token
		claims, err := security.ValidateToken(tokenString, h.Env.SECRET_KEY)
		if err != nil {
			errors.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired token"+err.Error())
			c.Abort()
			return
		}

		// 4. Store useful info in context
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.UserRole)

		// Continue request
		c.Next()
	}
}
