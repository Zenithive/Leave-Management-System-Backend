package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/helper"
)

func (h *HandlerFunc) ApplyLeave(c *gin.Context) {
	empID, ok := helper.ExtractEmployeeID(c)
	if !ok {
		pkg.RespondWithError(c, http.StatusUnauthorized, "missing EpID")
		return
	}
	role := c.GetString("role")
	var input models.LeaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		pkg.RespondWithError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	input.EmployeeID = empID
	if err := h.LeaveFlowService.Create(c, &input, role); err != nil {
		pkg.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "leave Applying Successfully",
	})
}

func (h *HandlerFunc) LeaveAction(c *gin.Context) {
	empID, ok := helper.ExtractEmployeeID(c)
	if !ok {
		pkg.RespondWithError(c, http.StatusUnauthorized, "missing EpID")
		return
	}
	var req models.ActionLeaveReq
	role := c.GetString("role")

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.RespondWithError(c, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}
	leaveID := c.Param("id")

	if err := h.LeaveFlowService.ActionLeave(c, req, leaveID, empID, role); err != nil {
		pkg.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "leave approver or  reject Successfully",
	})
}

func (h *HandlerFunc) GetLeaves(c *gin.Context) {
	empID, ok := helper.ExtractEmployeeID(c)
	if !ok {
		pkg.RespondWithError(c, http.StatusUnauthorized, "missing EpID")
		return
	}

	role := c.GetString("role")

	month, year, err := common.GetMonthYear(c)
	if err != nil {
		pkg.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.LeaveFlowService.GetLeaves(c, empID, role, month, year)
	if err != nil {
		pkg.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
func (h *HandlerFunc) GetAllMyLeave(c *gin.Context) {
	empID, ok := helper.ExtractEmployeeID(c)
	if !ok {
		pkg.RespondWithError(c, http.StatusUnauthorized, "missing EpID")
		return
	}

	month, year, err := common.GetMonthYear(c)
	if err != nil {
		pkg.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.LeaveFlowService.GetMyLeave(empID, month, year)
	if err != nil {
		pkg.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *HandlerFunc) CancelLeave(c *gin.Context) {
	// Parse leave ID from URL
	leaveID := c.Param("id")
	if leaveID == "" {
		pkg.RespondWithError(c, http.StatusBadRequest, "leave_id is required")
		return
	}
	h.LeaveFlowService.CancleLeave(c, leaveID)

	c.JSON(200, gin.H{
		"message":  "Leave cancelled successfully",
		"leave_id": leaveID,
	})
}

func (h *HandlerFunc) EditLeave(c *gin.Context) {
	empID, ok := helper.ExtractEmployeeID(c)
	if !ok {
		pkg.RespondWithError(c, http.StatusUnauthorized, "missing EpID")
		return
	}
	leaveID := c.Param("id")
	if leaveID == "" {
		pkg.RespondWithError(c, http.StatusBadRequest, "leave_id is required")
		return
	}
	// 3. Bind Input (JSON)
	var input models.LeaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		pkg.RespondWithError(c, http.StatusBadRequest, "Invalid input data: "+err.Error())
		return
	}
	input.EmployeeID = empID
	if err := h.LeaveFlowService.UpdateLeave(c, empID, leaveID, &input); err != nil {
		pkg.Error(c, err)
		return
	}
	c.JSON(200, gin.H{
		"message": "Leave update successfully",
	})
}
