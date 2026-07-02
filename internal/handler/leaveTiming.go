package handler

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/Zenithive/LeaveManagementSystem/internal/config/database"
	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/accessrole"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func (h *HandlerFunc) GetLeaveTiming(c *gin.Context) {

	// 2 Fetch from DB
	data, err := h.Query.GetLeaveTiming()
	if err != nil {
		slog.Error("GetLeaveTiming DB error", "err", err)
		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch leave timing")
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
	if role != accessrole.ROLE_SUPER_ADMIN && role != accessrole.ROLE_ADMIN {
		errors.RespondWithError(c, http.StatusForbidden, "Access denied")
		return
	}

	// 2️ Bind URI
	var req models.GetLeaveTimingByIDReq
	if err := c.ShouldBindUri(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 3️ Validate
	if err := models.Validate.Struct(req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 4️ Fetch data
	data, err := h.Query.GetLeaveTimingByID(req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			errors.RespondWithError(c, http.StatusNotFound, "Leave timing not found")
			return
		}
		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch leave timing")
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
	if role != accessrole.ROLE_SUPER_ADMIN && role != accessrole.ROLE_ADMIN {
		errors.RespondWithError(c, http.StatusForbidden, "Access denied")
		return
	}

	// 2️ Bind URI + Body
	var req models.UpdateLeaveTimingReq

	if err := c.ShouldBindUri(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 3️ Validate
	if err := models.Validate.Struct(req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		// 4️ Update DB

		err := h.Query.UpdateLeaveTiming(tx, req.ID, req.Timing)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.CustomErr(http.StatusNotFound, "Leave timing not found")

			}
			return errors.CustomErr(http.StatusInternalServerError, "Failed to update leave timing")
		}
		return nil
	})

	// If transaction returned an error, stop (CustomErr already responded)
	if err != nil {
		errors.RespondWithError(c, 500, "Failed to update settings: "+err.Error())
		return
	}

	// 5️ Response
	c.JSON(http.StatusOK, gin.H{
		"message": "Leave timing updated successfully",
	})
}
