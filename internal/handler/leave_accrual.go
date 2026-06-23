package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	utils "github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg"
)

// TriggerLeaveAccrual - POST /api/admin/leave-accrual/run
//
// Manually triggers the monthly leave accrual job for a given month and year.
// Only SUPERADMIN can call this.
//
// Query params:
//
//	?month=5&year=2026   (defaults to current month/year if omitted)
//
// This is useful when:
//   - The server was down on the 1st of the month and the cron missed
//   - You want to test accrual in a dev environment without waiting
//   - You need to back-fill a specific month
//
// The job is idempotent — running it twice for the same month is safe.
func (h *HandlerFunc) TriggerLeaveAccrual(c *gin.Context) {
	// Only SUPERADMIN can trigger this
	role := c.GetString("role")
	if role != "SUPERADMIN" {
		utils.RespondWithError(c, http.StatusForbidden, "only SUPERADMIN can trigger leave accrual")
		return
	}

	if h.LeaveAccrual == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "leave accrual service not initialized")
		return
	}

	// Parse optional month/year query params — default to current month/year
	now := time.Now()
	monthInt := int(now.Month())
	yearInt := now.Year()

	if m := c.Query("month"); m != "" {
		v, err := strconv.Atoi(m)
		if err != nil || v < 1 || v > 12 {
			utils.RespondWithError(c, http.StatusBadRequest, "invalid month: must be 1-12")
			return
		}
		monthInt = v
	}
	if y := c.Query("year"); y != "" {
		v, err := strconv.Atoi(y)
		if err != nil || v < 2000 || v > 2100 {
			utils.RespondWithError(c, http.StatusBadRequest, "invalid year: must be 2000-2100")
			return
		}
		yearInt = v
	}

	summary := h.LeaveAccrual.RunAccrualWithSummary(time.Month(monthInt), yearInt)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}
