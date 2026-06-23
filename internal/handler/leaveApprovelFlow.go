package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
)

func (h *HandlerFunc) CreateApprovelFlow(c *gin.Context) {
	var req models.LeaveApprovalFlowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.LeaveApproverFlowService.CreateLeaveApproverFlow(c, &req); err != nil {
		utils.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "approval flow fetched successfully",
	})
}
func (h *HandlerFunc) GetAllApprovelFlow(c *gin.Context) {

	res, err := h.LeaveApproverFlowService.GetAllLeaveApprovalFlows(c)
	if err != nil {
		utils.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res,
	})
}
func (h *HandlerFunc) UpdateLeaveApprovelFlow(c *gin.Context) {
	var req models.LeaveApprovalFlowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	id := c.Param("id")

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.LeaveApproverFlowService.UpdateLeaveApprovelFlow(c, id, &req); err != nil {
		utils.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "update Leave flow Succesfully",
	})
}
func (h *HandlerFunc) DeleteLeaveApprovelFlow(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "id is required")
		return
	}
	if err := h.LeaveApproverFlowService.DeleteLeaveApprovelFlow(c, id); err != nil {
		utils.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Delete Leavde Flow Succesfully",
	})
}
