package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/handler"
	pkg "github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg"
)

// AuthMiddleware verifies Bearer JWT Token
func AuthMiddleware(h *handler.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {

		// 1. Read Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			pkg.RespondWithError(c, http.StatusUnauthorized, "Missing Authorization header")
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
			pkg.RespondWithError(c, http.StatusUnauthorized, "Token missing")
			c.Abort()
			return
		}

		// 3. Validate JWT token
		claims, err := pkg.ValidateToken(tokenString, h.Env.SERACT_KEY)
		if err != nil {
			pkg.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired token"+err.Error())
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
