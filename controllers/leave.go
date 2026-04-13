package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

// AdminAddLeave - POST /api/leaves/admin-add

func (h *HandlerFunc) ApplyLeave(c *gin.Context) {

	// Extract Employee Info
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "Employee ID missing")
		return
	}

	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee ID format")
		return
	}

	employeeID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee UUID")
		return
	}

	// Validate Employee Status
	empStatus, err := h.Query.GetEmployeeStatus(employeeID)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to verify employee status")
		return
	}
	if empStatus == "deactive" {
		utils.RespondWithError(c, 403, "Your account is deactivated. You cannot apply leave")
		return
	}

	// Bind Input
	var input models.LeaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	input.EmployeeID = employeeID

	var leaveTiming time.Time = time.Time{} // Zero value indicates not provided

	// If LeaveTiming string is provided, validate it
	if input.LeaveTiming != nil {
		leaveTiming, err = service.ValidateLeaveTiming(*input.LeaveTiming)
		fmt.Println(leaveTiming)
		if err != nil {
			utils.RespondWithError(c, 400, err.Error())
			return
		}
	}

	// Validate timing ID if provided (must be 1, 2, or 3)
	if input.LeaveTimingID != nil && (*input.LeaveTimingID < 1 || *input.LeaveTimingID > 3) {
		utils.RespondWithError(c, 400, "Invalid leave timing ID. Must be 1 (First Half), 2 (Second Half), or 3 (Full Day)")
		return
	}

	// Validate Reason
	input.Reason = strings.TrimSpace(input.Reason)
	if len(input.Reason) < 10 {
		utils.RespondWithError(c, 400, "Leave reason must be at least 10 characters long")
		return
	}
	if len(input.Reason) > 500 {
		utils.RespondWithError(c, 400, "Leave reason is too long. Maximum 500 characters allowed")
		return
	}

	// // Validate Dates
	// now := time.Now()
	// cutoff := now.Add(-12 * time.Hour)

	// if input.StartDate.Before(cutoff) {
	// 	utils.RespondWithError(c, 400, "Start date cannot be earlier than today")
	// 	return
	// }
	if input.EndDate.Before(input.StartDate) {
		utils.RespondWithError(c, 400, "End date cannot be earlier than start date")
		return
	}

	// Final Leave ID to return
	var leaveID uuid.UUID
	var Days float64

	// Execute Transaction
	err = common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {

		// Fetch Leave Type first to check IsEarly
		leaveType, err := h.Query.GetLeaveTypeByIdTx(tx, input.LeaveTypeID)
		if err == sql.ErrNoRows {
			return utils.CustomErr(c, 400, "Invalid leave type")
		}
		if err != nil {
			return utils.CustomErr(c, 500, "Failed to fetch leave type: "+err.Error())
		}

		// Determine timing ID based on IsEarly flag
		timingID := 3 // Default to Full Day
		if input.LeaveTimingID != nil {
			timingID = *input.LeaveTimingID
		}

		//For IsEarly leave types, timing is not applicable
		if leaveType.IsEarly != nil && *leaveType.IsEarly {
			timingID = 3 // Force full day for early leave types
		}

		// Working days with timing consideration
		leaveDays, err := service.CalculateWorkingDaysWithTiming(h.Query, tx, input.StartDate, input.EndDate, timingID, leaveTiming)
		if err != nil {
			return utils.CustomErr(c, 400, err.Error())
		}
		if leaveDays <= 0 {
			return utils.CustomErr(c, 400, "Calculated leave days must be greater than zero. Please check the dates and timing")
		}
		input.Days = &leaveDays
		Days = leaveDays

		// Leave Balance - Skip balance check for IsEarly leave types
		if leaveType.IsEarly == nil || *leaveType.IsEarly == false {
			fmt.Println("leave", leaveType.IsEarly)
			balance, err := h.Query.GetLeaveBalance(tx, employeeID, input.LeaveTypeID)
			if err == sql.ErrNoRows {
				// Create balance if it doesn't exist
				balance = float64(leaveType.DefaultEntitlement)
				if err := h.Query.CreateLeaveBalance(tx, employeeID, input.LeaveTypeID, leaveType.DefaultEntitlement); err != nil {
					return utils.CustomErr(c, 500, "Failed to create leave balance: "+err.Error())
				}
			} else if err != nil {
				return utils.CustomErr(c, 500, "Failed to fetch leave balance: "+err.Error())
			}

			// Check balance
			if balance < leaveDays {
				return utils.CustomErr(c, 400, "Insufficient leave balance")
			}
		}

		// Overlapping Leave
		overlaps, err := h.Query.GetOverlappingLeaves(tx, employeeID, input.StartDate, input.EndDate)
		if err != nil {
			return utils.CustomErr(c, 500, "Failed to check overlapping leave")
		}
		if len(overlaps) > 0 {
			ov := overlaps[0]
			return utils.CustomErr(c, 400, fmt.Sprintf(
				"Overlapping leave exists: %s from %s to %s (Status: %s). Please cancel or modify the existing leave first",
				ov.LeaveType,
				ov.StartDate.Format("2006-01-02"),
				ov.EndDate.Format("2006-01-02"),
				ov.Status,
			))
		}

		// Insert Leave
		// For IsEarly leave types, pass the leave_timing string
		var leaveTimingStr *string
		if leaveType.IsEarly != nil && *leaveType.IsEarly && input.LeaveTiming != nil {
			leaveTimingStr = input.LeaveTiming
		}

		id, err := h.Query.InsertLeave(tx, employeeID, input.LeaveTypeID, timingID, input.StartDate, input.EndDate, leaveDays, input.Reason, leaveTimingStr)
		if err != nil {
			return utils.CustomErr(c, 500, "Failed to apply leave: "+err.Error())
		}
		leaveID = id

		// Log Entry
		data := &models.Common{
			Component:  constant.ComponentLeave,
			Action:     constant.ActionCreate,
			FromUserID: employeeID,
		}
		if err := h.Query.AddLog(data, tx); err != nil {
			return utils.CustomErr(c, 500, "Failed to create leave log: "+err.Error())
		}

		return nil // IMPORTANT FIX
	})

	// If transaction returned an error, stop (CustomErr already responded)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update leave: "+err.Error())
		return
	}

	// go func() {
	// 	leaveType, _ := h.Query.GetLeaveTypeById(input.LeaveTypeID)

	// 	recipients, err := h.Query.GetAdminAndEmployeeEmail(employeeID)

	// 	if err != nil {
	// 		fmt.Printf("Failed to get notification recipients: %v\n", err)
	// 		return
	// 	}

	// 	empDetails, err := h.Query.GetEmployeeDetailsForNotification(employeeID)
	// 	if err != nil {
	// 		fmt.Printf("Failed to get employee details for notification: %v\n", err)
	// 		return
	// 	}

	// 	if len(recipients) > 0 {
	// 		utils.SendLeaveApplicationEmail(
	// 			recipients,
	// 			empDetails.FullName,
	// 			leaveType.Name,
	// 			input.StartDate.Format("2006-01-02"),
	// 			input.EndDate.Format("2006-01-02"),
	// 			Days,
	// 			input.Reason,
	// 		)

	// 		// Send HR-specific email
	// 		var hrEmails []string
	// 		h.Query.DB.Select(&hrEmails, `
	// 			SELECT e.email
	// 			FROM Tbl_Employee e
	// 			JOIN Tbl_Role r ON e.role_id = r.id
	// 			WHERE r.type = 'HR' AND e.status = 'active'
	// 		`)
	// 		if len(hrEmails) > 0 {
	// 			utils.SendLeaveApplicationEmailToHR(
	// 				hrEmails,
	// 				empDetails.FullName,
	// 				empDetails.Email,
	// 				leaveType.Name,
	// 				input.StartDate.Format("2006-01-02"),
	// 				input.EndDate.Format("2006-01-02"),
	// 				Days,
	// 				input.Reason,
	// 			)
	// 		}
	// 	}
	// }()

	// Send response
	c.JSON(200, gin.H{
		"message":  "Leave applied successfully",
		"leave_id": leaveID,
		"days":     Days,
		"reason":   input.Reason,
	})
}

// ActionLeave - POST /api/leaves/:id/action
// Two-level approval/rejection system:
// APPROVAL FLOW:
// 1. MANAGER approves → Status: MANAGER_APPROVED (no balance deduction)
// 2. ADMIN/SUPERADMIN finalizes → Status: APPROVED (balance deducted)
// REJECTION FLOW:
// 1. MANAGER rejects → Status: MANAGER_REJECTED (pending final rejection)
// 2. ADMIN/SUPERADMIN finalizes → Status: REJECTED (final rejection)

func (s *HandlerFunc) ActionLeave(c *gin.Context) {
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)

	if role == "EMPLOYEE" {
		utils.RespondWithError(c, 403, "Employees cannot approve leaves")
		return
	}

	approverIDRaw, _ := c.Get("user_id")
	approverID, _ := uuid.Parse(approverIDRaw.(string))

	leaveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, 400, "Invalid leave ID")
		return
	}

	var body struct {
		Action string `json:"action" validate:"required"` // APPROVE/REJECT
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondWithError(c, 400, "Invalid payload: "+err.Error())
		return
	}

	body.Action = strings.ToUpper(body.Action)
	if body.Action != constant.LEAVE_APPROVE && body.Action != constant.LEAVE_REJECT {
		utils.RespondWithError(c, 400, "Action must be APPROVE or REJECT")
		return
	}

	tx, err := s.Query.DB.Beginx()
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	leave, err := s.Query.GetLeaveById(tx, leaveID)

	if err != nil {
		utils.RespondWithError(c, 404, "Leave not found: "+err.Error())
		return
	}

	//  Prevent self-approval
	if leave.EmployeeID == approverID {
		utils.RespondWithError(c, 403, "You cannot approve your own leave request")
		return
	}

	//  MANAGER validation
	if role == "MANAGER" {
		// Check manager permission setting
		exists, err := s.Query.ChackManagerPermission()
		if err != nil {
			utils.RespondWithError(c, 500, "Failed to get manager permission")
			return
		}
		if !exists {
			utils.RespondWithError(c, 403, "Manager approval is not enabled")
			return
		}

		// Verify reporting relationship
		var managerID uuid.UUID
		err = tx.Get(&managerID, "SELECT manager_id FROM Tbl_Employee WHERE id=$1", leave.EmployeeID)
		if err != nil {
			utils.RespondWithError(c, 500, "Failed to verify reporting relationship")
			return
		}

		if managerID != approverID {
			utils.RespondWithError(c, 403, "You can only approve leaves of employees who report to you")
			return
		}

		// Manager can only act on Pending leaves
		if leave.Status != "Pending" {
			utils.RespondWithError(c, 400, fmt.Sprintf("Cannot process leave with status: %s", leave.Status))
			return
		}
	}

	//  ADMIN/SUPERADMIN validation
	if role == "ADMIN" || role == "SUPERADMIN" {
		// Admin can act on Pending, MANAGER_APPROVED, or MANAGER_REJECTED leaves
		if leave.Status != "Pending" && leave.Status != "MANAGER_APPROVED" && leave.Status != "MANAGER_REJECTED" {
			utils.RespondWithError(c, 400, fmt.Sprintf("Cannot process leave with status: %s", leave.Status))
			return
		}
	}

	// ========================================
	// REJECT ACTION - Two-Step Process
	// ========================================
	if body.Action == "REJECT" {
		// MANAGER REJECTION (First Level)
		if role == "MANAGER" {
			_, err = tx.Exec(`UPDATE Tbl_Leave SET status='MANAGER_REJECTED', approved_by=$2, updated_at=NOW() WHERE id=$1`, leaveID, approverID)
			if err != nil {
				utils.RespondWithError(c, 500, "Failed to reject leave: "+err.Error())
				return
			}

			empDetails, err := s.Query.GetEmployeeDetailsForNotification(leave.EmployeeID)
			if err != nil {
				fmt.Printf("Failed to get employee details for notification: %v\n", err)
			}

			var leaveTypeName string
			s.Query.DB.Get(&leaveTypeName, "SELECT name FROM Tbl_Leave_type WHERE id=$1", leave.LeaveTypeID)

			// Fetch approver's full name
			var approverName string
			s.Query.DB.Get(&approverName, "SELECT full_name FROM Tbl_Employee WHERE id=$1", approverID)

			recipient, err := s.Query.GetAdminAndEmployeeEmail(leave.EmployeeID)
			if err != nil {
				fmt.Printf("Failed to get admin and employee emails for notification: %v\n", err)
			}
			var recipients []string
			recipients = append(recipients, recipient...)

			tx.Commit()

			// Send final rejection notification
			if len(recipients) > 0 {
				go func() {
					utils.SendLeaveManagerRejectionEmail(
						recipients,
						empDetails.Email,
						empDetails.FullName,
						leaveTypeName,
						leave.StartDate.Format("2006-01-02"),
						leave.EndDate.Format("2006-01-02"),
						leave.Days,
						approverName,
					)

					// Send HR-specific email
					var hrEmails []string
					s.Query.DB.Select(&hrEmails, `
						SELECT e.email 
						FROM Tbl_Employee e
						JOIN Tbl_Role r ON e.role_id = r.id
						WHERE r.type = 'HR' AND e.status = 'active'
					`)
					if len(hrEmails) > 0 {
						utils.SendLeaveRejectionEmailToHR(
							hrEmails,
							empDetails.FullName,
							empDetails.Email,
							leaveTypeName,
							leave.StartDate.Format("2006-01-02"),
							leave.EndDate.Format("2006-01-02"),
							leave.Days,
							approverName,
						)
					}
				}()
			}

			c.JSON(200, gin.H{
				"message": "Leave rejected by manager. Pending final rejection from ADMIN/SUPERADMIN",
				"status":  "MANAGER_REJECTED",
			})
			return
		}

		// ADMIN/SUPERADMIN FINAL REJECTION (Second Level)
		if role == "ADMIN" || role == "SUPERADMIN" || role == "HR" {
			_, err = tx.Exec(`UPDATE Tbl_Leave SET status='REJECTED', approved_by=$2, updated_at=NOW() WHERE id=$1`, leaveID, approverID)
			if err != nil {
				utils.RespondWithError(c, 500, "Failed to finalize leave rejection: "+err.Error())
				return
			}

			// Fetch employee details
			empDetails, err := s.Query.GetEmployeeDetailsForNotification(leave.EmployeeID)
			if err != nil {
				fmt.Printf("Failed to get employee details for notification: %v\n", err)
			}

			var leaveTypeName string
			s.Query.DB.Get(&leaveTypeName, "SELECT name FROM Tbl_Leave_type WHERE id=$1", leave.LeaveTypeID)

			// Fetch approver's full name
			var approverName string
			s.Query.DB.Get(&approverName, "SELECT full_name FROM Tbl_Employee WHERE id=$1", approverID)

			recipient, err := s.Query.GetAdminAndEmployeeEmail(leave.EmployeeID)
			if err != nil {
				fmt.Printf("Failed to get admin and employee emails for notification: %v\n", err)
			}
			var recipients []string
			recipients = append(recipients, recipient...)

			tx.Commit()

			// Send final rejection notification
			if len(recipients) > 0 {
				go func() {
					utils.SendLeaveRejectionEmail(
						recipients,
						empDetails.Email,
						empDetails.FullName,
						leaveTypeName,
						leave.StartDate.Format("2006-01-02"),
						leave.EndDate.Format("2006-01-02"),
						leave.Days,
						approverName,
					)

					// Send HR-specific email
					var hrEmails []string
					s.Query.DB.Select(&hrEmails, `
						SELECT e.email 
						FROM Tbl_Employee e
						JOIN Tbl_Role r ON e.role_id = r.id
						WHERE r.type = 'HR' AND e.status = 'active'
					`)
					if len(hrEmails) > 0 {
						utils.SendLeaveRejectionEmailToHR(
							hrEmails,
							empDetails.FullName,
							empDetails.Email,
							leaveTypeName,
							leave.StartDate.Format("2006-01-02"),
							leave.EndDate.Format("2006-01-02"),
							leave.Days,
							approverName,
						)
					}
				}()
			}

			c.JSON(200, gin.H{
				"message": "Leave finalized and rejected successfully",
				"status":  "REJECTED",
			})
			return
		}

		// Should not reach here
		utils.RespondWithError(c, 500, "Unexpected error in leave rejection process")
		return
	}

	// ========================================
	// APPROVE ACTION
	// ========================================

	// Fetch leave type to check if it's an early leave
	leaveType, err := s.Query.GetLeaveTypeByIdTx(tx, leave.LeaveTypeID)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to fetch leave type: "+err.Error())
		return
	}

	// Check balance before any approval - SKIP for is_early leave types
	isEarlyLeave := leaveType.IsEarly != nil && *leaveType.IsEarly
	
	if !isEarlyLeave {
		var currentBalance float64
		err = tx.Get(&currentBalance, `
			SELECT closing 
			FROM Tbl_Leave_balance 
			WHERE employee_id=$1 AND leave_type_id=$2 
			AND year = EXTRACT(YEAR FROM CURRENT_DATE)
		`, leave.EmployeeID, leave.LeaveTypeID)

		if err != nil {
			utils.RespondWithError(c, 500, "Failed to fetch leave balance: "+err.Error())
			return
		}

		if currentBalance < leave.Days {
			utils.RespondWithError(c, 400, fmt.Sprintf("Cannot approve: Insufficient leave balance. Available: %.1f days, Required: %.1f days", currentBalance, leave.Days))
			return
		}
	}

	// MANAGER APPROVAL (First Level)
	if role == "MANAGER" {
		_, err = tx.Exec(`UPDATE Tbl_Leave SET status='MANAGER_APPROVED', approved_by=$2, updated_at=NOW() WHERE id=$1`, leaveID, approverID)
		if err != nil {
			utils.RespondWithError(c, 500, "Failed to approve leave: "+err.Error())
			return
		}

		empDetails, err := s.Query.GetEmployeeDetailsForNotification(leave.EmployeeID)
		if err != nil {
			fmt.Printf("Failed to get employee details for notification: %v\n", err)
		}

		var leaveTypeName string
		s.Query.DB.Get(&leaveTypeName, "SELECT name FROM Tbl_Leave_type WHERE id=$1", leave.LeaveTypeID)

		// Fetch approver's full name
		var approverName string
		s.Query.DB.Get(&approverName, "SELECT full_name FROM Tbl_Employee WHERE id=$1", approverID)

		tx.Commit()

		admins, err := s.Query.GetAdminAndEmployeeEmail(leave.EmployeeID)
		if err != nil {
			fmt.Printf("Failed to get admin and employee emails for notification: %v\n", err)
		}

		// Send final approval notification
		if empDetails.Email != "" {
			go func() {
				utils.SendLeaveManagerApprovalEmail(
					admins,
					empDetails.Email,
					empDetails.FullName,
					leaveTypeName,
					leave.StartDate.Format("2006-01-02"),
					leave.EndDate.Format("2006-01-02"),
					leave.Days,
					approverName,
				)

				// Send HR-specific email
				var hrEmails []string
				s.Query.DB.Select(&hrEmails, `
					SELECT e.email 
					FROM Tbl_Employee e
					JOIN Tbl_Role r ON e.role_id = r.id
					WHERE r.type = 'HR' AND e.status = 'active'
				`)
				if len(hrEmails) > 0 {
					utils.SendLeaveApprovalEmailToHR(
						hrEmails,
						empDetails.FullName,
						empDetails.Email,
						leaveTypeName,
						leave.StartDate.Format("2006-01-02"),
						leave.EndDate.Format("2006-01-02"),
						leave.Days,
						approverName,
					)
				}
			}()
		}

		c.JSON(200, gin.H{
			"message": "Leave approved by manager. Pending final approval from ADMIN/SUPERADMIN",
			"status":  "MANAGER_APPROVED",
		})
		return
	}

	// ADMIN/SUPERADMIN FINAL APPROVAL (Second Level)
	if role == "ADMIN" || role == "SUPERADMIN" {
		// Update status to APPROVED
		_, err = tx.Exec(`UPDATE Tbl_Leave SET status='APPROVED', approved_by=$2, updated_at=NOW() WHERE id=$1`, leaveID, approverID)
		if err != nil {
			utils.RespondWithError(c, 500, "Failed to finalize leave approval: "+err.Error())
			return
		}

		// Deduct from leave balance - SKIP for is_early leave types
		if !isEarlyLeave {
			_, err = tx.Exec(`UPDATE Tbl_Leave_balance SET used = used + $3, closing = closing - $3, updated_at = NOW() WHERE employee_id=$1 AND leave_type_id=$2`,
				leave.EmployeeID, leave.LeaveTypeID, leave.Days)
			if err != nil {
				utils.RespondWithError(c, 500, "Failed to update leave balance: "+err.Error())
				return
			}
		}

		// Fetch employee details
		empDetails, err := s.Query.GetEmployeeDetailsForNotification(leave.EmployeeID)
		if err != nil {
			fmt.Printf("Failed to get employee details for notification: %v\n", err)
		}

		var leaveTypeName string
		s.Query.DB.Get(&leaveTypeName, "SELECT name FROM Tbl_Leave_type WHERE id=$1", leave.LeaveTypeID)

		// Fetch approver's full name
		var approverName string
		s.Query.DB.Get(&approverName, "SELECT full_name FROM Tbl_Employee WHERE id=$1", approverID)

		tx.Commit()

		admins, err := s.Query.GetAdminAndEmployeeEmail(leave.EmployeeID)
		if err != nil {
			fmt.Printf("Failed to get admin and employee emails for notification: %v\n", err)
		}

		// Send final approval notification
		if empDetails.Email != "" {
			go func() {
				utils.SendLeaveFinalApprovalEmail(
					admins,
					empDetails.Email,
					empDetails.FullName,
					leaveTypeName,
					leave.StartDate.Format("2006-01-02"),
					leave.EndDate.Format("2006-01-02"),
					leave.Days,
					approverName,
				)

				// Send HR-specific email
				var hrEmails []string
				s.Query.DB.Select(&hrEmails, `
					SELECT e.email 
					FROM Tbl_Employee e
					JOIN Tbl_Role r ON e.role_id = r.id
					WHERE r.type = 'HR' AND e.status = 'active'
				`)
				if len(hrEmails) > 0 {
					utils.SendLeaveApprovalEmailToHR(
						hrEmails,
						empDetails.FullName,
						empDetails.Email,
						leaveTypeName,
						leave.StartDate.Format("2006-01-02"),
						leave.EndDate.Format("2006-01-02"),
						leave.Days,
						approverName,
					)
				}
			}()
		}

		c.JSON(200, gin.H{
			"message": "Leave finalized and approved successfully. Balance deducted.",
			"status":  "APPROVED",
		})
		return
	}

	// Should not reach here
	utils.RespondWithError(c, 500, "Unexpected error in leave approval process")
}

func (h *HandlerFunc) GetAllLeaves(c *gin.Context) {
	// 1️ Get Role & User ID with validation
	role := c.GetString("role")
	if role == "" {
		utils.RespondWithError(c, http.StatusUnauthorized, "Role not found in context")
		return
	}
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		utils.RespondWithError(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid user ID format: "+err.Error())
		return
	}
	// 2️ Parse query parameters for month and year filtering
	now := time.Now()
	monthStr := c.DefaultQuery("month", fmt.Sprintf("%d", int(now.Month())))
	yearStr := c.DefaultQuery("year", fmt.Sprintf("%d", now.Year()))

	// Validate month (1-12)
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid month. Must be between 1-12")
		return
	}

	// Validate year
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid year. Must be between 2000-2100")
		return
	}

	// 3️ Execute query based on role with month/year filtering
	var result []models.LeaveResponse

	switch role {
	case constant.ROLE_EMPLOYEE:
		// Employees can only see their own leaves
		result, err = h.Query.GetAllEmployeeLeaveByMonthYear(userID, month, year)
	case constant.ROLE_MANAGER:
		// Manager can see: their own leaves + their team members' leaves
		result, err = h.Query.GetAllleavebaseonassignManagerByMonthYear(userID, month, year)
	case constant.ROLE_ADMIN, constant.ROLE_HR, constant.ROLE_SUPER_ADMIN:
		result, err = h.Query.GetAllLeaveByMonthYear(month, year)
		// HR, Admin and SuperAdmin can see all leaves
	default:
		utils.RespondWithError(c, http.StatusForbidden, "Invalid role: "+role)
		return
	}
	// 4️ Handle query errors
	if err != nil {
		fmt.Printf(" GetAllLeaves DB Error: %v\n", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch leaves: "+err.Error())
		return
	}
	// 5️ Handle empty result
	if result == nil {
		result = []models.LeaveResponse{}
	}

	// 6️ Return success with metadata
	c.JSON(http.StatusOK, gin.H{
		"message": "Leaves fetched successfully",
		"total":   len(result),
		"role":    role,
		"month":   month,
		"year":    year,
		"data":    result,
	})
}

// GetAllMyLeave - GET /api/leaves/my
// Get current user's own leaves with month/year filtering
func (h *HandlerFunc) GetAllMyLeave(c *gin.Context) {
	// 1️ Get User ID with validation
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		utils.RespondWithError(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid user ID format: "+err.Error())
		return
	}

	// 2️ Parse query parameters for month and year filtering
	now := time.Now()
	monthStr := c.DefaultQuery("month", fmt.Sprintf("%d", int(now.Month())))
	yearStr := c.DefaultQuery("year", fmt.Sprintf("%d", now.Year()))

	// Validate month (1-12)
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid month. Must be between 1-12")
		return
	}

	// Validate year
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 || year > 2100 {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid year. Must be between 2000-2100")
		return
	}

	// 3️ Execute query to get user's own leaves
	result, err := h.Query.GetMyLeavesByMonthYear(userID, month, year)

	// 4️ Handle query errors
	if err != nil {
		fmt.Printf("GetAllMyLeave DB Error: %v\n", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch my leaves: "+err.Error())
		return
	}

	// 5️ Handle empty result
	if result == nil {
		result = []models.LeaveResponse{}
	}

	// 6️ Return success with metadata
	c.JSON(http.StatusOK, gin.H{
		"message": "My leaves fetched successfully",
		"total":   len(result),
		"user_id": userID,
		"month":   month,
		"year":    year,
		"data":    result,
	})
}

// CancelLeave - DELETE /api/leaves/:id/cancel
// Allows employees to cancel their own pending leaves
func (h *HandlerFunc) CancelLeave(c *gin.Context) {
	// Get user info from middleware
	userIDRaw, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDRaw.(string))

	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)

	// Parse leave ID from URL
	leaveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, 400, "Invalid leave ID")
		return
	}

	err = common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {

		leave, err := h.Query.GetLeaveById(tx, leaveID)
		if err != nil {
			if err == sql.ErrNoRows {
				return utils.CustomErr(c, http.StatusNotFound, "Leave not found")
			}
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to fetch leave: "+err.Error())
		}
		// Role validation
		if role == constant.ROLE_EMPLOYEE && leave.EmployeeID != userID {
			return utils.CustomErr(c, http.StatusForbidden, "You can only cancel your own leave applications")
		}
		// Status validation using switch
		switch leave.Status {

		case constant.LEAVE_APPLOVED:
			return utils.CustomErr(c, http.StatusBadRequest, "Cannot cancel approved leave. Please contact your manager or admin")

		case constant.LEAVE_REJECTED:
			return utils.CustomErr(c, http.StatusBadRequest, "Leave is already rejected")

		case constant.LEAVE_CANCELLED:
			return utils.CustomErr(c, http.StatusBadRequest, "Leave is already cancelled")
		}
		if err := h.Query.UpdateLeaveStatus(tx.Tx, leaveID, constant.LEAVE_CANCELLED); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to cancel leave: "+err.Error())
		}
		return nil
	})

	if err != nil {
		if appErr, ok := err.(*utils.AppError); ok {
			utils.RespondWithError(c, appErr.Code, appErr.Message)
			return
		}
		utils.RespondWithError(c, 500, "Failed to cancel leave: "+err.Error())
		return
	}

	c.JSON(200, gin.H{
		"message":  "Leave cancelled successfully",
		"leave_id": leaveID,
	})
}

// WithdrawLeave - POST /api/leaves/:id/withdraw
// Two-level withdrawal approval system:
// 1. MANAGER initiates withdrawal → Status: WITHDRAWAL_PENDING (no balance restoration)
// 2. ADMIN/SUPERADMIN finalizes → Status: WITHDRAWN (balance restored)
func (h *HandlerFunc) WithdrawLeave(c *gin.Context) {
	// 1️ Get current user info
	role := c.GetString("role")
	currentUserIDRaw, _ := c.Get("user_id")
	currentUserID, _ := uuid.Parse(currentUserIDRaw.(string))

	// 2️ Permission check - Only Admin, SUPERADMIN, and Manager can withdraw
	if role != "SUPERADMIN" && role != "ADMIN" && role != "MANAGER" {
		utils.RespondWithError(c, 403, "only SUPERADMIN, ADMIN, and MANAGER can withdraw approved leaves")
		return
	}

	// 2️A Check if MANAGER has permission to withdraw leaves
	if role == "MANAGER" {
		hasPermission, err := h.Query.ChackManagerPermission()
		if err != nil {
			utils.RespondWithError(c, http.StatusInternalServerError, "failed to check manager permission")
			return
		}
		if !hasPermission {
			utils.RespondWithError(c, 403, "MANAGER does not have permission to withdraw leaves")
			return
		}
	}

	// 3️ Parse Leave ID
	leaveIDStr := c.Param("id")
	leaveID, err := uuid.Parse(leaveIDStr)
	if err != nil {
		utils.RespondWithError(c, 400, "invalid leave ID")
		return
	}

	// 4️ Parse optional reason from request body
	var input struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&input)

	// 5️ Start transaction
	tx, err := h.Query.DB.Beginx()
	if err != nil {
		utils.RespondWithError(c, 500, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	// 6️ Fetch leave details
	var leave models.Leave
	err = tx.Get(&leave, `
		SELECT id, employee_id, leave_type_id, start_date, end_date, days, status, created_at
		FROM Tbl_Leave 
		WHERE id=$1 
		FOR UPDATE
	`, leaveID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(c, 404, "leave request not found")
			return
		}
		utils.RespondWithError(c, 500, "failed to fetch leave: "+err.Error())
		return
	}

	// 7️ Prevent withdrawing own leave
	if leave.EmployeeID == currentUserID {
		utils.RespondWithError(c, 403, "you cannot withdraw your own leave. Please contact your manager or admin")
		return
	}

	// 8️ MANAGER validation
	if role == "MANAGER" {
		// Verify reporting relationship
		var managerID uuid.UUID
		err := tx.Get(&managerID, "SELECT manager_id FROM Tbl_Employee WHERE id=$1", leave.EmployeeID)
		if err != nil {
			utils.RespondWithError(c, 500, "failed to verify manager relationship")
			return
		}
		if managerID != currentUserID {
			utils.RespondWithError(c, 403, "managers can only withdraw leaves of their team members")
			return
		}

		// Manager can only act on APPROVED leaves
		if leave.Status != "APPROVED" {
			utils.RespondWithError(c, 400, fmt.Sprintf("cannot withdraw leave with status: %s. Only approved leaves can be withdrawn", leave.Status))
			return
		}
	}

	// 9️ ADMIN/SUPERADMIN validation
	if role == "ADMIN" || role == "SUPERADMIN" {
		// Admin can act on APPROVED or WITHDRAWAL_PENDING leaves
		if leave.Status != "APPROVED" && leave.Status != "WITHDRAWAL_PENDING" {
			utils.RespondWithError(c, 400, fmt.Sprintf("cannot withdraw leave with status: %s", leave.Status))
			return
		}
	}

	// ========================================
	// MANAGER WITHDRAWAL REQUEST (First Level)
	// ========================================
	if role == "MANAGER" {
		withdrawalReason := input.Reason
		if withdrawalReason == "" {
			withdrawalReason = "Withdrawal requested by Manager"
		}

		// Update status to WITHDRAWAL_PENDING
		_, err = tx.Exec(`
			UPDATE Tbl_Leave 
			SET status='WITHDRAWAL_PENDING', reason=$1, approved_by=$2, updated_at=NOW() 
			WHERE id=$3
		`, withdrawalReason, currentUserID, leaveID)
		if err != nil {
			utils.RespondWithError(c, 500, "failed to request withdrawal: "+err.Error())
			return
		}

		// Fetch employee and manager details
		var empDetails struct {
			Email    string `db:"email"`
			FullName string `db:"full_name"`
		}
		h.Query.DB.Get(&empDetails, "SELECT email, full_name FROM Tbl_Employee WHERE id=$1", leave.EmployeeID)

		var managerName string
		h.Query.DB.Get(&managerName, "SELECT full_name FROM Tbl_Employee WHERE id=$1", currentUserID)

		var leaveTypeName string
		h.Query.DB.Get(&leaveTypeName, "SELECT name FROM Tbl_Leave_type WHERE id=$1", leave.LeaveTypeID)

		tx.Commit()

		// Notify admins about pending withdrawal
		go func() {
			var adminEmails []string
			h.Query.DB.Select(&adminEmails, `
				SELECT e.email 
				FROM Tbl_Employee e
				JOIN Tbl_Role r ON e.role_id = r.id
				WHERE r.type IN ('ADMIN', 'SUPERADMIN') AND e.status = 'active'
			`)

			if len(adminEmails) > 0 {
				utils.SendLeaveWithdrawalPendingEmail(
					adminEmails,
					empDetails.FullName,
					leaveTypeName,
					leave.StartDate.Format("2006-01-02"),
					leave.EndDate.Format("2006-01-02"),
					leave.Days,
					managerName,
					withdrawalReason,
				)
			}
		}()

		c.JSON(200, gin.H{
			"message":           "withdrawal request submitted. Pending final approval from ADMIN/SUPERADMIN",
			"status":            "WITHDRAWAL_PENDING",
			"leave_id":          leaveID,
			"withdrawal_by":     currentUserID,
			"withdrawal_reason": withdrawalReason,
		})
		return
	}

	// ========================================
	// ADMIN/SUPERADMIN FINAL WITHDRAWAL (Second Level)
	// ========================================
	if role == "ADMIN" || role == "SUPERADMIN" {
		withdrawalReason := input.Reason
		if withdrawalReason == "" {
			withdrawalReason = fmt.Sprintf("Withdrawn by %s", role)
		}

		// Update status to WITHDRAWN
		_, err = tx.Exec(`
			UPDATE Tbl_Leave 
			SET status='WITHDRAWN', reason=$1, approved_by=$2, updated_at=NOW() 
			WHERE id=$3
		`, withdrawalReason, currentUserID, leaveID)
		if err != nil {
			utils.RespondWithError(c, 500, "failed to finalize withdrawal: "+err.Error())
			return
		}

		// Fetch leave type to check if it's an early leave
		leaveTypeID, err := h.Query.GetLeaveTypeByLeaveID(leaveID)
		if err != nil {
			utils.RespondWithError(c, 500, "failed to fetch leave type: "+err.Error())
			return
		}

		leaveType, err := h.Query.GetLeaveTypeByIdTx(tx, leaveTypeID)
		if err != nil {
			utils.RespondWithError(c, 500, "failed to fetch leave type details: "+err.Error())
			return
		}

		// Restore leave balance (reverse the deduction) - SKIP for is_early leave types
		isEarlyLeave := leaveType.IsEarly != nil && *leaveType.IsEarly
		if !isEarlyLeave {
			_, err = tx.Exec(`
				UPDATE Tbl_Leave_balance 
				SET used = used - $1, closing = closing + $1, updated_at = NOW()
				WHERE employee_id=$2 AND leave_type_id=$3 AND year = EXTRACT(YEAR FROM CURRENT_DATE)
			`, leave.Days, leave.EmployeeID, leave.LeaveTypeID)
			if err != nil {
				utils.RespondWithError(c, 500, "failed to restore leave balance: "+err.Error())
				return
			}
		}

		// Fetch data BEFORE committing transaction
		empDetails, err := h.Query.GetEmployeeDetailsForNotification(leave.EmployeeID)
		if err != nil {
			fmt.Printf("Failed to fetch employee details: %v\n", err)
		}

		var leaveTypeName string
		err = h.Query.DB.Get(&leaveTypeName, "SELECT name FROM Tbl_Leave_type WHERE id=$1", leave.LeaveTypeID)
		if err != nil {
			fmt.Printf("Failed to fetch leave type: %v\n", err)
		}

		var withdrawnByName string
		err = h.Query.DB.Get(&withdrawnByName, "SELECT full_name FROM Tbl_Employee WHERE id=$1", currentUserID)
		if err != nil {
			fmt.Printf("Failed to fetch withdrawn by name: %v\n", err)
		}
		admins, err := h.Query.GetAdminAndEmployeeEmail(leave.EmployeeID)
		if err != nil {
			fmt.Printf("Failed to fetch admin and employee emails: %v\n", err)
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			utils.RespondWithError(c, 500, "failed to commit transaction")
			return
		}

		// Send final withdrawal notification (async)
		if empDetails.Email != "" && leaveTypeName != "" && withdrawnByName != "" {
			go func(email, name, leaveType, startDate, endDate string, days float64, withdrawnBy, withdrawnRole, reason string) {
				fmt.Printf(" Sending withdrawal email to %s...\n", email)
				err := utils.SendLeaveWithdrawalEmail(
					admins,
					email,
					name,
					leaveType,
					startDate,
					endDate,
					days,
					withdrawnBy,
					withdrawnRole,
					reason,
				)
				if err != nil {
					fmt.Printf(" Failed to send withdrawal email: %v\n", err)
				} else {
					fmt.Printf(" Withdrawal email sent successfully to %s\n", email)
				}

				// Send HR-specific email
				var hrEmails []string
				h.Query.DB.Select(&hrEmails, `
					SELECT e.email 
					FROM Tbl_Employee e
					JOIN Tbl_Role r ON e.role_id = r.id
					WHERE r.type = 'HR' AND e.status = 'active'
				`)
				if len(hrEmails) > 0 {
					utils.SendLeaveWithdrawalEmailToHR(
						hrEmails,
						name,
						email,
						leaveType,
						startDate,
						endDate,
						days,
						withdrawnBy,
						withdrawnRole,
						reason,
					)
				}
			}(empDetails.Email, empDetails.FullName, leaveTypeName,
				leave.StartDate.Format("2006-01-02"),
				leave.EndDate.Format("2006-01-02"),
				leave.Days, withdrawnByName, role, input.Reason)
		}

		c.JSON(200, gin.H{
			"message":           "leave withdrawn successfully and balance restored",
			"status":            "WITHDRAWN",
			"leave_id":          leaveID,
			"days_restored":     leave.Days,
			"withdrawal_by":     currentUserID,
			"withdrawal_role":   role,
			"withdrawal_reason": withdrawalReason,
		})
		return
	}

	// Should not reach here
	utils.RespondWithError(c, 500, "unexpected error in leave withdrawal process")
}

// GetManagerLeaveHistory - GET /api/leaves/manager/history
// Manager gets leave history of their team members
func (h *HandlerFunc) GetManagerLeaveHistory(c *gin.Context) {
	// 1️ Get current user info with validation
	role := c.GetString("role")
	if role == "" {
		utils.RespondWithError(c, http.StatusUnauthorized, "Role not found in context")
		return
	}

	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		utils.RespondWithError(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}

	currentUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid user ID format: "+err.Error())
		return
	}

	// 2️ Permission check - Only MANAGER can use this endpoint
	if role != "MANAGER" {
		utils.RespondWithError(c, http.StatusForbidden, "Only managers can access team leave history")
		return
	}

	// 3️ Query to get team members' leave history
	query := `
		SELECT 
			l.id,
			e.full_name AS employee,
			lt.name AS leave_type,
			lt.is_paid AS is_paid,
			COALESCE(h.type, 'FULL') AS leave_timing_type,
			COALESCE(h.timing, 'Full Day') AS leave_timing,
			l.start_date,
			l.end_date,
			l.days,
			COALESCE(l.reason, '') AS reason,
			l.status,
			l.created_at AS applied_at,
			approver.full_name AS approval_name
		FROM Tbl_Leave l
		INNER JOIN Tbl_Employee e ON l.employee_id = e.id
		INNER JOIN Tbl_Leave_Type lt ON lt.id = l.leave_type_id
		LEFT JOIN Tbl_Half h ON l.half_id = h.id
		LEFT JOIN Tbl_Employee approver ON l.approved_by = approver.id
		WHERE e.manager_id = $1
		ORDER BY l.created_at DESC
	`

	// 4️ Execute query with proper error handling
	var result []models.LeaveResponse
	err = h.Query.DB.Select(&result, query, currentUserID)
	if err != nil {
		// Log the error for debugging
		fmt.Printf(" GetManagerLeaveHistory DB Error: %v\n", err)
		fmt.Printf("Manager ID: %s\n", currentUserID)

		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch team leave history: "+err.Error())
		return
	}

	// 5️ Handle empty result
	if result == nil {
		result = []models.LeaveResponse{}
	}

	// 6️ Response with metadata
	c.JSON(http.StatusOK, gin.H{
		"message":      "Team leave history fetched successfully",
		"manager_id":   currentUserID,
		"total_leaves": len(result),
		"leaves":       result,
	})
}

// GetLeaveByID - GET /api/leaves/:id
// Get specific leave details by ID
func (h *HandlerFunc) GetLeaveByID(c *gin.Context) {
	// Get user info from middleware
	userIDRaw, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDRaw.(string))
	role := c.GetString("role")

	// Parse leave ID from URL
	leaveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, 400, "Invalid leave ID")
		return
	}

	// Query to get leave details with timing information
	query := `
		SELECT 
			l.id,
			e.full_name AS employee,
			lt.name AS leave_type,
			lt.is_paid AS is_paid,
			COALESCE(h.type, 'FULL') AS leave_timing_type,
			COALESCE(h.timing, 'Full Day') AS leave_timing,
			l.start_date,
			l.end_date,
			l.days,
			COALESCE(l.reason, '') AS reason,
			l.status,
			l.created_at AS applied_at,
			approver.full_name AS approval_name
		FROM Tbl_Leave l
		INNER JOIN Tbl_Employee e ON l.employee_id = e.id
		INNER JOIN Tbl_Leave_Type lt ON lt.id = l.leave_type_id
		LEFT JOIN Tbl_Half h ON l.half_id = h.id
		LEFT JOIN Tbl_Employee approver ON l.approved_by = approver.id
		WHERE l.id = $1
	`

	var result models.LeaveResponse
	err = h.Query.DB.Get(&result, query, leaveID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(c, 404, "Leave not found")
			return
		}
		utils.RespondWithError(c, 500, "Failed to fetch leave details: "+err.Error())
		return
	}

	// Permission check - employees can only see their own leaves
	// Get the employee ID for this leave
	var leaveEmployeeID uuid.UUID
	err = h.Query.DB.Get(&leaveEmployeeID, "SELECT employee_id FROM Tbl_Leave WHERE id = $1", leaveID)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to verify leave ownership")
		return
	}

	// Role-based access control
	switch role {
	case "EMPLOYEE":
		if leaveEmployeeID != userID {
			utils.RespondWithError(c, 403, "You can only view your own leave applications")
			return
		}
	case "MANAGER":
		// Manager can see their own leaves + their team members' leaves
		var managerID uuid.UUID
		err = h.Query.DB.Get(&managerID, "SELECT COALESCE(manager_id, '00000000-0000-0000-0000-000000000000') FROM Tbl_Employee WHERE id = $1", leaveEmployeeID)
		if err != nil {
			utils.RespondWithError(c, 500, "Failed to verify manager relationship")
			return
		}
		if leaveEmployeeID != userID && managerID != userID {
			utils.RespondWithError(c, 403, "You can only view leaves of your team members or your own leaves")
			return
		}
	case "HR", "ADMIN", "SUPERADMIN":
		// HR, Admin and SuperAdmin can see all leaves - no additional check needed
	default:
		utils.RespondWithError(c, 403, "Invalid role")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Leave details fetched successfully",
		"data":    result,
	})
}

// GET_LEAVE_VARIENT
// GetLeaveTiming - GET /api/leave-timing
// SuperAdmin & Admin can view all leave timing variants
func (h *HandlerFunc) GetLeaveTiming(c *gin.Context) {

	// 2 Fetch from DB
	data, err := h.Query.GetLeaveTiming()
	if err != nil {
		fmt.Printf("GetLeaveTiming DB Error: %v\n", err)
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch leave timing")
		return
	}

	//2 Empty slice safety
	if data == nil {
		data = []models.LeaveTimingResponse{}
	}

	// 3 Response
	c.JSON(http.StatusOK, gin.H{
		"message": "Leave timing fetched successfully",
		"total":   len(data),
		"data":    data,
	})
}

// GetLeaveTimingByID - GET /api/leave-timing/:id
func (h *HandlerFunc) GetLeaveTimingByID(c *gin.Context) {

	// 1️ Role validation
	role := c.GetString("role")
	if role != constant.ROLE_SUPER_ADMIN && role != constant.ROLE_ADMIN {
		utils.RespondWithError(c, http.StatusForbidden, "Access denied")
		return
	}

	// 2️ Bind URI
	var req models.GetLeaveTimingByIDReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 3️ Validate
	if err := models.Validate.Struct(req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 4️ Fetch data
	data, err := h.Query.GetLeaveTimingByID(req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(c, http.StatusNotFound, "Leave timing not found")
			return
		}
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch leave timing")
		return
	}

	// 5️ Response
	c.JSON(http.StatusOK, gin.H{
		"message": "Leave timing fetched successfully",
		"data":    data,
	})
}

func (h *HandlerFunc) UpdateLeaveTiming(c *gin.Context) {

	// 1️ Role validation
	role := c.GetString("role")
	if role != constant.ROLE_SUPER_ADMIN && role != constant.ROLE_ADMIN {
		utils.RespondWithError(c, http.StatusForbidden, "Access denied")
		return
	}

	// 2️ Bind URI + Body
	var req models.UpdateLeaveTimingReq

	if err := c.ShouldBindUri(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 3️ Validate
	if err := models.Validate.Struct(req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		// 4️ Update DB

		err := h.Query.UpdateLeaveTiming(tx, req.ID, req.Timing)
		if err != nil {
			if err == sql.ErrNoRows {
				return utils.CustomErr(c, http.StatusNotFound, "Leave timing not found")

			}
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to update leave timing")
		}
		return nil
	})

	// If transaction returned an error, stop (CustomErr already responded)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update settings: "+err.Error())
		return
	}

	// 5️ Response
	c.JSON(http.StatusOK, gin.H{
		"message": "Leave timing updated successfully",
	})
}

func (h *HandlerFunc) EditMyLeave(c *gin.Context) {
	// 1. Get Leave ID from URL
	leaveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, 400, "Invalid Leave ID")
		return
	}

	// 2. Get Current Employee ID from Context
	empIDRaw, _ := c.Get("user_id")
	empID, _ := uuid.Parse(empIDRaw.(string))

	// Validate Employee Status
	empStatus, err := h.Query.GetEmployeeStatus(empID)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to verify employee status")
		return
	}
	if empStatus == "deactive" {
		utils.RespondWithError(c, 403, "Your account is deactivated. You cannot edit leave")
		return
	}

	// 3. Bind Input (JSON)
	var input models.LeaveUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, 400, "Invalid input data: "+err.Error())
		return
	}

	var leaveTiming time.Time = time.Time{} // Zero value indicates not provided

	// If LeaveTiming string is provided, validate it
	if input.LeaveTiming != nil {
		leaveTiming, err = service.ValidateLeaveTiming(*input.LeaveTiming)
		if err != nil {
			utils.RespondWithError(c, 400, err.Error())
			return
		}
	}

	// Validate timing ID if provided (must be 1, 2, or 3)
	if input.LeaveTimingID != nil && (*input.LeaveTimingID < 1 || *input.LeaveTimingID > 3) {
		utils.RespondWithError(c, 400, "Invalid leave timing ID. Must be 1 (First Half), 2 (Second Half), or 3 (Full Day)")
		return
	}

	// Validate Reason
	input.Reason = strings.TrimSpace(input.Reason)
	if len(input.Reason) < 10 {
		utils.RespondWithError(c, 400, "Leave reason must be at least 10 characters long")
		return
	}
	if len(input.Reason) > 500 {
		utils.RespondWithError(c, 400, "Leave reason is too long. Maximum 500 characters allowed")
		return
	}

	// Validate Dates
	if input.EndDate.Before(input.StartDate) {
		utils.RespondWithError(c, 400, "End date cannot be earlier than start date")
		return
	}

	// 4. Execute Transaction
	err = common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		// Fetch Leave Type to check IsEarly
		leaveType, err := h.Query.GetLeaveTypeByIdTx(tx, input.LeaveTypeID)
		if err == sql.ErrNoRows {
			return utils.CustomErr(c, 400, "Invalid leave type")
		}
		if err != nil {
			return utils.CustomErr(c, 500, "Failed to fetch leave type: "+err.Error())
		}

		// Determine timing ID based on IsEarly flag
		timingID := 3 // Default to Full Day
		if input.LeaveTimingID != nil {
			timingID = *input.LeaveTimingID
		}

		// For IsEarly leave types, timing is not applicable
		if leaveType.IsEarly != nil && *leaveType.IsEarly {
			timingID = 3 // Force full day for early leave types
		}

		// Calculate new working days with timing consideration
		newDays, err := service.CalculateWorkingDaysWithTiming(h.Query, tx, input.StartDate, input.EndDate, timingID, leaveTiming)
		if err != nil {
			return utils.CustomErr(c, 400, err.Error())
		}
		if newDays <= 0 {
			return utils.CustomErr(c, 400, "Calculated leave days must be greater than zero. Please check the dates and timing")
		}

		// Leave Balance - Skip balance check for IsEarly leave types
		if leaveType.IsEarly == nil || *leaveType.IsEarly == false {
			balance, err := h.Query.GetLeaveBalance(tx, empID, input.LeaveTypeID)

			if err == sql.ErrNoRows {
				// During edit, if balance doesn't exist, it means the leave type is invalid or not assigned
				return utils.CustomErr(c, 400, "Leave balance not found for this leave type. Please contact HR")
			} else if err != nil {
				return utils.CustomErr(c, 500, "Failed to fetch leave balance: "+err.Error())
			}

			// IMPORTANT: For PENDING leaves, balance is NOT yet deducted
			// Balance is only deducted when leave status becomes APPROVED
			// So we need to check if the NEW total days fit within available balance

			// Check if sufficient balance exists for the edited leave
			if balance < newDays {
				return utils.CustomErr(c, 400, fmt.Sprintf("Insufficient leave balance. You have %.2f days available but the edited leave requires %.2f days", balance, newDays))
			}
		}

		// Check for overlapping leaves (excluding current leave being edited)
		overlaps, err := h.Query.GetOverlappingLeaves(tx, empID, input.StartDate, input.EndDate)
		if err != nil {
			return utils.CustomErr(c, 500, "Failed to check overlapping leave")
		}
		// Filter out the current leave being edited
		for _, ov := range overlaps {
			if ov.ID != leaveID {
				return utils.CustomErr(c, 400, fmt.Sprintf(
					"Overlapping leave exists: %s from %s to %s (Status: %s). Please cancel or modify the existing leave first",
					ov.LeaveType,
					ov.StartDate.Format("2006-01-02"),
					ov.EndDate.Format("2006-01-02"),
					ov.Status,
				))
			}
		}

		// Update the leave
		err = h.Query.UpdatePendingLeave(tx, leaveID, empID, input, newDays)
		if err != nil {
			return utils.CustomErr(c, 500, "Failed to update leave: "+err.Error())
		}

		// Log Entry
		data := &models.Common{
			Component:  constant.ComponentLeave,
			Action:     constant.ActionUpdate,
			FromUserID: empID,
		}
		if err := h.Query.AddLog(data, tx); err != nil {
			return utils.CustomErr(c, 500, "Failed to create leave log: "+err.Error())
		}

		return nil
	})

	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update settings: "+err.Error())
		return
	}

	c.JSON(200, gin.H{
		"message":  "Leave updated successfully",
		"leave_id": leaveID,
	})
}
