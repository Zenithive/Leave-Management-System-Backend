package handler

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

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/accessrole"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/gin-gonic/gin"
)

func (h *HandlerFunc) LeavePolicy(c *gin.Context) {

	if role := c.GetString("role"); role != accessrole.ROLE_SUPER_ADMIN {
		errors.RespondWithError(c, http.StatusUnauthorized, "not permitted to create leave policies")
		return
	}

	// ── Input ─────────────────────────────────────────────────────────────────
	var input models.LeaveTypeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	res, err := h.LeavePolicyService.Create(c, &input)
	if err != nil {
		errors.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "leave policy updated successfully",
		"res":     res,
	})
}

func (h *HandlerFunc) GetAllLeavePolicies(c *gin.Context) {
	res, err := h.LeavePolicyService.Get(c)
	if err != nil {
		errors.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *HandlerFunc) UpdateLeavePolicy(c *gin.Context) {

	if role := c.GetString("role"); role != accessrole.ROLE_SUPER_ADMIN &&
		role != accessrole.ROLE_ADMIN && role != accessrole.ROLE_HR {
		errors.RespondWithError(c, http.StatusUnauthorized, "not permitted to update leave policies")
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
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	res, err := h.LeavePolicyService.Update(c, leaveTypeID, &input)
	if err != nil {
		errors.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "leave policy updated successfully",
		"res":     res,
	})
}

func (h *HandlerFunc) DeleteLeavePolicy(c *gin.Context) {

	if role := c.GetString("role"); role != accessrole.ROLE_SUPER_ADMIN &&
		role != accessrole.ROLE_ADMIN && role != accessrole.ROLE_HR {
		errors.RespondWithError(c, http.StatusUnauthorized, "not permitted to delete leave policies")
		return
	}

	// ── Path param ────────────────────────────────────────────────────────────
	leaveTypeID, ok := parseLeaveTypeID(c)
	if !ok {
		return
	}
	if err := h.LeavePolicyService.Delete(c, leaveTypeID); err != nil {
		errors.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "leave policy deleted successfully",
		"id":      leaveTypeID,
	})
}

// parseLeaveTypeID parses and validates the ":id" path parameter.
func parseLeaveTypeID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid leave type ID")
		return 0, false
	}
	return id, true
}
