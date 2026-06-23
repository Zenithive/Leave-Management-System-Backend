package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
)

func (h *HandlerFunc) GetLeaveLog(c *gin.Context) {
	leaveID := c.Query("leave_id")
	if leaveID == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "leave_id is required")
		return
	}

	res, err := h.LeaveFlowLogService.GetByLeaveID(c.Request.Context(), uuid.MustParse(leaveID))
	if err != nil {
		utils.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
