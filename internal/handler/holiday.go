package handler

import (
	"net/http"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/accessrole"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/gin-gonic/gin"
)

func (h *HandlerFunc) AddHoliday(c *gin.Context) {

	role := c.GetString("role")

	if role != accessrole.ROLE_SUPER_ADMIN &&
		role != accessrole.ROLE_ADMIN &&
		role != accessrole.ROLE_HR {
		errors.RespondWithError(c, http.StatusForbidden, "not permitted")
		return
	}
	var input models.Holiday

	if err := c.ShouldBindJSON(&input); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	if err := h.Validator.Struct(input); err != nil {
		errors.RespondWithError(c, http.StatusBadRequest, "validation failed: "+err.Error())
		return
	}

	id, err := h.Holidayservice.AddHoliday(c, &input)
	if err != nil {
		errors.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "holiday added successfully",
		"id":      id,
	})
}

func (h *HandlerFunc) GetHolidays(c *gin.Context) {

	data, err := h.Holidayservice.GetAllHolidays(c)
	if err != nil {
		errors.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

func (h *HandlerFunc) DeleteHoliday(c *gin.Context) {

	role := c.GetString("role")

	if err := accessrole.Admin_SuperAdmin_Hr(role, "only ADMIN, SUPERADMIN, and HR can delete holidays"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	id := c.Param("id")
	if id == "" {
		errors.RespondWithError(c, http.StatusBadRequest, "holiday id is required")
		return
	}

	if err := h.Holidayservice.DeleteHoliday(c, id); err != nil {
		errors.Error(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "holiday deleted successfully",
	})
}