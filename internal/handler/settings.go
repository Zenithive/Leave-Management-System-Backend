package handler

import (
	"net/http"
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

	// 3. Get Employee ID for Audit Logs
	empIDRaw, ok := c.Get("user_id")
	if !ok {
		errors.RespondWithError(c, http.StatusUnauthorized, "Employee ID missing")
		return
	}
	empID, _ := uuid.Parse(empIDRaw.(string))

	// 4. Execute Database Transaction
	err := database.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		if err := h.Query.UpdateCompanySettings(tx, input); err != nil {
			return err
		}
		return h.Query.AddLog(models.NewCommon(constant.CompanySettings, constant.ActionUpdate, empID), tx)
	})
	if err != nil {
		errors.RespondWithError(c, 500, "Failed to update settings: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Company settings updated successfully",
	})
}

// GetCompanyLogo - GET /api/settings/logo  (public — no auth required)
// Returns the company logo URL from the COMPANY_LOGO environment variable.
// Falls back to a placeholder image URL if the env var is not set.
func (h *HandlerFunc) GetCompanyLogo(c *gin.Context) {
	logoURL := h.Env.COMPANY_LOGO
	if logoURL == "" {
	logoURL = "https://ui-avatars.com/api/?name=Z&background=000000&color=ffffff&size=256&rounded=true"
}
	c.JSON(http.StatusOK, gin.H{
		"logo": logoURL,
	})
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
