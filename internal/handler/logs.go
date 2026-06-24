package handler

import (
	"net/http"
	"strconv"
	"time"

	accessrole "github.com/Zenithive/LeaveManagementSystem/pkg/accessrole"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/gin-gonic/gin"
)

// GetLogs - only for super_admin to get logs filtered by days
func (h *HandlerFunc) GetLogs(c *gin.Context) {
	// Check if user is SUPERADMIN
	role := c.GetString("role")

	if err := accessrole.SuperAdmin(role, "Access denied. Only SUPERADMIN can view logs"); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "Invalid days parameter. Must be a positive integer")
		return
	}
	// Get days parameter from query (default to 7 days if not provided or empty)
	daysParam := c.Query("days")
	days := 7 // Default value

	// If parameter is provided and not empty, try to parse it
	if daysParam != "" {
		parsedDays, err := strconv.Atoi(daysParam)
		if err != nil || parsedDays < 1 {
			errors.RespondWithError(c, http.StatusInternalServerError, "Invalid days parameter. Must be a positive integer")
			return
		}
		days = parsedDays
	}
	// Calculate the date threshold
	dateThreshold := time.Now().AddDate(0, 0, -days)
	// Query to get logs with user names, filtered by days
	logs, err := h.Query.GetLogs(dateThreshold)
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "failed to get logs")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Logs retrieved successfully",
		"data": gin.H{
			"logs":        logs,
			"total_count": len(logs),
			"days_filter": days,
			"date_from":   dateThreshold.Format("2006-01-02"),
		},
	})
}
