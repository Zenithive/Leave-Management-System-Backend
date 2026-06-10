package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
)

// DailyLeaveSlackNotification - Cron endpoint to send daily leave summary to Slack
// GET /api/cron/daily-leave-slack?token=<CRON_SECRET_TOKEN>
//
// This endpoint mimics the Supabase Edge Function behavior:
// 1. Fetches all leaves active today (start_date <= today AND end_date >= today)
// 2. Formats them into a Slack markdown table
// 3. Sends to configured Slack webhook
//
// Authentication: Requires CRON_SECRET_TOKEN in query param or X-Cron-Token header
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

	// 2️ Check if Slack webhook is configured
	if h.Env.EXTERNAL_API_URL == "" {
		fmt.Println("[Cron] EXTERNAL_API_UR not configured")
		utils.RespondWithError(c, http.StatusServiceUnavailable, "Slack webhook not configured")
		return
	}

	// 3️ Fetch today's active leaves
	leaves, err := h.Query.GetTodaysActiveLeaves()
	if err != nil {
		fmt.Printf("[Cron] Failed to fetch today's leaves: %v\n", err)
		utils.RespondWithError(c, http.StatusInternalServerError,
			"Failed to fetch leaves: "+err.Error())
		return
	}

	today := time.Now().Format("2006-01-02")

	// 4️ If no leaves, return early (optional: still send empty notification)
	if len(leaves) == 0 {
		fmt.Printf("[Cron] No active leaves found for %s\n", today)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("No active leaves today (%s)", today),
			"matched": 0,
			"date":    today,
		})
		return
	}

	fmt.Printf("[Cron] Found %d active leave(s) for %s\n", len(leaves), today)

	// 5️ Format Slack message
	slackService := service.NewDailyLeaveSlackService(h.Env.EXTERNAL_API_URL)
	message := slackService.FormatSlackTable(today, leaves)

	// 6️ Send to Slack
	err = slackService.SendToSlack(message)
	if err != nil {
		fmt.Printf("[Cron] Failed to send to Slack: %v\n", err)
		utils.RespondWithError(c, http.StatusInternalServerError,
			"Failed to send to Slack: "+err.Error())
		return
	}

	fmt.Printf("[Cron] ✅ Successfully sent leave summary to Slack (%d leaves)\n", len(leaves))

	// 7️ Success response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Leave summary sent to Slack",
		"matched": len(leaves),
		"date":    today,
	})
}
