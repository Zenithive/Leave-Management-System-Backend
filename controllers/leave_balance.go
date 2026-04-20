package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
)

// GetLeaveBalances - GET /api/employees/:id/leave-balances
// GetLeaveBalances - GET /api/employees/:id/leave-balances
func (s *HandlerFunc) GetLeaveBalances(c *gin.Context) {
	// 1. Parse employee ID from path
	employeeIDParam := c.Param("id")
	employeeID, err := uuid.Parse(employeeIDParam)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid employee ID")
		return
	}

	// 2. Role check (employees can only view their own balances)
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)
	userIDRaw, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDRaw.(string))

	if role == "EMPLOYEE" && userID != employeeID {
		utils.RespondWithError(c, http.StatusForbidden, "Employees can only view their own balances")
		return
	}

	// 3. Get current year for filtering
	currentYear := time.Now().Year()

	// 4. Fetch all leave types with their default entitlements (repository layer)
	leaveTypes, err := s.Query.GetAllLeaveTypesWithEntitlements()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError,
			"Failed to fetch leave types: "+err.Error())
		return
	}

	// 5. Fetch leave balances for current year (repository layer)
	balanceRecords, err := s.Query.GetLeaveBalancesByEmployeeAndYear(employeeID, currentYear)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError,
			"Failed to fetch leave balances: "+err.Error())
		return
	}

	// 6. Convert to service layer types
	serviceLeaveTypes := make([]service.LeaveTypeData, len(leaveTypes))
	for i, lt := range leaveTypes {
		serviceLeaveTypes[i] = service.LeaveTypeData{
			LeaveTypeID:        lt.LeaveTypeID,
			LeaveTypeName:      lt.LeaveTypeName,
			DefaultEntitlement: lt.DefaultEntitlement,
		}
	}

	serviceBalanceRecords := make([]service.LeaveBalanceData, len(balanceRecords))
	for i, br := range balanceRecords {
		serviceBalanceRecords[i] = service.LeaveBalanceData{
			LeaveTypeID: br.LeaveTypeID,
			Opening:     br.Opening,
			Accrued:     br.Accrued,
			Used:        br.Used,
			Adjusted:    br.Adjusted,
			Closing:     br.Closing,
		}
	}

	// 7. Calculate balances using service layer business logic (map-based)
	calculatedBalances := service.CalculateLeaveBalances(serviceLeaveTypes, serviceBalanceRecords)

	// 8. Convert back to response format

	balances := make([]models.Balance, len(calculatedBalances))
	for i, cb := range calculatedBalances {
		balances[i] = models.Balance{
			LeaveTypeID: cb.LeaveTypeID,
			LeaveType:   cb.LeaveType,
			Opening:     cb.Opening,
			Accrued:     cb.Accrued,
			Used:        cb.Used,
			Adjusted:    cb.Adjusted,
			Total:       cb.Total,
			Available:   cb.Available,
		}
	}
	fmt.Println(balances)

	// 8. Send response
	c.JSON(http.StatusOK, gin.H{
		"employee_id": employeeID,
		"year":        currentYear,
		"balances":    balances,
	})
}

// AdjustLeaveBalance - POST /api/leave-balances/:id/adjust
// AdjustLeaveBalance - POST /api/leave-balances/adjust
// AdjustLeaveBalance - POST /api/leave-balances/:id/adjust
func (s *HandlerFunc) AdjustLeaveBalance(c *gin.Context) {
	// 1️ Role check - Only ADMIN/HR allowed
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)
	if role != "ADMIN" && role != "SUPERADMIN" && role != "HR" {
		utils.RespondWithError(c, 403, "Not authorized to adjust leave balances")
		return
	}

	// 2️ Get employee ID from params
	employeeIDParam := c.Param("id")
	employeeID, err := uuid.Parse(employeeIDParam)
	if err != nil {
		utils.RespondWithError(c, 400, "Invalid employee ID")
		return
	}

	// 3️ Parse JSON input
	var input struct {
		LeaveTypeID int     `json:"leave_type_id" validate:"required"`
		Quantity    float64 `json:"quantity" validate:"required"` // +ve or -ve
		Reason      string  `json:"reason" validate:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, 400, "Invalid input: "+err.Error())
		return
	}

	currentYear := time.Now().Year()

	// 4️ Start transaction
	tx, err := s.Query.DB.Beginx()
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// 5️ Fetch or create leave balance (repository layer)
	balance, err := s.Query.GetLeaveBalanceForAdjustment(tx, employeeID, input.LeaveTypeID, currentYear)

	if err == sql.ErrNoRows {
		// 5A: Fetch default entitlement (repository layer)
		defaultEntitlement, err := s.Query.GetDefaultEntitlementByLeaveTypeID(tx, input.LeaveTypeID)
		if err != nil {
			utils.RespondWithError(c, 500, "Failed to fetch leave type: "+err.Error())
			return
		}

		// 5B: Create balance row (repository layer)
		balance, err = s.Query.CreateLeaveBalanceForAdjustment(tx, employeeID, input.LeaveTypeID, currentYear, defaultEntitlement)
		if err != nil {
			utils.RespondWithError(c, 500, "Failed to create leave balance: "+err.Error())
			return
		}

	} else if err != nil {
		utils.RespondWithError(c, 500, "Failed to fetch leave balance: "+err.Error())
		return
	}

	// 6️ Apply adjustment
	newAdjusted := balance.Adjusted + input.Quantity
	newClosing := balance.Opening + balance.Accrued - balance.Used + newAdjusted

	// Update leave balance (repository layer)
	err = s.Query.UpdateLeaveBalanceAdjustment(tx, balance.ID, newAdjusted, newClosing)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update leave balance: "+err.Error())
		return
	}

	// 7️ Insert into adjustment log (repository layer)
	err = s.Query.InsertLeaveAdjustment(tx, employeeID, input.LeaveTypeID, input.Quantity, input.Reason, c.GetString("user_id"), currentYear)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to record leave adjustment: "+err.Error())
		return
	}

	// 8️ Commit
	if err := tx.Commit(); err != nil {
		utils.RespondWithError(c, 500, "Transaction commit failed")
		return
	}

	c.JSON(200, gin.H{
		"message":      "Leave balance adjusted successfully",
		"new_adjusted": newAdjusted,
		"new_closing":  newClosing,
		"year":         currentYear,
	})
}
