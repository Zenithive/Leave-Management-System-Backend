package controllers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

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
