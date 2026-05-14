package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
)

// POST /api/settings/holidays
func (h *HandlerFunc) AddHoliday(c *gin.Context) {
	callerID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "access denied")
		return
	}

	var input models.HolidayInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	if err := h.Validator.Struct(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	var result *service.AddHolidayResult
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		var err error
		result, err = h.HolidaySvc.Add(tx, input, callerID)
		return err
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Holiday added successfully",
		"id":      result.ID,
		"date":    result.Date,
	})
}

// GET /api/settings/holidays
func (h *HandlerFunc) GetHolidays(c *gin.Context) {
	holidays, err := h.HolidaySvc.GetAll()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, holidays)
}

// DELETE /api/settings/holidays/:id
func (h *HandlerFunc) DeleteHoliday(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "holiday ID is required")
		return
	}

	callerID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "access denied")
		return
	}

	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		return h.HolidaySvc.Delete(tx, id, callerID)
	}); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Holiday deleted successfully"})
}
