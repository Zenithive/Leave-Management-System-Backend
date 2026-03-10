package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

// ======================
// Create Equipment Category
// ======================
func (h *HandlerFunc) CreateCategory(c *gin.Context) {
	// 1️ Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN, SUPERADMIN, and HR can create categories")
		return
	}

	// 2️ Get Employee ID for logging
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return
	}
	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return
	}

	// 3️ Bind JSON and validate
	var req models.EquipmentCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	// 4️ Execute transaction
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.CreateCategory(tx, req); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create category: "+err.Error())
		}
		logData := utils.NewCommon(constant.EquipmentCategory, constant.ActionCreate, empID)
		if err := common.AddLog(logData, tx); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create log: "+err.Error())
		}
		return nil
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "category created successfully",
	})
}

// ======================
// Get All Equipment Categories
// ======================
func (h *HandlerFunc) GetAllCategory(c *gin.Context) {
	// 1️ Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN, SUPERADMIN, and HR can view categories")
		return
	}

	// 2️ Fetch categories
	data, err := h.Query.GetAllCategory()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to get categories: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "success",
		"categories": data,
	})
}

// ======================
// Delete Equipment Category
// ======================
func (h *HandlerFunc) DeleteCategory(c *gin.Context) {
	// 1️ Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN, SUPERADMIN, and HR can delete categories")
		return
	}

	// 2️ Get Category ID from URL param
	idStr := c.Param("id")
	categoryID, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid category ID")
		return
	}

	// 3️ Get Employee ID for logging
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return
	}
	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return
	}

	// 4️ Execute deletion in transaction
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.DeleteCategory(tx, categoryID); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to delete category: "+err.Error())
		}
		logData := utils.NewCommon(constant.EquipmentCategory, constant.ActionDelete, empID)
		if err := common.AddLog(logData, tx); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create log: "+err.Error())
		}
		return nil
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "category deleted successfully",
	})
}

func (h *HandlerFunc) UpdateCategory(c *gin.Context) {
	// 1️ Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN, SUPERADMIN, and HR can update categories")
		return
	}

	// 2️ Get category ID from URL param
	idStr := c.Param("id")
	categoryID, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid category ID")
		return
	}

	// 3️ Get employee ID for logging
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return
	}
	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return
	}

	// 4️ Bind JSON + validate
	var req models.EquipmentCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	// 5️ Execute transaction
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.UpdateCategory(tx, categoryID, req); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to update category: "+err.Error())
		}

		logData := utils.NewCommon(constant.EquipmentCategory, constant.ActionUpdate, empID)
		if err := common.AddLog(logData, tx); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create log: "+err.Error())
		}

		return nil
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 6️ Success response
	c.JSON(http.StatusOK, gin.H{
		"message": "category updated successfully",
	})
}

// CreateEquipment

func (h *HandlerFunc) CreateEquipment(c *gin.Context) {
	// Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN, SUPERADMIN, and HR can create equipment")
		return
	}

	// Get employee ID for logging
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return
	}
	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return
	}

	// Bind JSON + validation
	var req models.EquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	// Execute transaction
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.CreateEquipment(tx, req); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create equipment: "+err.Error())
		}

		logData := utils.NewCommon(constant.Equipment, constant.ActionCreate, empID)
		if err := common.AddLog(logData, tx); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create log: "+err.Error())
		}

		return nil
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "equipment created successfully",
	})
}

// GetEquipmentByCategory fetches all equipment for a specific category.
// Steps:
// 1. Check permissions (SUPERADMIN, ADMIN only)
// 2. Get category ID from query params
// 3. Call repository to fetch equipment by category
// 4. Return response
func (h *HandlerFunc) GetEquipmentByCategory(c *gin.Context) {
	// 1️ Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN and SUPERADMIN can view equipment")
		return
	}

	// 2️ Get category ID from  parameter
	categoryIDStr := c.Query("id")
	if categoryIDStr == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "category_id query parameter is required")
		return
	}

	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid category ID")
		return
	}

	// 3️ Fetch equipment by category from repository
	data, err := h.Query.GetEquipmentByCategory(categoryID)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to get equipment: "+err.Error())
		return
	}

	// 4️ Ensure we return empty array instead of null when no data
	if data == nil {
		data = []models.EquipmentRes{}
	}

	// 5️ Return success response
	c.JSON(http.StatusOK, gin.H{
		"message":   "success",
		"equipment": data,
	})
}

// GetAllEquipment fetches all equipment.
// Permission: SUPERADMIN, ADMIN only
func (h *HandlerFunc) GetAllEquipment(c *gin.Context) {
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN and SUPERADMIN can view equipment")
		return
	}

	data, err := h.Query.GetAllEquipment()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to get equipment: "+err.Error())
		return
	}

	// Ensure we return empty array instead of null when no data
	if data == nil {
		data = []models.EquipmentRes{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "success",
		"equipment": data,
	})
}

func (h *HandlerFunc) UpdateEquipment(c *gin.Context) {
	// Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN, SUPERADMIN, and HR can update equipment")
		return
	}

	// Get equipment ID from URL
	idStr := c.Param("id")
	equipmentID, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid equipment ID")
		return
	}

	// Get employee ID for logging
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return
	}
	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return
	}

	// Bind JSON + validate
	var req models.EquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	// Execute transaction
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.UpdateEquipment(tx, equipmentID, req); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to update equipment: "+err.Error())
		}

		logData := utils.NewCommon(constant.Equipment, constant.ActionUpdate, empID)
		if err := common.AddLog(logData, tx); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create log: "+err.Error())
		}

		return nil
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "equipment updated successfully",
	})
}

func (h *HandlerFunc) DeleteEquipment(c *gin.Context) {
	// Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN, SUPERADMIN, and HR can delete equipment")
		return
	}

	// Get equipment ID from URL
	idStr := c.Param("id")
	equipmentID, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid equipment ID")
		return
	}

	// Get employee ID for logging
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return
	}
	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return
	}

	// Execute deletion in transaction
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.DeleteEquipment(tx, equipmentID); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to delete equipment: "+err.Error())
		}

		logData := utils.NewCommon(constant.Equipment, constant.ActionDelete, empID)
		if err := common.AddLog(logData, tx); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create log: "+err.Error())
		}

		return nil
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "equipment deleted successfully",
	})
}

func (h *HandlerFunc) AssignEquipment(c *gin.Context) {
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "access denied")
		return
	}
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return
	}
	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return
	}

	var req models.AssignEquipmentRequest
	req.AssignedBy = empID
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.AssignEquipment(tx, req); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to assign equipment: "+err.Error())
		}
		logData := utils.NewCommon(constant.EquipmentAssign, constant.ActionCreate, empID)
		if err := common.AddLog(logData, tx); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create log: "+err.Error())
		}
		return nil

	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "equipment assigned successfully",
	})
}

func (h *HandlerFunc) GetAllAssignedEquipment(c *gin.Context) {
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "access denied")
		return
	}
	data, err := h.Query.GetAllAssignedEquipment()
	fmt.Println("data", data)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *HandlerFunc) GetAssignedEquipmentByEmployee(c *gin.Context) {
	id := c.Param("id")
	employeeID, err := uuid.Parse(id)
	data, err := h.Query.GetAssignedEquipmentByEmployee(employeeID)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// RemoveEquipment removes/returns equipment from an employee
func (h *HandlerFunc) RemoveEquipment(c *gin.Context) {
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "access denied")
		return
	}

	// Get employee ID for logging
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return
	}
	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return
	}

	var req models.RemoveEquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := models.Validate.Struct(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.RemoveEquipment(tx, req); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, err.Error())
		}

		logData := utils.NewCommon(constant.Equipment, constant.ActionDelete, empID)
		if err := common.AddLog(logData, tx); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create log: "+err.Error())
		}

		return nil
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "equipment removed successfully",
	})
}

// UpdateAssignment handles both quantity updates and reassignments
func (h *HandlerFunc) UpdateAssignment(c *gin.Context) {
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "access denied")
		return
	}

	// Get employee ID for logging
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return
	}
	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return
	}
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return
	}

	var req models.UpdateAssignmentRequest
	req.AssignedBy = empID
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := models.Validate.Struct(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.UpdateAssignment(tx, req); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, err.Error())
		}

		logData := utils.NewCommon(constant.Equipment, constant.ActionUpdate, empID)
		if err := common.AddLog(logData, tx); err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "failed to create log: "+err.Error())
		}

		return nil
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Determine response message based on operation type
	message := "assignment updated successfully"
	if req.ToEmployeeID != nil {
		message = "equipment reassigned successfully"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
}
