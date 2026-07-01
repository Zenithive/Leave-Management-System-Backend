package handler

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Zenithive/LeaveManagementSystem/internal/config/database"
	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/internal/service"
	accessrole "github.com/Zenithive/LeaveManagementSystem/pkg/accessrole"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/Zenithive/LeaveManagementSystem/pkg/constant"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetCompanySettings - GET /api/settings/company
func (h *HandlerFunc) GetCompanySettings(c *gin.Context) {
	// Only SUPERADMIN and ADMIN allowed
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)
	if role != "SUPERADMIN" && role != "ADMIN" {
		errors.RespondWithError(c, 403, "Not authorized to view settings")
		return
	}
	var settings models.CompanySettings
	err := h.Query.GetCompanySettings(&settings)
	if err != nil {
		errors.RespondWithError(c, 500, "Failed to fetch settings: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
	})
}

func (h *HandlerFunc) UpdateCompanySettings(c *gin.Context) {
	// 1. Authorization check
	roleRaw, _ := c.Get("role")
	role, ok := roleRaw.(string)
	if !ok || (role != "SUPERADMIN" && role != "ADMIN") {
		errors.RespondWithError(c, 403, "Not authorized to update settings")
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
	var newLogoPath string
	file, err := c.FormFile("Logo")
	if err == nil {
		newLogoPath = "uploads/logos/" + uuid.New().String() + "-" + file.Filename
		if err := c.SaveUploadedFile(file, newLogoPath); err != nil {
			errors.RespondWithError(c, 500, "Failed to save logo file")
			return
		}
	}

	// 4. Get Employee ID for Audit Logs
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		errors.RespondWithError(c, http.StatusUnauthorized, "Employee ID missing")
		return
	}
	empID, _ := uuid.Parse(empIDRaw.(string))

	// 5. Fetch old logo path before overwriting — so we can delete it after commit
	var oldLogoPath string
	if newLogoPath != "" {
		_ = h.Query.GetCompanyLogoPath(&oldLogoPath)
	}

	// 6. Execute Database Transaction
	err = database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.UpdateCompanySettings(tx, input, newLogoPath); err != nil {
			return err
		}
		return h.Query.AddLog(models.NewCommon(constant.CompanySettings, constant.ActionUpdate, empID), tx)
	})
	if err != nil {
		// Transaction failed — remove the newly saved file to avoid orphans
		if newLogoPath != "" {
			_ = os.Remove(newLogoPath)
		}
		errors.RespondWithError(c, 500, "Failed to update settings: "+err.Error())
		return
	}

	// 7. Delete the old logo file now that the DB is committed
	if oldLogoPath != "" && oldLogoPath != newLogoPath {
		_ = os.Remove(oldLogoPath)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Company settings updated successfully",
		"logo":    newLogoPath,
	})
}

// GetCompanyLogo - GET /api/settings/logo  (public — no auth required)
// Returns the company logo file directly as an image response.
// If no logo has been uploaded yet, returns 404.
func (h *HandlerFunc) GetCompanyLogo(c *gin.Context) {
	var logoPath string
	err := h.Query.GetCompanyLogoPath(&logoPath)
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "failed to fetch logo: "+err.Error())
		return
	}
	if logoPath == "" {
		errors.RespondWithError(c, http.StatusNotFound, "no logo uploaded yet")
		return
	}
	c.File(logoPath)
}
// Returns the rendered birthday message using the current template and provided placeholders.
func (h *HandlerFunc) PreviewBirthdayMessage(c *gin.Context) {
	if err := accessrole.Admin_SuperAdmin(c.GetString("role"), "not authorized"); err != nil {
		errors.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	name := c.DefaultQuery("name", "Employee")
	birthDateStr := c.Query("birth_date") // optional, format: 2006-01-02

	var birthDate *time.Time
	if birthDateStr != "" {
		t, err := time.Parse("2006-01-02", birthDateStr)
		if err != nil {
			errors.RespondWithError(c, http.StatusBadRequest, "invalid birth_date format, use YYYY-MM-DD")
			return
		}
		birthDate = &t
	}

	tmpl, err := h.Query.GetBirthdayMessageTemplate()
	if err != nil {
		errors.RespondWithError(c, http.StatusInternalServerError, "failed to fetch template: "+err.Error())
		return
	}

	rendered := service.RenderBirthdayMessage(tmpl, name, birthDate)

	c.JSON(http.StatusOK, gin.H{
		"template":     tmpl,
		"rendered":     rendered,
		"placeholders": []string{"{name}", "{date}", "{age}"},
	})
}
