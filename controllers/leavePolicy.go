package controllers

// LeaveType controller — HTTP adapter layer.
//
// Responsibilities of this file:
//   - Parse and validate HTTP request data (path params, JSON body, auth claims)
//   - Enforce role-based access control
//   - Open a transaction, call LeaveTypeService, write an audit log, commit
//   - Translate service errors into appropriate HTTP status codes
//   - Serialize the response
//
// Business logic lives in service/leave_type_service.go.
// Database queries live in repositories/leaveType.go and repositories/leaveBalance.go.

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/helper"
)

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/leaves/admin-add/policy
// ─────────────────────────────────────────────────────────────────────────────

// AdminAddLeavePolicy creates a new leave type (policy).
// Only SUPERADMIN may call this endpoint.
func (h *HandlerFunc) LeavePolicy(c *gin.Context) {

	if role := c.GetString("role"); role != constant.ROLE_SUPER_ADMIN {
		utils.RespondWithError(c, http.StatusUnauthorized, "not permitted to create leave policies")
		return
	}

	// ── Input ─────────────────────────────────────────────────────────────────
	var input models.LeaveTypeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	svcResult, err := h.LeavePolicyService.Create(c, &input)
	if err != nil {
		utils.Error(c, err)
		return
	}
	var result models.LeaveType

	result = *svcResult.LeaveType

	c.JSON(http.StatusOK, result)
}

// GetAllLeavePolicies returns all leave type definitions.
func (h *HandlerFunc) GetAllLeavePolicies(c *gin.Context) {
	res, err := h.LeavePolicyService.Get(c)
	if err != nil {
		utils.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /api/leaves/admin-update/policy/:id
// ─────────────────────────────────────────────────────────────────────────────

// UpdateLeavePolicy updates an existing leave type and recalculates employee balances.
// Allowed roles: SUPERADMIN, ADMIN, HR.
func (h *HandlerFunc) UpdateLeavePolicy(c *gin.Context) {
	// ── Auth ─────────────────────────────────────────────────────────────────
	employeeID, ok := helper.ExtractEmployeeID(c)
	if !ok {
		return
	}

	if role := c.GetString("role"); role != constant.ROLE_SUPER_ADMIN &&
		role != constant.ROLE_ADMIN && role != constant.ROLE_HR {
		utils.RespondWithError(c, http.StatusUnauthorized, "not permitted to update leave policies")
		return
	}

	// ── Path param ────────────────────────────────────────────────────────────
	leaveTypeID, ok := parseLeaveTypeID(c)
	if !ok {
		return
	}

	// ── Input ─────────────────────────────────────────────────────────────────
	var input models.LeaveTypeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	// ── Transaction: service call + audit log ─────────────────────────────────
	err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if _, err := h.LeaveTypeSvc.UpdateLeaveType(tx, leaveTypeID, input); err != nil {
			// Map "not found" to 404, everything else to 400/500.
			if err.Error() == "leave type not found" {
				return utils.CustomErr(c, http.StatusNotFound, err.Error())
			}
			if err.Error() == "default entitlement cannot be negative" {
				return utils.CustomErr(c, http.StatusBadRequest, err.Error())
			}
			return utils.CustomErr(c, http.StatusInternalServerError, err.Error())
		}
		return h.writeLeaveTypeLog(c, tx, constant.ActionUpdate, employeeID)
	})
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to update leave policy: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "leave policy updated successfully",
		"id":      leaveTypeID,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// DELETE /api/leaves/admin-delete/policy/:id
// ─────────────────────────────────────────────────────────────────────────────

// DeleteLeavePolicy deletes a leave type.
// Allowed roles: SUPERADMIN, ADMIN, HR.
func (h *HandlerFunc) DeleteLeavePolicy(c *gin.Context) {
	// ── Auth ─────────────────────────────────────────────────────────────────
	employeeID, ok := helper.ExtractEmployeeID(c)
	if !ok {
		return
	}

	if role := c.GetString("role"); role != constant.ROLE_SUPER_ADMIN &&
		role != constant.ROLE_ADMIN && role != constant.ROLE_HR {
		utils.RespondWithError(c, http.StatusUnauthorized, "not permitted to delete leave policies")
		return
	}

	// ── Path param ────────────────────────────────────────────────────────────
	leaveTypeID, ok := parseLeaveTypeID(c)
	if !ok {
		return
	}

	// ── Transaction: service call + audit log ─────────────────────────────────
	err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.LeaveTypeSvc.DeleteLeaveType(tx, leaveTypeID); err != nil {
			if err.Error() == "leave type not found" {
				return utils.CustomErr(c, http.StatusNotFound, err.Error())
			}
			if err.Error() == "cannot delete leave type: it is being used in existing leave applications" {
				return utils.CustomErr(c, http.StatusConflict, err.Error())
			}
			return utils.CustomErr(c, http.StatusInternalServerError, err.Error())
		}
		return h.writeLeaveTypeLog(c, tx, constant.ActionDelete, employeeID)
	})
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to delete leave policy: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "leave policy deleted successfully",
		"id":      leaveTypeID,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/leaves/policies
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// Private helpers
// ─────────────────────────────────────────────────────────────────────────────

// parseLeaveTypeID parses and validates the ":id" path parameter.
func parseLeaveTypeID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid leave type ID")
		return 0, false
	}
	return id, true
}

// writeLeaveTypeLog writes an audit log entry for a leave type operation
// inside an already-open transaction.
func (h *HandlerFunc) writeLeaveTypeLog(c *gin.Context, tx *sqlx.Tx, action string, fromUserID uuid.UUID) error {
	data := &models.Common{
		Component:  constant.ComponentLeaveType,
		Action:     action,
		FromUserID: fromUserID,
	}
	if err := h.Query.AddLog(data, tx); err != nil {
		return utils.CustomErr(c, http.StatusInternalServerError, "failed to write audit log: "+err.Error())
	}
	return nil
}
