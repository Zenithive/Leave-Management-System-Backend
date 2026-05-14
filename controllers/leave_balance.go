package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/access_role"
)

// GetLeaveBalances - GET /api/leave-balances/employee/:id
func (h *HandlerFunc) GetLeaveBalances(c *gin.Context) {
	employeeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	callerID, _ := uuid.Parse(c.GetString("user_id"))
	callerRole := c.GetString("role")

	result, err := h.LeaveBalanceSvc.GetBalances(employeeID, callerID, callerRole)
	if err != nil {
		utils.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"employee_id": result.EmployeeID,
		"year":        result.Year,
		"balances":    result.Balances,
	})
}

// AdjustLeaveBalance - POST /api/leave-balances/:id/adjust
func (h *HandlerFunc) AdjustLeaveBalance(c *gin.Context) {
	callerRole := c.GetString("role")
	if err := access_role.Admin_SuperAdmin(callerRole, "not authorized to adjust leave balances"); err != nil {
		utils.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	employeeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	var input models.AdjustLeaveBalanceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	tx, err := h.Query.DB.Beginx()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	result, err := h.LeaveBalanceSvc.Adjust(tx, employeeID, input, c.GetString("user_id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "transaction commit failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "leave balance adjusted successfully",
		"new_adjusted": result.NewAdjusted,
		"new_closing":  result.NewClosing,
		"year":         result.Year,
	})
}
