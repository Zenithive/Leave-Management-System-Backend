package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/Zenithive/LeaveManagementSystem/pkg/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EmployeeAuthData struct {
	ID       string `db:"id"`
	Email    string `db:"email"`
	Password string `db:"password"`
	Role     string `db:"role"`
	Status   string `db:"status"`
}

// GetAllRoles — GET /api/auth/roles
// Returns all available role types from Tbl_Role.
// Public endpoint — no authentication required (used for registration dropdowns etc.)
func (s *HandlerFunc) GetAllRoles(c *gin.Context) {
	roles, err := s.Query.GetAllRoles()
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch roles: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Roles fetched successfully",
		"data":    roles,
	})
}

func (s *HandlerFunc) Login(c *gin.Context) {
	// 0. Check if user is already authenticated
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		var tokenString string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			tokenString = authHeader
		}

		// If token exists and is valid, user is already logged in
		if tokenString != "" {
			claims, err := security.ValidateToken(tokenString, s.Env.SERACT_KEY)
			if err == nil {
				// Token is valid, user already logged in
				userID, parseErr := uuid.Parse(claims.UserID)
				if parseErr == nil {
					emp, empErr := s.Query.GetEmployeeByID(userID)
					if empErr == nil && emp.Status != "deactive" {
						c.JSON(http.StatusOK, gin.H{
							"success": true,
							"message": "Already logged in",
							"token":   tokenString,
							"user": gin.H{
								"id":    emp.ID,
								"email": emp.Email,
								"role":  emp.Role,
							},
						})
						return
					}
				}
			}
		}
	}

	// 1. Parse request body
	var input models.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 2. Fetch employee + role name
	emp, err := s.Query.GetEmployeeByEmail(input.Email)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, fmt.Sprintf("Login failed — email not found: %v", err.Error()))
		return
	}

	// 3. Validate password
	if !security.CheckPassword(input.Password, emp.Password) {
		log.Printf("Login failed — wrong password for email: %s", input.Email)
		errors.RespondWithError(c, http.StatusUnauthorized, "Login failed — wrong password for email: "+input.Email)
		return
	}

	// 3.5 Check employee status
	if emp.Status == "deactive" {
		errors.RespondWithError(c, http.StatusForbidden, "Your account is deactivated. You cannot login")
		return
	}

	// 4. Generate JWT with role name
	token, err := security.GenerateToken(emp.ID, emp.Role, s.Env.SERACT_KEY)
	if err != nil {
		log.Printf("JWT generation error: %v", err)
		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}

	// 5. Success Response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":    emp.ID,
			"email": emp.Email,
			"role":  emp.Role,
		},
	})
}

// VerifyToken verifies if the provided token is valid and returns user data
func (s *HandlerFunc) VerifyToken(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		errors.RespondWithError(c, http.StatusUnauthorized, "Missing Authorization header")
		return
	}

	// Extract token (support both "Bearer token" and "token" formats)
	var tokenString string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenString = authHeader[7:]
	} else {
		tokenString = authHeader
	}

	if tokenString == "" {
		errors.RespondWithError(c, http.StatusUnauthorized, "Token missing")
		return
	}

	// Validate token
	claims, err := security.ValidateToken(tokenString, s.Env.SERACT_KEY)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	// Parse user ID to UUID
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, "Invalid user ID")
		return
	}

	// Get employee details from database
	emp, err := s.Query.GetEmployeeByID(userID)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, "User not found")
		return
	}

	// Check if employee is still active
	if emp.Status == "deactive" {
		errors.RespondWithError(c, http.StatusForbidden, "Account is deactivated")
		return
	}

	// Return user data
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token is valid",
		"user": gin.H{
			"id":    emp.ID,
			"email": emp.Email,
			"role":  emp.Role,
		},
	})
}

// CheckAuthStatus checks if user is authenticated without requiring authentication
func (s *HandlerFunc) CheckAuthStatus(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")

	// No token provided
	if authHeader == "" {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"message":       "No token provided",
		})
		return
	}

	// Extract token
	var tokenString string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenString = authHeader[7:]
	} else {
		tokenString = authHeader
	}

	if tokenString == "" {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"message":       "Token missing",
		})
		return
	}

	// Validate token
	claims, err := security.ValidateToken(tokenString, s.Env.SERACT_KEY)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"message":       "Invalid or expired token",
		})
		return
	}

	// Parse user ID and check user exists
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"message":       "Invalid user ID",
		})
		return
	}

	emp, err := s.Query.GetEmployeeByID(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"message":       "User not found",
		})
		return
	}

	// Check if user is active
	if emp.Status == "deactive" {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"message":       "Account is deactivated",
		})
		return
	}

	// User is authenticated
	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"message":       "User is authenticated",
		"user": gin.H{
			"id":    emp.ID,
			"email": emp.Email,
			"role":  emp.Role,
		},
	})
}

func (s *HandlerFunc) Logout(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userRoleRaw, _ := c.Get("role")

	expiredToken, err := security.GenerateExpiredToken(userIDRaw.(string), userRoleRaw.(string), s.Env.SERACT_KEY)
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to generate expired token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
		"token":   expiredToken,
	})
}
