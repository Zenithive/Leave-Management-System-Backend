package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/access_role"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

// AddHoliday handles adding a new holiday
func (s *HandlerFunc) AddHoliday(c *gin.Context) {
	role, _ := c.Get("role")
	fmt.Println("role", role)
	if role.(string) != constant.ROLE_SUPER_ADMIN && role.(string) != constant.ROLE_ADMIN && role.(string) != constant.ROLE_HR {
		utils.RespondWithError(c, http.StatusUnauthorized, "not permitted")
		return
	}
	empID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusForbidden, "Access Denied")
		return
	}
	var input *models.Holiday

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
		return
	}

	// Normalize date to UTC midnight to avoid timezone issues
	normalizedDate := time.Date(input.Date.Year(), input.Date.Month(), input.Date.Day(), 0, 0, 0, 0, time.UTC)
	var holidayId string

	err = common.ExecuteTransaction(c, s.Query.DB, func(tx *sqlx.Tx) error {
		id, err := s.Query.AddHoliday(tx, input.Name, normalizedDate, input.Type)
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to add holiday: "+err.Error())
		}
		holidayId = id
		data := models.NewCommon(constant.ComponentHoliday, constant.ActionCreate, empID)

		err = s.Query.AddLog(data, tx)
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to log action: "+err.Error())
		}

		return err
	})
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Holiday added successfully",
		"id":      holidayId,
		"date":    normalizedDate.Format("2006-01-02"),
	})
}

// GetHolidays returns all holidays
func (s *HandlerFunc) GetHolidays(c *gin.Context) {
	holidays, err := s.Query.GetAllHolidays()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch holidays: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, holidays)
}

// DeleteHoliday removes a holiday
func (s *HandlerFunc) DeleteHoliday(c *gin.Context) {
	role := c.GetString("role")
	if err := access_role.Admin_SuperAdmin_Hr(role, "only ADMIN, SUPERADMIN, and HR can delete designations"); err != nil {
		utils.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}
	id := c.Param("id")
	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Holiday ID is required")
		return
	}

	empID, err := common.GetEmployeeId(c)
	if err != nil {
		utils.RespondWithError(c, http.StatusForbidden, "Access Denied")
		return
	}

	err = common.ExecuteTransaction(c, s.Query.DB, func(tx *sqlx.Tx) error {
		err := s.Query.DeleteHoliday(id, tx)
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to delete holiday: "+err.Error())
		}
		data := models.NewCommon(constant.ComponentHoliday, constant.ActionDelete, empID)
		err = s.Query.AddLog(data, tx)
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to log action: "+err.Error())
		}
		return err
	})
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Holiday deleted successfully",
	})
}
