package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type ResendConfig struct {
	APIKey string
	From   string
}

// GetResendConfig reads Resend configuration from environment variables
func GetResendConfig() (*ResendConfig, error) {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("RESEND_FROM")

	if apiKey == "" || from == "" {
		return nil, fmt.Errorf("missing Resend configuration: ensure RESEND_API_KEY and RESEND_FROM are set")
	}

	return &ResendConfig{
		APIKey: apiKey,
		From:   from,
	}, nil
}

// ResendEmailRequest represents the email request payload for Resend API
type ResendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text,omitempty"`
	HTML    string   `json:"html,omitempty"`
}

// ResendEmailResponse represents the response from Resend API
type ResendEmailResponse struct {
	ID      string   `json:"id"`
	From    string   `json:"from"`
	To      []string `json:"to"`
	Created string   `json:"created_at"`
	Error   *struct {
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"error,omitempty"`
}

// SendEmail sends an email using Resend API
func SendEmail(to, subject, body string) error {
	config, err := GetResendConfig()
	if err != nil {
		return fmt.Errorf("Resend configuration error: %v", err)
	}

	fmt.Printf("Attempting to send email to: %s with subject: %s\n", to, subject)

	// Prepare email request
	emailReq := ResendEmailRequest{
		From:    config.From,
		To:      []string{to},
		Subject: subject,
		Text:    body,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(emailReq)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))

	// Send request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Printf("Sending email via Resend API...\n")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Resend API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Resend API response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
			return fmt.Errorf("Resend API error (status %d): %s", resp.StatusCode, errorResp.Message)
		}
		return fmt.Errorf("Resend API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var emailResp ResendEmailResponse
	if err := json.Unmarshal(bodyBytes, &emailResp); err != nil {
		return fmt.Errorf("failed to parse Resend API response: %w", err)
	}

	if emailResp.Error != nil {
		return fmt.Errorf("Resend API error: %s (status: %d)", emailResp.Error.Message, emailResp.Error.Status)
	}

	fmt.Printf("Email sent successfully to: %s (ID: %s)\n", to, emailResp.ID)
	return nil
}

// SendEmailToMultiple sends email to multiple recipients using Resend API
func SendEmailToMultiple(recipients []string, subject, body string) error {
	config, err := GetResendConfig()
	if err != nil {
		return fmt.Errorf("Resend configuration error: %v", err)
	}

	if len(recipients) == 0 {
		return fmt.Errorf("no recipients provided")
	}

	fmt.Printf("Attempting to send email to %d recipients with subject: %s\n", len(recipients), subject)

	// Prepare email request
	emailReq := ResendEmailRequest{
		From:    config.From,
		To:      recipients,
		Subject: subject,
		Text:    body,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(emailReq)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))

	// Send request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Printf("Sending email to %d recipients via Resend API...\n", len(recipients))
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Resend API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read Resend API response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
			return fmt.Errorf("Resend API error (status %d): %s", resp.StatusCode, errorResp.Message)
		}
		return fmt.Errorf("Resend API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var emailResp ResendEmailResponse
	if err := json.Unmarshal(bodyBytes, &emailResp); err != nil {
		return fmt.Errorf("failed to parse Resend API response: %w", err)
	}

	if emailResp.Error != nil {
		return fmt.Errorf("Resend API error: %s (status: %d)", emailResp.Error.Message, emailResp.Error.Status)
	}

	fmt.Printf("Email sent successfully to %d recipients (ID: %s)\n", len(recipients), emailResp.ID)
	return nil
}

// SendEmployeeCreationEmail sends notification to newly created employee
func SendEmployeeCreationEmail(employeeEmail, employeeName, password string) error {
	subject := "Welcome to Zenithive - Your Account Has Been Created"
	body := fmt.Sprintf(`
Dear %s,

Welcome to Zenithive!

Your employee account has been successfully created. Below are your login credentials:

Email: %s
Password: %s

Please login to the system and change your password at your earliest convenience.

Login URL: [https://leave.zenithive.com]

If you have any questions, please contact your HR department.

Best regards,
Zenithive HR Team
`, employeeName, employeeEmail, password)

	return SendEmail(employeeEmail, subject, body)
}

// SendLeaveApplicationEmail sends notification to manager, admin, and superadmin
func SendLeaveApplicationEmail(recipients []string, employeeName, leaveType, startDate, endDate string, days float64, reason string) error {
	subject := fmt.Sprintf("Leave Application - %s", employeeName)
	body := fmt.Sprintf(`
Dear Manager/Admin,

A new leave application has been submitted and requires your review.

Employee: %s
Leave Type: %s
Start Date: %s
End Date: %s
Duration: %.1f days
Reason: %s
Status: Pending Approval

Please login to the system to approve or reject this leave request.

Best regards,
Zenithive Leave Management System
`, employeeName, leaveType, startDate, endDate, days, reason)

	return SendEmailToMultiple(recipients, subject, body)
}

func SendLeaveManagerRejectionEmail(AdminEmail []string, empEmail string, employeeName, leaveType, startDate, endDate string, days float64, rejectedBy string) error {
	subject := "Leave Request - Manager Rejection (Pending Final Decision)"

	// ------------------------
	// EMPLOYEE EMAIL (Step 1)
	// ------------------------
	empBody := fmt.Sprintf(`
Dear %s,

Your leave request has been REJECTED by your manager %s.

This is a first-level rejection. The request has now been forwarded to Admin/SuperAdmin for final review.

Leave Details:
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: MANAGER_REJECTED

For more information, please contact your manager.

Best regards,
Zenithive Leave Management System
`, employeeName, rejectedBy, leaveType, startDate, endDate, days)

	if err := SendEmail(empEmail, subject, empBody); err != nil {
		return err
	}

	// ------------------------
	// ADMIN EMAIL (Step 1)
	// ------------------------
	adminBody := fmt.Sprintf(`
Dear Admin,

A leave request has been REJECTED at manager level by %s.

This leave now requires final rejection approval from Admin/SuperAdmin.

Leave Details:
- Employee: %s
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: MANAGER_REJECTED

Please log in to the admin panel to complete the final review.

Best regards,
Zenithive Leave Management System
`, rejectedBy, employeeName, leaveType, startDate, endDate, days)

	return SendEmailToMultiple(AdminEmail, subject, adminBody)
}

// SendLeaveManagerApprovalEmail sends notification for manager-level approval (first step)
func SendLeaveManagerApprovalEmail(
	AdminEmail []string,
	employeeEmail, employeeName, leaveType, startDate, endDate string,
	days float64, approvedBy string,
) error {

	subject := "Leave Approved by Manager"

	// ------------------------
	// 1) EMPLOYEE EMAIL
	// ------------------------
	empBody := fmt.Sprintf(`
Dear %s,

Your leave application has been APPROVED by your manager %s.

Leave Details:
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: MANAGER APPROVED

Note: Your leave is now pending final approval from Admin/SuperAdmin.

Best regards,
Zenithive Leave Management System
`, employeeName, approvedBy, leaveType, startDate, endDate, days)

	if err := SendEmail(employeeEmail, subject, empBody); err != nil {
		return err
	}

	// ------------------------
	// 2) ADMIN EMAIL
	// ------------------------
	adminBody := fmt.Sprintf(`
Dear Admin,

A leave request has been APPROVED by the manager %s.

Leave Details:
- Employee: %s
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: MANAGER APPROVED

Please review and take final action.

Best regards,
Zenithive Leave Management System
`, approvedBy, employeeName, leaveType, startDate, endDate, days)

	return SendEmailToMultiple(AdminEmail, subject, adminBody)
}

// SendLeaveApprovalEmail sends notification to employee when leave is approved
func SendLeaveFinalApprovalEmail(
	AdminEmail []string,
	employeeEmail, employeeName, leaveType, startDate, endDate string,
	days float64, approvedBy string,
) error {

	subject := "Leave Approved"

	// ------------------------
	// 1) EMPLOYEE EMAIL
	// ------------------------
	empBody := fmt.Sprintf(`
Dear %s,

Your leave application has been APPROVED by %s.

Leave Details:
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: APPROVED

Enjoy your time off!

Best regards,
Zenithive Leave Management System
`, employeeName, approvedBy, leaveType, startDate, endDate, days)

	if err := SendEmail(employeeEmail, subject, empBody); err != nil {
		return err
	}

	// ------------------------
	// 2) ADMIN EMAIL
	// ------------------------
	adminBody := fmt.Sprintf(`
Dear Admin,

The leave request for employee %s has been APPROVED by %s.

Leave Details:
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: APPROVED

Best regards,
Zenithive Leave Management System
`, employeeName, approvedBy, leaveType, startDate, endDate, days)

	return SendEmailToMultiple(AdminEmail, subject, adminBody)
}

// SendLeaveRejectionEmail sends notification to employee when leave is rejected
func SendLeaveRejectionEmail(
	AdminEmail []string,
	empEmail string,
	employeeName, leaveType, startDate, endDate string,
	days float64, rejectedBy string,
) error {

	subject := "Leave Request Rejected"

	// ------------------------
	// 1) EMPLOYEE EMAIL
	// ------------------------
	empBody := fmt.Sprintf(`
Dear %s,

We regret to inform you that your leave application has been REJECTED by %s.

Leave Details:
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: REJECTED

Please contact your manager if you require more information.

Best regards,
Zenithive Leave Management System
`, employeeName, rejectedBy, leaveType, startDate, endDate, days)

	if err := SendEmail(empEmail, subject, empBody); err != nil {
		return err
	}

	// ------------------------
	// 2) ADMIN EMAIL
	// ------------------------
	adminBody := fmt.Sprintf(`
Dear Admin,

A leave request has been REJECTED by %s.

Leave Details:
- Employee: %s
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: REJECTED

Best regards,
Zenithive Leave Management System
`, rejectedBy, employeeName, leaveType, startDate, endDate, days)

	return SendEmailToMultiple(AdminEmail, subject, adminBody)
}

// SendLeaveAddedByAdminEmail sends notification to employee when admin/manager adds leave on their behalf
func SendLeaveAddedByAdminEmail(employeeEmail, employeeName, leaveType, startDate, endDate string, days float64, addedBy, addedByRole string) error {
	subject := fmt.Sprintf("Leave Added to Your Account - %s", leaveType)
	body := fmt.Sprintf(`
Dear %s,

A leave has been added to your account by %s (%s).

Leave Type: %s
Start Date: %s
End Date: %s
Duration: %.1f days
Status: APPROVED

This leave has been automatically approved and your leave balance has been updated accordingly.

If you have any questions about this leave entry, please contact your manager or HR department.

Best regards,
Zenithive Leave Management System
`, employeeName, addedBy, addedByRole, leaveType, startDate, endDate, days)

	return SendEmail(employeeEmail, subject, body)
}

// SendPasswordUpdateEmail sends notification to employee when their password is updated by admin
func SendPasswordUpdateEmail(employeeEmail, employeeName, newPassword, updatedByEmail, updatedByRole string) error {
	subject := "Your Password Has Been Updated"
	body := fmt.Sprintf(`
Dear %s,

Your account password has been updated by %s (%s).

Your new login credentials are:
Email: %s
Password: %s

If you did not request this change, please contact your HR department immediately.

For security reasons, we recommend:
1. Login with your new password
2. Change your password to something memorable
3. Keep your password secure and do not share it with anyone

Login URL: [https://leave.zenithive.com]

Best regards,
Zenithive HR Team
`, employeeName, updatedByEmail, updatedByRole, employeeEmail, newPassword)

	return SendEmail(employeeEmail, subject, body)
}

// SendLeaveCancellationEmail sends notification when leave is cancelled
func SendLeaveCancellationEmail(employeeEmail, employeeName, leaveType, startDate, endDate string, days float64) error {
	subject := "Leave Request Cancelled"
	body := fmt.Sprintf(`
Dear %s,

Your leave request has been cancelled.

Leave Type: %s
Start Date: %s
End Date: %s
Duration: %.1f days
Status: CANCELLED

If you did not cancel this leave request, please contact your manager or HR department immediately.

Best regards,
Zenithive Leave Management System
`, employeeName, leaveType, startDate, endDate, days)

	return SendEmail(employeeEmail, subject, body)
}

// SendLeaveWithdrawalPendingEmail sends notification to admins when manager requests withdrawal
func SendLeaveWithdrawalPendingEmail(recipients []string, employeeName, leaveType, startDate, endDate string, days float64, requestedBy, reason string) error {
	subject := fmt.Sprintf("Leave Withdrawal Request - %s", employeeName)

	reasonText := ""
	if reason != "" {
		reasonText = fmt.Sprintf("\nReason: %s", reason)
	}

	body := fmt.Sprintf(`
Dear Admin,

A leave withdrawal request has been submitted and requires your approval.

Employee: %s
Leave Type: %s
Start Date: %s
End Date: %s
Duration: %.1f days
Requested By: %s (MANAGER)
Status: Pending Withdrawal Approval%s

Please login to the system to approve or reject this withdrawal request.

Best regards,
Zenithive Leave Management System
`, employeeName, leaveType, startDate, endDate, days, requestedBy, reasonText)

	return SendEmailToMultiple(recipients, subject, body)
}

// SendLeaveWithdrawalEmail sends notification when approved leave is withdrawn
func SendLeaveWithdrawalEmail(
	adminEmails []string,
	employeeEmail, employeeName, leaveType, startDate, endDate string,
	days float64, withdrawnBy, withdrawnByRole, reason string,
) error {

	subject := "Leave Request Withdrawn"

	// Optional reason text
	reasonText := ""
	if reason != "" {
		reasonText = fmt.Sprintf("\nReason: %s", reason)
	}

	// ------------------------
	// 1) EMPLOYEE EMAIL
	// ------------------------
	empBody := fmt.Sprintf(`
Dear %s,

Your approved leave request has been WITHDRAWN by %s (%s).

Leave Details:
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: WITHDRAWN%s

Your leave balance has been restored. The %.1f days have been credited back to your account.

If you have any questions regarding this withdrawal, please contact your manager or HR department.

Best regards,
Zenithive Leave Management System
`, employeeName, withdrawnBy, withdrawnByRole, leaveType, startDate, endDate, days, reasonText, days)

	if err := SendEmail(employeeEmail, subject, empBody); err != nil {
		return err
	}

	// ------------------------
	// 2) ADMIN EMAIL
	// ------------------------
	adminBody := fmt.Sprintf(`
Dear Admin,

The leave request of employee %s has been WITHDRAWN by %s (%s).

Leave Details:
- Leave Type: %s
- Start Date: %s
- End Date: %s
- Duration: %.1f days
- Status: WITHDRAWN%s

The employee's leave balance has been restored.

Best regards,
Zenithive Leave Management System
`, employeeName, withdrawnBy, withdrawnByRole, leaveType, startDate, endDate, days, reasonText)

	return SendEmailToMultiple(adminEmails, subject, adminBody)
}

// SendPayslipWithdrawalEmail sends notification when payslip is withdrawn
func SendPayslipWithdrawalEmail(employeeEmail, employeeName string, month, year int, netSalary float64, withdrawnBy, withdrawnByRole, reason string) error {
	monthNames := []string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}

	subject := fmt.Sprintf("Payslip Withdrawn - %s %d", monthNames[month], year)

	reasonText := ""
	if reason != "" {
		reasonText = fmt.Sprintf("\nReason: %s", reason)
	}

	body := fmt.Sprintf(`
Dear %s,

Your payslip for %s %d has been withdrawn by %s (%s).

Pay Period: %s %d
Net Salary: ₹%.2f
Status: WITHDRAWN%s

This payslip has been marked as withdrawn and may require reprocessing. Please contact your HR department or payroll administrator for more information.

If you have any questions about this withdrawal, please reach out to your manager or HR department.

Best regards,
Zenithive Payroll Management System
`, employeeName, monthNames[month], year, withdrawnBy, withdrawnByRole, monthNames[month], year, netSalary, reasonText)

	return SendEmail(employeeEmail, subject, body)
}

// ==================== HR-SPECIFIC EMAIL FUNCTIONS ====================

// SendLeaveApplicationEmailToHR sends HR-specific notification for new leave applications
func SendLeaveApplicationEmailToHR(hrEmails []string, employeeName, employeeEmail, leaveType, startDate, endDate string, days float64, reason string) error {
	subject := fmt.Sprintf("[HR] Leave Application - %s", employeeName)
	body := fmt.Sprintf(`
Leave Application Notification

Employee: %s (%s)
Leave Type: %s
Start Date: %s
End Date: %s
Duration: %.1f days
Reason: %s
Status: Pending Approval

Best regards,
Zenithive Leave Management System
`, employeeName, employeeEmail, leaveType, startDate, endDate, days, reason)

	return SendEmailToMultiple(hrEmails, subject, body)
}

// SendLeaveApprovalEmailToHR sends HR-specific notification when leave is approved
func SendLeaveApprovalEmailToHR(hrEmails []string, employeeName, employeeEmail, leaveType, startDate, endDate string, days float64, approvedBy string) error {
	subject := fmt.Sprintf("[HR] Leave Approved - %s", employeeName)
	body := fmt.Sprintf(`
Leave Approval Record

Employee: %s (%s)
Leave Type: %s
Start Date: %s
End Date: %s
Duration: %.1f days
Approved By: %s
Status: APPROVED

Best regards,
Zenithive Leave Management System
`, employeeName, employeeEmail, leaveType, startDate, endDate, days, approvedBy)

	return SendEmailToMultiple(hrEmails, subject, body)
}

// SendLeaveRejectionEmailToHR sends HR-specific notification when leave is rejected
func SendLeaveRejectionEmailToHR(hrEmails []string, employeeName, employeeEmail, leaveType, startDate, endDate string, days float64, rejectedBy string) error {
	subject := fmt.Sprintf("[HR] Leave Rejected - %s", employeeName)
	body := fmt.Sprintf(`
Leave Rejection Record

Employee: %s (%s)
Leave Type: %s
Start Date: %s
End Date: %s
Duration: %.1f days
Rejected By: %s
Status: REJECTED

Best regards,
Zenithive Leave Management System
`, employeeName, employeeEmail, leaveType, startDate, endDate, days, rejectedBy)

	return SendEmailToMultiple(hrEmails, subject, body)
}

// SendLeaveWithdrawalEmailToHR sends HR-specific notification when leave is withdrawn
func SendLeaveWithdrawalEmailToHR(hrEmails []string, employeeName, employeeEmail, leaveType, startDate, endDate string, days float64, withdrawnBy, withdrawnByRole, reason string) error {
	subject := fmt.Sprintf("[HR] Leave Withdrawn - %s", employeeName)

	reasonText := ""
	if reason != "" {
		reasonText = fmt.Sprintf("\nReason: %s", reason)
	}

	body := fmt.Sprintf(`
Leave Withdrawal Record

Employee: %s (%s)
Leave Type: %s
Start Date: %s
End Date: %s
Duration: %.1f days
Withdrawn By: %s (%s)
Status: WITHDRAWN%s

Best regards,
Zenithive Leave Management System
`, employeeName, employeeEmail, leaveType, startDate, endDate, days, withdrawnBy, withdrawnByRole, reasonText)

	return SendEmailToMultiple(hrEmails, subject, body)
}
