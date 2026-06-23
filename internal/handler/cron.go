package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
)

// DailyLeaveSlackNotification — cron endpoint that sends the daily leave summary to Slack.
//
// GET /api/cron/daily-leave-slack?token=<CRON_SECRET_TOKEN>
//
// The endpoint is a no-op (200, no Slack message) when:
//   - Today is Saturday or Sunday
//   - Today is a configured holiday in Tbl_Holiday
//   - There are no approved leaves active today
//
// Authentication: CRON_SECRET_TOKEN in ?token= query param or X-Cron-Token header.
func (h *HandlerFunc) DailyLeaveSlackNotification(c *gin.Context) {
	// 1️ Authenticate cron request
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("X-Cron-Token")
	}
	if token == "" || token != h.Env.CRON_SECRET {
		utils.RespondWithError(c, http.StatusUnauthorized, "Invalid or missing cron token")
		return
	}

	// 2️ Check Slack webhook is configured
	if h.Env.EXTERNAL_API_URL == "" {
		fmt.Println("[Cron] EXTERNAL_API_URL not configured")
		utils.RespondWithError(c, http.StatusServiceUnavailable, "Slack webhook not configured")
		return
	}

	// 3️ Skip weekends and holidays (reusable guard)
	now := time.Now()
	skip, err := utils.ShouldSkipCronToday(now, h.Query)
	if err != nil {
		fmt.Printf("[Cron] Holiday check error: %v\n", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "Holiday check failed: "+err.Error())
		return
	}
	if skip != nil {
		fmt.Printf("[Cron] Skipping notification — %s\n", skip.Error())
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("Skipped: today is a %s", skip.Reason),
			"date":    skip.Date,
		})
		return
	}

	// 4️ Fetch today's approved/active leaves
	leaves, err := h.Query.GetTodaysActiveLeaves()
	if err != nil {
		fmt.Printf("[Cron] Failed to fetch today's leaves: %v\n", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch leaves: "+err.Error())
		return
	}

	today := now.Format("2006-01-02")

	// 5️ Nothing on leave today — skip silently
	if len(leaves) == 0 {
		fmt.Printf("[Cron] No active leaves for %s — notification skipped\n", today)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("No active leaves today (%s) — notification skipped", today),
			"matched": 0,
			"date":    today,
		})
		return
	}

	fmt.Printf("[Cron] Found %d active leave(s) for %s\n", len(leaves), today)

	// 6️ Format and send Slack message
	slackService := service.NewDailyLeaveSlackService(h.Env.EXTERNAL_API_URL)
	message := slackService.FormatSlackTable(today, leaves)

	if err := slackService.SendToSlack(message); err != nil {
		fmt.Printf("[Cron] Failed to send to Slack: %v\n", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to send to Slack: "+err.Error())
		return
	}

	fmt.Printf("[Cron] ✅ Slack leave summary sent (%d leaves, %s)\n", len(leaves), today)

	// 7️ Success
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Leave summary sent to Slack",
		"matched": len(leaves),
		"date":    today,
	})
}
