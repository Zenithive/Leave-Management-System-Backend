package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
)

// POST /api/designations
func (h *HandlerFunc) CreateDesignation(c *gin.Context) {
	callerID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "access denied")
		return
	}

	var input models.DesignationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := h.Validator.Struct(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	var designationID string
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		var err error
		designationID, err = h.DesignationSvc.Create(tx, &input, callerID)
		return err
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "designation created successfully",
		"designation_id": designationID,
	})
}

// GET /api/designations
func (h *HandlerFunc) GetAllDesignations(c *gin.Context) {
	designations, err := h.DesignationSvc.GetAll()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":      "designations fetched successfully",
		"designations": designations,
	})
}

// GET /api/designations/:id
func (h *HandlerFunc) GetDesignationByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid designation ID")
		return
	}

	designation, err := h.DesignationSvc.GetByID(id)
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "designation fetched successfully",
		"designation": designation,
	})
}

// PATCH /api/designations/:id
func (h *HandlerFunc) UpdateDesignation(c *gin.Context) {
	callerID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "access denied")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid designation ID")
		return
	}

	var input models.DesignationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := h.Validator.Struct(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		return h.DesignationSvc.Update(tx, id, &input, callerID)
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "designation updated successfully",
		"designation_id": id,
	})
}

// DELETE /api/designations/:id
func (h *HandlerFunc) DeleteDesignation(c *gin.Context) {
	callerID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "access denied")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid designation ID: "+err.Error())
		return
	}

	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		return h.DesignationSvc.Delete(tx, id, callerID)
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "designation deleted successfully. Employee designation_id set to NULL.",
	})
}
