package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Zenithive/LeaveManagementSystem/internal/service"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/gin-gonic/gin"
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
		errors.RespondWithError(c, http.StatusUnauthorized, "Invalid or missing cron token")
		return
	}

	// 2️ Check Slack webhook is configured
	if h.Env.EXTERNAL_API_URL == "" {
		slog.Warn("[Cron] EXTERNAL_API_URL not configured")
		errors.RespondWithError(c, http.StatusServiceUnavailable, "Slack webhook not configured")
		return
	}

	// 3️ Skip weekends and holidays (reusable guard)
	now := time.Now()
	skip, err := service.ShouldSkipCronToday(now, h.Query)
	if err != nil {
		slog.Error("[Cron] holiday check error", "err", err)
		errors.RespondWithError(c, http.StatusInternalServerError, "Holiday check failed: "+err.Error())
		return
	}
	if skip != nil {
		slog.Info("[Cron] skipping notification", "reason", skip.Reason, "date", skip.Date)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Skipped: today is a " + skip.Reason,
			"date":    skip.Date,
		})
		return
	}

	// 4️ Fetch today's approved/active leaves
	leaves, err := h.Query.GetTodaysActiveLeaves()
	if err != nil {
		slog.Error("[Cron] failed to fetch today's leaves", "err", err)
		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch leaves: "+err.Error())
		return
	}

	today := now.Format("2006-01-02")

	// 5️ Nothing on leave today — skip silently
	if len(leaves) == 0 {
		slog.Info("[Cron] no active leaves, notification skipped", "date", today)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "No active leaves today (" + today + ") — notification skipped",
			"matched": 0,
			"date":    today,
		})
		return
	}

	// 6️ Format and send Slack message
	slackService := service.NewDailyLeaveSlackService(h.Env.EXTERNAL_API_URL)
	message := slackService.FormatSlackTable(today, leaves)

	if err := slackService.SendToSlack(message); err != nil {
		slog.Error("[Cron] failed to send to Slack", "err", err)
		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to send to Slack: "+err.Error())
		return
	}

	slog.Info("[Cron] Slack leave summary sent", "leaves", len(leaves), "date", today)

	// 7️ Success
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Leave summary sent to Slack",
		"matched": len(leaves),
		"date":    today,
	})
}
