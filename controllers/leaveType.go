package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
)

// POST /api/leaves/admin-add/policy
func (h *HandlerFunc) AdminAddLeavePolicy(c *gin.Context) {
	callerID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "access denied")
		return
	}

	var input models.LeaveTypeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	var leave models.LeaveType
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		var err error
		leave, err = h.LeaveTypeSvc.Create(tx, &input, callerID)
		return err
	}); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, leave)
}

// PUT /api/leaves/admin-update/policy/:id
func (h *HandlerFunc) UpdateLeavePolicy(c *gin.Context) {
	callerID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "access denied")
		return
	}

	leaveTypeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid leave type ID")
		return
	}

	var input models.LeaveTypeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		return h.LeaveTypeSvc.Update(tx, leaveTypeID, &input, callerID)
	}); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Leave policy updated successfully",
		"id":      leaveTypeID,
	})
}

// DELETE /api/leaves/admin-delete/policy/:id
func (h *HandlerFunc) DeleteLeavePolicy(c *gin.Context) {
	callerID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "access denied")
		return
	}

	leaveTypeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid leave type ID")
		return
	}

	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		return h.LeaveTypeSvc.Delete(tx, leaveTypeID, callerID)
	}); err != nil {
		// Distinguish 404 vs 409 vs 500
		msg := err.Error()
		if msg == "leave type not found" {
			utils.RespondWithError(c, http.StatusNotFound, msg)
		} else if msg == "cannot delete leave type: it is being used in existing leave applications" {
			utils.RespondWithError(c, http.StatusConflict, msg)
		} else {
			utils.RespondWithError(c, http.StatusInternalServerError, msg)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Leave policy deleted successfully",
		"id":      leaveTypeID,
	})
}

// GET /api/leaves/Get-All-Leave-Policy
func (h *HandlerFunc) GetAllLeavePolicies(c *gin.Context) {
	leaveTypes, err := h.LeaveTypeSvc.GetAll()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, leaveTypes)
}
