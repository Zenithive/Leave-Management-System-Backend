package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/config/database"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/access_role"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/common/errors"
	pagi "github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/common/pagination"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/constant"
)

// ======================
// CATEGORY
// ======================

func (h *HandlerFunc) CreateCategory(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "only ADMIN, SUPERADMIN, and HR can create categories"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.EquipmentCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.CreateCategory(tx, req); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to create category: "+err.Error())
		}
		return h.Query.AddLog(models.NewCommon(constant.EquipmentCategory, constant.ActionCreate, empID), tx)
	}); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "category created successfully"})
}

func (h *HandlerFunc) GetAllCategory(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "only ADMIN, SUPERADMIN, and HR can view categories"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	pagination := pagi.GetPaginationParams(c)
	filters := pagi.GetFilterParams(c, pagi.CategorySortFields)

	data, total, err := h.Query.GetAllCategory(
		pagination.PageSize, pagination.Offset,
		filters.Search, filters.SortBy, filters.SortDir,
	)
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "failed to get categories: "+err.Error())
		return
	}

	response := gin.H{"message": "success", "categories": data}
	if pagination.PageSize > 0 {
		response["pagination"] = pagi.CalculatePaginationResponse(pagination.Page, pagination.PageSize, total)
	}
	c.JSON(http.StatusOK, response)
}

func (h *HandlerFunc) UpdateCategory(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "only ADMIN, SUPERADMIN, and HR can update categories"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid category ID")
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.EquipmentCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.UpdateCategory(tx, categoryID, req); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to update category: "+err.Error())
		}
		return h.Query.AddLog(models.NewCommon(constant.EquipmentCategory, constant.ActionUpdate, empID), tx)
	}); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "category updated successfully"})
}

func (h *HandlerFunc) DeleteCategory(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "only ADMIN, SUPERADMIN, and HR can delete categories"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid category ID")
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	if err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.DeleteCategory(tx, categoryID); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to delete category: "+err.Error())
		}
		return h.Query.AddLog(models.NewCommon(constant.EquipmentCategory, constant.ActionDelete, empID), tx)
	}); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "category deleted successfully"})
}

// ======================
// EQUIPMENT
// ======================

func (h *HandlerFunc) CreateEquipment(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "only ADMIN, SUPERADMIN, and HR can create equipment"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.EquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.CreateEquipment(tx, req); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to create equipment: "+err.Error())
		}
		return h.Query.AddLog(models.NewCommon(constant.Equipment, constant.ActionCreate, empID), tx)
	}); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "equipment created successfully"})
}

func (h *HandlerFunc) GetAllEquipment(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin(c.GetString("role"), "only ADMIN and SUPERADMIN can view equipment"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	pagination := pagi.GetPaginationParams(c)
	filters := pagi.GetFilterParams(c, pagi.EquipmentSortFields)

	data, total, err := h.Query.GetAllEquipment(
		pagination.PageSize, pagination.Offset,
		filters.Search, filters.SortBy, filters.SortDir,
	)
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "failed to get equipment: "+err.Error())
		return
	}
	if data == nil {
		data = []models.EquipmentRes{}
	}

	response := gin.H{"message": "success", "equipment": data}
	if pagination.PageSize > 0 {
		response["pagination"] = pagi.CalculatePaginationResponse(pagination.Page, pagination.PageSize, total)
	}
	c.JSON(http.StatusOK, response)
}

func (h *HandlerFunc) GetEquipmentByCategory(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin(c.GetString("role"), "only ADMIN and SUPERADMIN can view equipment"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	categoryIDStr := c.Query("id")
	if categoryIDStr == "" {
		errors.RespondWithError(c, http.StatusBadRequest, "category_id query parameter is required")
		return
	}
	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid category ID")
		return
	}

	pagination := pagi.GetPaginationParams(c)
	filters := pagi.GetFilterParams(c, pagi.EquipmentSortFields)

	data, total, err := h.Query.GetEquipmentByCategory(
		categoryID,
		pagination.PageSize, pagination.Offset,
		filters.Search, filters.SortBy, filters.SortDir,
	)
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "failed to get equipment: "+err.Error())
		return
	}
	if data == nil {
		data = []models.EquipmentRes{}
	}

	response := gin.H{"message": "success", "equipment": data}
	if pagination.PageSize > 0 {
		response["pagination"] = pagi.CalculatePaginationResponse(pagination.Page, pagination.PageSize, total)
	}
	c.JSON(http.StatusOK, response)
}

func (h *HandlerFunc) UpdateEquipment(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "only ADMIN, SUPERADMIN, and HR can update equipment"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	equipmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid equipment ID")
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.EquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.UpdateEquipment(tx, equipmentID, req); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to update equipment: "+err.Error())
		}
		return h.Query.AddLog(models.NewCommon(constant.Equipment, constant.ActionUpdate, empID), tx)
	}); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "equipment updated successfully"})
}

func (h *HandlerFunc) DeleteEquipment(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "only ADMIN, SUPERADMIN, and HR can delete equipment"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	equipmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid equipment ID")
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	if err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.DeleteEquipment(tx, equipmentID); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to delete equipment: "+err.Error())
		}
		return h.Query.AddLog(models.NewCommon(constant.Equipment, constant.ActionDelete, empID), tx)
	}); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "equipment deleted successfully"})
}

// ======================
// ASSIGNMENT
// ======================

func (h *HandlerFunc) AssignEquipment(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "access denied"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.AssignEquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	// Always override assigned_by from auth context — never trust the request body
	req.AssignedBy = empID

	if err := models.Validate.Struct(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.AssignEquipment(tx, req); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, "failed to assign equipment: "+err.Error())
		}
		return h.Query.AddLog(models.NewCommon(constant.EquipmentAssign, constant.ActionCreate, empID), tx)
	}); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "equipment assigned successfully"})
}

func (h *HandlerFunc) GetAllAssignedEquipment(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "access denied"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	pagination := pagi.GetPaginationParams(c)
	filters := pagi.GetFilterParams(c, pagi.AssignmentSortFields)

	data, total, err := h.Query.GetAllAssignedEquipment(
		pagination.PageSize, pagination.Offset,
		filters.Search, filters.SortBy, filters.SortDir,
	)
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := gin.H{"message": "success", "data": data}
	if pagination.PageSize > 0 {
		response["pagination"] = pagi.CalculatePaginationResponse(pagination.Page, pagination.PageSize, total)
	}
	c.JSON(http.StatusOK, response)
}

func (h *HandlerFunc) GetAssignedEquipmentByEmployee(c *gin.Context) {
	employeeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	pagination := pagi.GetPaginationParams(c)
	filters := pagi.GetFilterParams(c, pagi.AssignmentSortFields)

	data, total, err := h.Query.GetAssignedEquipmentByEmployee(
		employeeID,
		pagination.PageSize, pagination.Offset,
		filters.Search, filters.SortBy, filters.SortDir,
	)
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	response := gin.H{"message": "success", "data": data}
	if pagination.PageSize > 0 {
		response["pagination"] = pagi.CalculatePaginationResponse(pagination.Page, pagination.PageSize, total)
	}
	c.JSON(http.StatusOK, response)
}

func (h *HandlerFunc) RemoveEquipment(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "access denied"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.RemoveEquipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := models.Validate.Struct(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.RemoveEquipment(tx, req); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, err.Error())
		}
		return h.Query.AddLog(models.NewCommon(constant.Equipment, constant.ActionDelete, empID), tx)
	}); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "equipment removed successfully"})
}

func (h *HandlerFunc) UpdateAssignment(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "access denied"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		errors.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	var req models.UpdateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	// Always override assigned_by from auth context — never trust the request body
	req.AssignedBy = empID

	if err := models.Validate.Struct(&req); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	if err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.UpdateAssignment(tx, req); err != nil {
			return errors.CustomErr(c, http.StatusInternalServerError, err.Error())
		}
		return h.Query.AddLog(models.NewCommon(constant.Equipment, constant.ActionUpdate, empID), tx)
	}); err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	message := "assignment updated successfully"
	if req.ToEmployeeID != nil {
		message = "equipment reassigned successfully"
	}
	c.JSON(http.StatusOK, gin.H{"message": message})
}
