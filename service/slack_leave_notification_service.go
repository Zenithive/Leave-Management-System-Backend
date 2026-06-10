package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

type DailyLeaveSlackService struct {
	SlackWebhookURL string
}

func NewDailyLeaveSlackService(webhookURL string) *DailyLeaveSlackService {
	return &DailyLeaveSlackService{
		SlackWebhookURL: webhookURL,
	}
}

// SlackBlock represents Slack's Block Kit structure
type SlackMessage struct {
	Blocks []SlackBlock `json:"blocks"`
}

type SlackBlock struct {
	Type string     `json:"type"`
	Text *SlackText `json:"text,omitempty"`
}

type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// FormatSlackTable creates the Slack markdown table (matches Deno formatSlackTable)
func (s *DailyLeaveSlackService) FormatSlackTable(date string, leaves []models.DailyLeaveRecord) SlackMessage {
	// Header row
	header := []string{"Employee", "Type", "From", "To", "Days", "Status", "Approved By"}

	// Data rows
	rows := [][]string{}
	for _, leave := range leaves {
		approvedBy := "-"
		if leave.ApprovedBy != nil {
			approvedBy = *leave.ApprovedBy
		}

		row := []string{
			leave.EmployeeName,
			leave.LeaveType,
			leave.StartDate.Format("2006-01-02"),
			leave.EndDate.Format("2006-01-02"),
			fmt.Sprintf("%.1f", leave.Days),
			leave.Status,
			approvedBy,
		}
		rows = append(rows, row)
	}

	// Calculate column widths
	colWidths := make([]int, len(header))
	for i := range header {
		maxWidth := len(header[i])
		for _, row := range rows {
			if len(row[i]) > maxWidth {
				maxWidth = len(row[i])
			}
		}
		colWidths[i] = maxWidth
	}

	// Format rows with padding
	formatRow := func(row []string) string {
		formatted := make([]string, len(row))
		for i, cell := range row {
			formatted[i] = padRight(cell, colWidths[i])
		}
		return join(formatted, " | ")
	}

	// Create separator
	separatorParts := make([]string, len(colWidths))
	for i, width := range colWidths {
		separatorParts[i] = repeatChar('-', width)
	}
	separator := join(separatorParts, "-|-")

	// Build table
	tableLines := []string{formatRow(header), separator}
	for _, row := range rows {
		tableLines = append(tableLines, formatRow(row))
	}
	table := join(tableLines, "\n")

	// Create Slack message with code block formatting
	return SlackMessage{
		Blocks: []SlackBlock{
			{
				Type: "section",
				Text: &SlackText{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Leave Summary (%s)*\nTotal: %d\n```\n%s\n```",
						date, len(leaves), table),
				},
			},
		},
	}
}

// SendToSlack sends the formatted message to Slack webhook
func (s *DailyLeaveSlackService) SendToSlack(message SlackMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	resp, err := http.Post(s.SlackWebhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send to Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack returned status %d", resp.StatusCode)
	}

	return nil
}

// Helper functions
func join(parts []string, sep string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += sep
		}
		result += part
	}
	return result
}

func repeatChar(char rune, count int) string {
	result := make([]rune, count)
	for i := range result {
		result[i] = char
	}
	return string(result)
}

func padRight(str string, width int) string {
	if len(str) >= width {
		return str
	}
	padding := repeatChar(' ', width-len(str))
	return str + padding
}
