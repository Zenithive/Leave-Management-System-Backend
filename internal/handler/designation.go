package handler

import (
	"net/http"

	"github.com/Zenithive/LeaveManagementSystem/internal/config/database"
	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/accessrole"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/Zenithive/LeaveManagementSystem/pkg/constant"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CreateDesignation - POST /api/designations
// Only ADMIN, SUPERADMIN, and HR can create designations
func (h *HandlerFunc) CreateDesignation(c *gin.Context) {
	// 1️ Permission check
	role := c.GetString("role")

	if err := accessrole.Admin_SuperAdmin_Hr(role, "only ADMIN, SUPERADMIN, and HR can create designations"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	empId, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusForbidden, "Access Denied")
		return
	}

	// 2️ Bind input JSON
	var input *models.DesignationInput

	if err := c.ShouldBindJSON(&input); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	if err := h.Validator.Struct(input); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	// 3️ Create designation
	var designationID string
	err = database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		designationID, err = h.Query.CreateDesignation(tx, input)
		if err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to create designation: "+err.Error())
		}
		logData := models.NewCommon(constant.ComponentDesignation, constant.ActionCreate, empId)
		if err := h.Query.AddLog(logData, tx); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to create  degisnation log: "+err.Error())
		}
		return nil
	})
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 4️ Response
	c.JSON(http.StatusCreated, gin.H{
		"message":        "designation created successfully",
		"designation_id": designationID,
	})
}

// GetAllDesignations - GET /api/designations
// All authenticated users can view designations
func (h *HandlerFunc) GetAllDesignations(c *gin.Context) {
	// 1️ Fetch all designations
	designations, err := h.Query.GetAllDesignations()
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "failed to fetch designations: "+err.Error())
		return
	}

	// 2️ Response
	c.JSON(http.StatusOK, gin.H{
		"message":      "designations fetched successfully",
		"designations": designations,
	})
}

// GetDesignationByID - GET /api/designations/:id
// All authenticated users can view a specific designation
func (h *HandlerFunc) GetDesignationByID(c *gin.Context) {
	// 1️ Parse designation ID
	designationIDStr := c.Param("id")
	designationID, err := uuid.Parse(designationIDStr)
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid designation ID")
		return
	}

	// 2️ Fetch designation
	designation, err := h.Query.GetDesignationByID(designationID)
	if err != nil {
		errors.RespondWithError(c, http.StatusNotFound, "designation not found")
		return
	}

	// 3️ Response
	c.JSON(http.StatusOK, gin.H{
		"message":     "designation fetched successfully",
		"designation": designation,
	})
}

// UpdateDesignation - PATCH /api/designations/:id
// Only ADMIN, SUPERADMIN, and HR can update designations
func (h *HandlerFunc) UpdateDesignation(c *gin.Context) {
	// 1️ Permission check
	role := c.GetString("role")
	if err := accessrole.Admin_SuperAdmin_Hr(role, "only ADMIN, SUPERADMIN, and HR can update designations"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	empId, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusForbidden, "Access Denied")
		return
	}

	// 2️ Parse designation ID
	designationIDStr := c.Param("id")
	designationID, err := uuid.Parse(designationIDStr)
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid designation ID")
		return
	}

	// 3️ Bind input JSON
	var input *models.DesignationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	if err := h.Validator.Struct(input); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	// 4️ Update designation
	err = database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		err = h.Query.UpdateDesignation(tx, designationID, input)
		if err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to update designation: "+err.Error())
		}
		logData := models.NewCommon(constant.ComponentDesignation, constant.ActionUpdate, empId)
		if err := h.Query.AddLog(logData, tx); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to create  degisnation log: "+err.Error())
		}
		return nil
	})
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	// 5️ Response
	c.JSON(http.StatusOK, gin.H{
		"message":        "designation updated successfully",
		"designation_id": designationID,
	})
}

// DeleteDesignation - DELETE /api/designations/:id
// Only ADMIN, SUPERADMIN, and HR can delete designations
func (h *HandlerFunc) DeleteDesignation(c *gin.Context) {
	// 1️ Permission check
	role := c.GetString("role")
	if err := accessrole.Admin_SuperAdmin_Hr(role, "only ADMIN, SUPERADMIN, and HR can delete designations"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	empId, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusForbidden, "Access Denied")
		return
	}

	// 2️ Parse designation ID
	designationIDStr := c.Param("id")

	designationID, err := uuid.Parse(designationIDStr)
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid designation ID "+err.Error())
		return
	}

	// 3️ Delete designation (will set employee designation_id to NULL due to ON DELETE SET NULL)
	err = database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		err = h.Query.DeleteDesignation(tx, designationID)
		if err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to delete designation: "+err.Error())
		}
		logData := models.NewCommon(constant.ComponentDesignation, constant.ActionDelete, empId)
		if err := h.Query.AddLog(logData, tx); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to create  degisnation log: "+err.Error())
		}
		return nil
	})
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 4️ Response
	c.JSON(http.StatusOK, gin.H{
		"message": "designation deleted successfully. Employee designation_id set to NULL.",
	})
}
