package handler

import (
	"net/http"

	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *HandlerFunc) GetLeaveLog(c *gin.Context) {
	leaveID := c.Query("leave_id")
	if leaveID == "" {
		errors.RespondWithError(c, http.StatusBadRequest, "leave_id is required")
		return
	}

	res, err := h.LeaveFlowLogService.GetByLeaveID(c.Request.Context(), uuid.MustParse(leaveID))
	if err != nil {
		errors.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
