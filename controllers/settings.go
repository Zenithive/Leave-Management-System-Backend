package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/access_role"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

// GetCompanySettings - GET /api/settings/company
func (h *HandlerFunc) GetCompanySettings(c *gin.Context) {
	// Only SUPERADMIN and ADMIN allowed
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)
	if role != "SUPERADMIN" && role != "ADMIN" {
		utils.RespondWithError(c, 403, "Not authorized to view settings")
		return
	}
	var settings models.CompanySettings
	err := h.Query.GetCompanySettings(&settings)
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to fetch settings: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
	})
}

/*
// UpdateCompanySettings - PUT /api/settings/company
func (h *HandlerFunc) UpdateCompanySettings(c *gin.Context) {
	// Only SUPERADMIN and ADMIN allowed
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)
	if role != "SUPERADMIN" && role != "ADMIN" {
		utils.RespondWithError(c, 403, "Not authorized to update settings")
		return
	}
	var input models.CompanyField


	if err := c.ShouldBindWith(&input, binding.FormMultipart); err != nil {
		utils.RespondWithError(c, 400, "Invalid input (Form error): "+err.Error())
		return
	}
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "Employee ID missing")
		return

	}

	empIDStr, ok := empIDRaw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee ID format")
		return
	}

	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid employee UUID")
		return
	}

	err = common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		err := h.Query.UpdateCompanySettings(tx, input)
		if err != nil {
			return utils.CustomErr(c, 500, "Failed to fetch settings: "+err.Error())
		}
		//add log
		data := models.NewCommon(constant.CompanySettings, constant.ActionCreate, empID)

		err = h.Query.AddLog(data, tx)
		if err != nil {
			return utils.CustomErr(c, http.StatusInternalServerError, "Failed to log action: "+err.Error())
		}
		return err
	})

	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update settings: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Company settings updated successfully",
	})
}
*/

func (h *HandlerFunc) UpdateCompanySettings(c *gin.Context) {
	// 1. Authorization check
	roleRaw, _ := c.Get("role")
	role, ok := roleRaw.(string)
	if !ok || (role != "SUPERADMIN" && role != "ADMIN") {
		utils.RespondWithError(c, 403, "Not authorized to update settings")
		return
	}

	// 2. Extract values from multipart form
	workingDays, _ := strconv.Atoi(c.PostForm("WorkingDaysPerMonth"))
	if workingDays == 0 {
		workingDays = 22
	}

	input := models.CompanyField{
		WorkingDaysPerMonth:     workingDays,
		AllowManagerAddLeave:    c.PostForm("AllowManagerAddLeave") == "true",
		CompanyName:             c.PostForm("CompanyName"),
		PrimaryColor:            c.PostForm("PrimaryColor"),
		SecondaryColor:          c.PostForm("SecondaryColor"),
		BirthdayMessageTemplate: c.PostForm("BirthdayMessageTemplate"),
	}

	// 3. Handle Logo File Upload
	var logoPath string
	file, err := c.FormFile("Logo")
	if err == nil {
		logoPath = "uploads/logos/" + uuid.New().String() + "-" + file.Filename
		if err := c.SaveUploadedFile(file, logoPath); err != nil {
			utils.RespondWithError(c, 500, "Failed to save logo file")
			return
		}
	}

	// 4. Get Employee ID for Audit Logs
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "Employee ID missing")
		return
	}
	empID, _ := uuid.Parse(empIDRaw.(string))

	// 5. Execute Database Transaction
	err = common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.UpdateCompanySettings(tx, input, logoPath); err != nil {
			return err
		}
		return h.Query.AddLog(models.NewCommon(constant.CompanySettings, constant.ActionUpdate, empID), tx)
	})
	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update settings: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Company settings updated successfully",
		"logo":    logoPath,
	})
}

// PreviewBirthdayMessage - GET /api/settings/birthday-preview?name=John&birth_date=1995-04-16
// Returns the rendered birthday message using the current template and provided placeholders.
func (h *HandlerFunc) PreviewBirthdayMessage(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin(c.GetString("role"), "not authorized"); err != nil {
		utils.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	name := c.DefaultQuery("name", "Employee")
	birthDateStr := c.Query("birth_date") // optional, format: 2006-01-02

	var birthDate *time.Time
	if birthDateStr != "" {
		t, err := time.Parse("2006-01-02", birthDateStr)
		if err != nil {
			utils.RespondWithError(c, http.StatusBadRequest, "invalid birth_date format, use YYYY-MM-DD")
			return
		}
		birthDate = &t
	}

	tmpl, err := h.Query.GetBirthdayMessageTemplate()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to fetch template: "+err.Error())
		return
	}

	rendered := repositories.RenderBirthdayMessage(tmpl, name, birthDate)

	c.JSON(http.StatusOK, gin.H{
		"template": tmpl,
		"rendered": rendered,
		"placeholders": []string{"{name}", "{date}", "{age}"},
	})
}

// GetTodayBirthdays - GET /api/settings/birthdays/today
// Returns all employees whose birthday is today, with their rendered birthday message.
func (h *HandlerFunc) GetTodayBirthdays(c *gin.Context) {
	if err := access_role.Admin_SuperAdmin_Hr(c.GetString("role"), "not authorized"); err != nil {
		utils.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	tmpl, err := h.Query.GetBirthdayMessageTemplate()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to fetch template: "+err.Error())
		return
	}

	employees, err := h.Query.GetTodayBirthdays()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to fetch birthdays: "+err.Error())
		return
	}

	type BirthdayEntry struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Message string `json:"message"`
	}

	result := make([]BirthdayEntry, 0, len(employees))
	for _, emp := range employees {
		result = append(result, BirthdayEntry{
			ID:      emp.ID,
			Name:    emp.Name,
			Email:   emp.Email,
			Message: repositories.RenderBirthdayMessage(tmpl, emp.Name, emp.BirthDate),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"date":    time.Now().Format("2006-01-02"),
		"total":   len(result),
		"data":    result,
	})
}
