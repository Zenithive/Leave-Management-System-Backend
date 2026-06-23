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

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/constant"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
)

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

	res, err := h.LeavePolicyService.Create(c, &input)
	if err != nil {
		utils.Error(c, err)
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
		utils.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *HandlerFunc) UpdateLeavePolicy(c *gin.Context) {

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
	res, err := h.LeavePolicyService.Update(c, leaveTypeID, &input)
	if err != nil {
		utils.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "leave policy updated successfully",
		"res":     res,
	})
}

func (h *HandlerFunc) DeleteLeavePolicy(c *gin.Context) {

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
	h.LeavePolicyService.Delete(c, leaveTypeID)

	c.JSON(http.StatusOK, gin.H{
		"message": "leave policy deleted successfully",
		"id":      leaveTypeID,
	})
}

// parseLeaveTypeID parses and validates the ":id" path parameter.
func parseLeaveTypeID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid leave type ID")
		return 0, false
	}
	return id, true
}

// func (h *HandlerFunc) writeLeaveTypeLog(c *gin.Context, tx *sqlx.Tx, action string, fromUserID uuid.UUID) error {
// 	data := &models.Common{
// 		Component:  constant.ComponentLeaveType,
// 		Action:     action,
// 		FromUserID: fromUserID,
// 	}
// 	if err := h.Query.AddLog(data, tx); err != nil {
// 		return utils.CustomErr(c, http.StatusInternalServerError, "failed to write audit log: "+err.Error())
// 	}
// 	return nil
// }
