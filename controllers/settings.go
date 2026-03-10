package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
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
		fmt.Println("error", err.Error())
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
		data := utils.NewCommon(constant.CompanySettings, constant.ActionCreate, empID)

		err = common.AddLog(data, tx)
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

	// 2. THE BYPASS: Extract values directly from FormPost
	// This avoids the "invalid character '-'" JSON error entirely
	workingDays, _ := strconv.Atoi(c.PostForm("WorkingDaysPerMonth"))
	if workingDays == 0 {
		workingDays = 22 // Default fallback
	}

	input := models.CompanyField{
		WorkingDaysPerMonth:  workingDays,
		AllowManagerAddLeave: c.PostForm("AllowManagerAddLeave") == "true",
		CompanyName:          c.PostForm("CompanyName"),
		PrimaryColor:         c.PostForm("PrimaryColor"),
		SecondaryColor:       c.PostForm("SecondaryColor"),
	}

	// 3. Handle Logo File Upload
	var logoPath string
	file, err := c.FormFile("Logo") // "Logo" must match the key in your React FormData
	if err == nil {
		// Ensure you have an 'uploads' directory in your project root
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
	empIDStr := empIDRaw.(string)
	empID, _ := uuid.Parse(empIDStr)

	// 5. Execute Database Transaction
	err = common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		// Note: Update your Repo function to accept logoPath as the 3rd argument
		err := h.Query.UpdateCompanySettings(tx, input, logoPath)
		if err != nil {
			return err
		}

		// Add Audit Log
		data := utils.NewCommon(constant.CompanySettings, constant.ActionUpdate, empID)
		return common.AddLog(data, tx)
	})

	if err != nil {
		utils.RespondWithError(c, 500, "Failed to update settings: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Company settings updated successfully",
		"logo":    logoPath, // Optional: send back the path to the frontend
	})
}
