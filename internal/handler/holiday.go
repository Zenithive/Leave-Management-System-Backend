package handler

import (
	"net/http"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/access_role"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/gin-gonic/gin"
)

func (h *HandlerFunc) AddHoliday(c *gin.Context) {

	role := c.GetString("role")

	if role != access_role.ROLE_SUPER_ADMIN &&
		role != access_role.ROLE_ADMIN &&
		role != access_role.ROLE_HR {
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

	if err := access_role.Admin_SuperAdmin_Hr(role, "only ADMIN, SUPERADMIN, and HR can delete holidays"); err != nil {
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

// AddHoliday handles adding a new holiday
// func (s *HandlerFunc) AddHoliday(c *gin.Context) {
// 	role, _ := c.Get("role")
// 	if role.(string) != access_role.ROLE_SUPER_ADMIN && role.(string) != access_role.ROLE_ADMIN && role.(string) != access_role.ROLE_HR {
// 		errors.RespondWithError(c, http.StatusUnauthorized, "not permitted")
// 		return
// 	}
// 	empID, err := common.GetEmployeeId(c)
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusForbidden, "Access Denied")
// 		return
// 	}
// 	var input *models.Holiday

// 	if err := c.ShouldBindJSON(&input); err != nil {
// 		errors.RespondWithError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
// 		return
// 	}

// 	if err := s.Validator.Struct(input); err != nil {
// 		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
// 		return
// 	}

// 	// Normalize date to UTC midnight to avoid timezone issues
// 	normalizedDate := time.Date(input.Date.Year(), input.Date.Month(), input.Date.Day(), 0, 0, 0, 0, time.UTC)
// 	var holidayId string

// 	err =database.ExecuteTransaction(c, s.Query.DB, func(tx *sqlx.Tx) error {
// 		id, err := s.Query.AddHoliday(tx, input.Name, normalizedDate, input.Type)
// 		if err != nil {
// 			return errors.CustomErr(c, http.StatusInternalServerError, "Failed to add holiday: "+err.Error())
// 		}
// 		holidayId = id
// 		data := models.NewCommon(constant.ComponentHoliday, constant.ActionCreate, empID)

// 		err = s.Query.AddLog(data, tx)
// 		if err != nil {
// 			return errors.CustomErr(c, http.StatusInternalServerError, "Failed to log action: "+err.Error())
// 		}

// 		return err
// 	})
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Holiday added successfully",
// 		"id":      holidayId,
// 		"date":    normalizedDate.Format("2006-01-02"),
// 	})
// }

// // GetHolidays returns all holidays
// func (s *HandlerFunc) GetHolidays(c *gin.Context) {
// 	holidays, err := s.Query.GetAllHolidays()
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch holidays: "+err.Error())
// 		return
// 	}
// 	c.JSON(http.StatusOK, holidays)
// }

// // DeleteHoliday removes a holiday
// func (s *HandlerFunc) DeleteHoliday(c *gin.Context) {
// 	role := c.GetString("role")
// 	if err := access_role.Admin_SuperAdmin_Hr(role, "only ADMIN, SUPERADMIN, and HR can delete designations"); err != nil {
// 		errors.RespondWithError(c, http.StatusForbidden, err.Error())
// 		return
// 	}
// 	id := c.Param("id")
// 	if id == "" {
// 		errors.RespondWithError(c, http.StatusBadRequest, "Holiday ID is required")
// 		return
// 	}

// 	empID, err := common.GetEmployeeId(c)
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusForbidden, "Access Denied")
// 		return
// 	}

// 	err =database.ExecuteTransaction(c, s.Query.DB, func(tx *sqlx.Tx) error {
// 		err := s.Query.DeleteHoliday(id, tx)
// 		if err != nil {
// 			return errors.CustomErr(c, http.StatusInternalServerError, "Failed to delete holiday: "+err.Error())
// 		}
// 		data := models.NewCommon(constant.ComponentHoliday, constant.ActionDelete, empID)
// 		err = s.Query.AddLog(data, tx)
// 		if err != nil {
// 			return errors.CustomErr(c, http.StatusInternalServerError, "Failed to log action: "+err.Error())
// 		}
// 		return err
// 	})
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Holiday deleted successfully",
// 	})
// }

// package handler

// import (
// 	"net/http"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/jmoiron/sqlx"
// 	"github.com/Zenithive/LeaveManagementSystem/internal/models"
// 	"github.com/Zenithive/LeaveManagementSystem/pkg"
// 	"github.com/Zenithive/LeaveManagementSystem/pkg/access_role"
// 	"github.com/Zenithive/LeaveManagementSystem/pkg/common"
// 	"github.com/Zenithive/LeaveManagementSystem/pkg/constant"
// )

// // AddHoliday handles adding a new holiday
// func (s *HandlerFunc) AddHoliday(c *gin.Context) {
// 	role, _ := c.Get("role")
// 	if role.(string) != access_role.ROLE_SUPER_ADMIN && role.(string) != access_role.ROLE_ADMIN && role.(string) != access_role.ROLE_HR {
// 		errors.RespondWithError(c, http.StatusUnauthorized, "not permitted")
// 		return
// 	}
// 	empID, err := common.GetEmployeeId(c)
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusForbidden, "Access Denied")
// 		return
// 	}
// 	var input *models.Holiday

// 	if err := c.ShouldBindJSON(&input); err != nil {
// 		errors.RespondWithError(c, http.StatusBadRequest, "Invalid input: "+err.Error())
// 		return
// 	}

// 	if err := s.Validator.Struct(input); err != nil {
// 		errors.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
// 		return
// 	}

// 	// Normalize date to UTC midnight to avoid timezone issues
// 	normalizedDate := time.Date(input.Date.Year(), input.Date.Month(), input.Date.Day(), 0, 0, 0, 0, time.UTC)
// 	var holidayId string

// 	err =database.ExecuteTransaction(c, s.Query.DB, func(tx *sqlx.Tx) error {
// 		id, err := s.Query.AddHoliday(tx, input.Name, normalizedDate, input.Type)
// 		if err != nil {
// 			return errors.CustomErr(c, http.StatusInternalServerError, "Failed to add holiday: "+err.Error())
// 		}
// 		holidayId = id
// 		data := models.NewCommon(constant.ComponentHoliday, constant.ActionCreate, empID)

// 		err = s.Query.AddLog(data, tx)
// 		if err != nil {
// 			return errors.CustomErr(c, http.StatusInternalServerError, "Failed to log action: "+err.Error())
// 		}

// 		return err
// 	})
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Holiday added successfully",
// 		"id":      holidayId,
// 		"date":    normalizedDate.Format("2006-01-02"),
// 	})
// }

// // GetHolidays returns all holidays
// func (s *HandlerFunc) GetHolidays(c *gin.Context) {
// 	holidays, err := s.Query.GetAllHolidays()
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch holidays: "+err.Error())
// 		return
// 	}
// 	c.JSON(http.StatusOK, holidays)
// }

// // DeleteHoliday removes a holiday
// func (s *HandlerFunc) DeleteHoliday(c *gin.Context) {
// 	role := c.GetString("role")
// 	if err := access_role.Admin_SuperAdmin_Hr(role, "only ADMIN, SUPERADMIN, and HR can delete designations"); err != nil {
// 		errors.RespondWithError(c, http.StatusForbidden, err.Error())
// 		return
// 	}
// 	id := c.Param("id")
// 	if id == "" {
// 		errors.RespondWithError(c, http.StatusBadRequest, "Holiday ID is required")
// 		return
// 	}

// 	empID, err := common.GetEmployeeId(c)
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusForbidden, "Access Denied")
// 		return
// 	}

// 	err =database.ExecuteTransaction(c, s.Query.DB, func(tx *sqlx.Tx) error {
// 		err := s.Query.DeleteHoliday(id, tx)
// 		if err != nil {
// 			return errors.CustomErr(c, http.StatusInternalServerError, "Failed to delete holiday: "+err.Error())
// 		}
// 		data := models.NewCommon(constant.ComponentHoliday, constant.ActionDelete, empID)
// 		err = s.Query.AddLog(data, tx)
// 		if err != nil {
// 			return errors.CustomErr(c, http.StatusInternalServerError, "Failed to log action: "+err.Error())
// 		}
// 		return err
// 	})
// 	if err != nil {
// 		errors.RespondWithError(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Holiday deleted successfully",
// 	})
// }
