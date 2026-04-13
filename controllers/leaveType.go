package controllers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

func (s *HandlerFunc) AdminAddLeavePolicy(c *gin.Context) {
	// Extract Employee Info
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "Employee ID missing")
		return
	}

	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee ID format")
		return
	}

	employeeID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee UUID")
		return
	}
	roleValue, exists := c.Get("role")
	if !exists {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to get role")
		return
	}
	userRole := roleValue.(string)
	if userRole != "SUPERADMIN" {
		utils.RespondWithError(c, http.StatusUnauthorized, "not permitted to assign manager")
		return
	}
	var input models.LeaveTypeInput

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}
	// Set defaults
	if input.IsPaid == nil {
		defaultPaid := false
		input.IsPaid = &defaultPaid
	}
	if input.DefaultEntitlement == nil {
		defaultEntitlement := 0
		input.DefaultEntitlement = &defaultEntitlement
	}
	if input.LeaveCount == nil {
		defaultCount := 2
		input.LeaveCount = &defaultCount
	}
	if input.IsEarly == nil {
		defaultEarly := false
		input.IsEarly = &defaultEarly
	}

	if *input.LeaveCount <= 0 {
		utils.RespondWithError(c, http.StatusBadRequest, "leave_count must be greater than 0")
		return
	}
	var leave models.LeaveType

	err = common.ExecuteTransaction(c, s.Query.DB, func(tx *sqlx.Tx) error {
		Leave, err := s.Query.AddLeaveType(tx, input)
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to insert leave type: "+err.Error())
		}
		Leave.Name = input.Name
		Leave.IsPaid = *input.IsPaid
		Leave.DefaultEntitlement = *input.DefaultEntitlement
		leave = Leave
		if input.IsEarly != nil {
			Leave.IsEarly = input.IsEarly
		}
		// Log Entry
		data := &models.Common{
			Component:  constant.ComponentLeaveType,
			Action:     constant.ActionCreate,
			FromUserID: employeeID,
		}
		if err := s.Query.AddLog(data, tx); err != nil {
			return utils.CustomErr(c, 500, "Failed to create leave log: "+err.Error())
		}
		return nil // IMPORTANT FIX
	})

	// If transaction returned an error, stop (CustomErr already responded)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update settings: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, leave)
}

// UpdateLeavePolicy - PUT /api/leaves/admin-update/policy/:id
// Admin, SuperAdmin, and HR can update leave policies
func (h *HandlerFunc) UpdateLeavePolicy(c *gin.Context) {
	// Extract Employee Info
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "Employee ID missing")
		return
	}

	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee ID format")
		return
	}

	employeeID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee UUID")
		return
	}

	// Role validation
	roleValue, exists := c.Get("role")
	if !exists {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get role")
		return
	}
	userRole := roleValue.(string)
	if userRole != "SUPERADMIN" && userRole != "ADMIN" && userRole != "HR" {
		utils.RespondWithError(c, http.StatusUnauthorized, "Not permitted to update leave policy")
		return
	}

	// Parse leave type ID from URL
	leaveTypeIDStr := c.Param("id")
	leaveTypeID, err := strconv.Atoi(leaveTypeIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid leave type ID")
		return
	}

	var input models.LeaveTypeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Set defaults if not provided
	if input.IsPaid == nil {
		defaultPaid := false
		input.IsPaid = &defaultPaid
	}
	if input.DefaultEntitlement == nil {
		defaultEntitlement := 0
		input.DefaultEntitlement = &defaultEntitlement
	}

	if *input.DefaultEntitlement < 0 {
		utils.RespondWithError(c, http.StatusBadRequest, "Default entitlement cannot be negative")
		return
	}

	err = common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		// Check if leave type exists and get old default entitlement
		oldLeaveType, err := h.Query.GetLeaveTypeByIdTx(tx, leaveTypeID)
		if err == sql.ErrNoRows {
			return utils.CustomErr(c, http.StatusNotFound, "Leave type not found")
		}
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to fetch leave type: "+err.Error())
		}

		// Get old and new default entitlement values
		oldDefaultEntitlement := oldLeaveType.DefaultEntitlement
		newDefaultEntitlement := *input.DefaultEntitlement

		// Update leave type
		err = h.Query.UpdateLeaveType(tx, leaveTypeID, input)
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to update leave type: "+err.Error())
		}

		// Update leave balances if default entitlement changed
		if oldDefaultEntitlement != newDefaultEntitlement && (oldLeaveType.IsEarly == nil || !*oldLeaveType.IsEarly) {
			currentYear := time.Now().Year()
			err = h.Query.UpdateLeaveBalancesForEntitlementChange(tx, leaveTypeID, oldDefaultEntitlement, newDefaultEntitlement, currentYear)
			if err != nil {
				return utils.CustomErr(c, http.StatusInternalServerError, "Failed to update leave balances: "+err.Error())
			}
		}
		// Log Entry
		data := &models.Common{
			Component:  constant.ComponentLeaveType,
			Action:     constant.ActionUpdate,
			FromUserID: employeeID,
		}
		if err := h.Query.AddLog(data, tx); err != nil {
			return utils.CustomErr(c, 500, "Failed to create leave log: "+err.Error())
		}
		return nil
	})

	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update leave policy: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Leave policy updated successfully",
		"id":      leaveTypeID,
	})
}

// DeleteLeavePolicy - DELETE /api/leaves/admin-delete/policy/:id
// Admin, SuperAdmin, and HR can delete leave policies
func (h *HandlerFunc) DeleteLeavePolicy(c *gin.Context) {
	// Extract Employee Info
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "Employee ID missing")
		return
	}

	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee ID format")
		return
	}

	employeeID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee UUID")
		return
	}

	// Role validation
	roleValue, exists := c.Get("role")
	if !exists {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get role")
		return
	}
	userRole := roleValue.(string)
	if userRole != "SUPERADMIN" && userRole != "ADMIN" && userRole != "HR" {
		utils.RespondWithError(c, http.StatusUnauthorized, "Not permitted to delete leave policy")
		return
	}

	// Parse leave type ID from URL
	leaveTypeIDStr := c.Param("id")
	leaveTypeID, err := strconv.Atoi(leaveTypeIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid leave type ID")
		return
	}

	err = common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		// Check if leave type exists
		_, err := h.Query.GetLeaveTypeByIdTx(tx, leaveTypeID)
		if err == sql.ErrNoRows {
			return utils.CustomErr(c, http.StatusNotFound, "Leave type not found")
		}
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to fetch leave type: "+err.Error())
		}

		// Delete leave type
		err = h.Query.DeleteLeaveType(tx, leaveTypeID)
		if err == sql.ErrNoRows {
			return utils.CustomErr(c, http.StatusConflict, "Cannot delete leave type: it is being used in existing leave applications")
		}
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to delete leave type: "+err.Error())
		}

		// Log Entry
		data := &models.Common{
			Component:  constant.ComponentLeaveType,
			Action:     constant.ActionDelete,
			FromUserID: employeeID,
		}
		if err := h.Query.AddLog(data, tx); err != nil {
			return utils.CustomErr(c, 500, "Failed to create leave log: "+err.Error())
		}
		return nil
	})

	if err != nil {
		utils.RespondWithError(c, 500, "Failed to delete leave policy: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Leave policy deleted successfully",
		"id":      leaveTypeID,
	})
}

func (s *HandlerFunc) GetAllLeavePolicies(c *gin.Context) {
	leaveType, err := s.Query.GetAllLeaveType()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch leave types: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, leaveType)
}
