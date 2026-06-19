package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/helper"
)

func (h *HandlerFunc) ApplyLeave(c *gin.Context) {
	empID, ok := helper.ExtractEmployeeID(c)
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "missing EpID")
		return
	}
	role := c.GetString("role")
	var input models.LeaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	input.EmployeeID = empID
	if err := h.LeaveFlowService.Create(c, &input, role); err != nil {
		utils.Error(c, err)
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
		utils.RespondWithError(c, http.StatusUnauthorized, "missing EpID")
		return
	}
	var req models.ActionLeaveReq
	role := c.GetString("role")

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, 400, "Invalid payload: "+err.Error())
		return
	}
	leaveID := c.Param("id")

	if err := h.LeaveFlowService.ActionLeave(c, req, leaveID, empID, role); err != nil {
		utils.Error(c, err)
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
		utils.RespondWithError(c, http.StatusUnauthorized, "missing EpID")
		return
	}

	role := c.GetString("role")

	month, err := strconv.Atoi(
		c.DefaultQuery("month", fmt.Sprintf("%d", int(time.Now().Month()))),
	)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid month")
		return
	}

	year, err := strconv.Atoi(
		c.DefaultQuery("year", fmt.Sprintf("%d", time.Now().Year())),
	)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid year")
		return
	}

	res, err := h.LeaveFlowService.GetLeaves(c, empID, role, month, year)
	if err != nil {
		utils.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
