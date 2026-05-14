package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
)

// GET /api/leaves/timming
func (h *HandlerFunc) GetLeaveTiming(c *gin.Context) {
	data, err := h.LeaveTimingSvc.GetAll()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Leave timing fetched successfully",
		"total":   len(data),
		"data":    data,
	})
}

// GET /api/leave-timing/:id
func (h *HandlerFunc) GetLeaveTimingByID(c *gin.Context) {
	var req models.GetLeaveTimingByIDReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := models.Validate.Struct(req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	data, err := h.LeaveTimingSvc.GetByID(req.ID)
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Leave timing fetched successfully",
		"data":    data,
	})
}

// PUT /api/leaves/timming
func (h *HandlerFunc) UpdateLeaveTiming(c *gin.Context) {
	var req models.UpdateLeaveTimingReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := models.Validate.Struct(req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		return h.LeaveTimingSvc.Update(tx, req.ID, req.Timing)
	}); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Leave timing updated successfully"})
}
