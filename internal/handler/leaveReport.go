package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	accessrole "github.com/Zenithive/LeaveManagementSystem/pkg/accessrole"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/gin-gonic/gin"
)

func (h *HandlerFunc) GetLeaveReport(c *gin.Context) {
	reportType := c.Query("report_type")
	if reportType == "" {
		errors.RespondWithError(c, http.StatusBadRequest, "Missing required query param: report_type (monthly|yearly|range)")
		return
	}

	req := models.LeaveReportRequest{ReportType: reportType}

	// 3️ Parse date params based on report type
	switch reportType {
	case "monthly":
		monthStr := c.Query("month")
		yearStr := c.Query("year")
		if monthStr == "" || yearStr == "" {
			errors.RespondWithError(c, http.StatusBadRequest, "Monthly report requires: month, year")
			return
		}
		month, err := strconv.Atoi(monthStr)
		if err != nil || month < 1 || month > 12 {
			errors.RespondWithError(c, http.StatusBadRequest, "Invalid month. Must be between 1-12")
			return
		}
		year, err := strconv.Atoi(yearStr)
		if err != nil || year < 2000 || year > 2100 {
			errors.RespondWithError(c, http.StatusBadRequest, "Invalid year. Must be between 2000-2100")
			return
		}
		req.Month = month
		req.Year = year

	case "yearly":
		yearStr := c.Query("year")
		if yearStr == "" {
			errors.RespondWithError(c, http.StatusBadRequest, "Yearly report requires: year")
			return
		}
		year, err := strconv.Atoi(yearStr)
		if err != nil || year < 2000 || year > 2100 {
			errors.RespondWithError(c, http.StatusBadRequest, "Invalid year. Must be between 2000-2100")
			return
		}
		req.Year = year

	case "range":
		fromMonthStr := c.Query("from_month")
		fromYearStr := c.Query("from_year")
		toMonthStr := c.Query("to_month")
		toYearStr := c.Query("to_year")

		if fromMonthStr == "" || fromYearStr == "" || toMonthStr == "" || toYearStr == "" {
			errors.RespondWithError(c, http.StatusBadRequest, "Range report requires: from_month, from_year, to_month, to_year")
			return
		}

		fromMonth, err := strconv.Atoi(fromMonthStr)
		if err != nil || fromMonth < 1 || fromMonth > 12 {
			errors.RespondWithError(c, http.StatusBadRequest, "Invalid from_month. Must be between 1-12")
			return
		}
		fromYear, err := strconv.Atoi(fromYearStr)
		if err != nil || fromYear < 2000 || fromYear > 2100 {
			errors.RespondWithError(c, http.StatusBadRequest, "Invalid from_year. Must be between 2000-2100")
			return
		}
		toMonth, err := strconv.Atoi(toMonthStr)
		if err != nil || toMonth < 1 || toMonth > 12 {
			errors.RespondWithError(c, http.StatusBadRequest, "Invalid to_month. Must be between 1-12")
			return
		}
		toYear, err := strconv.Atoi(toYearStr)
		if err != nil || toYear < 2000 || toYear > 2100 {
			errors.RespondWithError(c, http.StatusBadRequest, "Invalid to_year. Must be between 2000-2100")
			return
		}

		req.FromMonth = fromMonth
		req.FromYear = fromYear
		req.ToMonth = toMonth
		req.ToYear = toYear

	default:
		errors.RespondWithError(c, http.StatusBadRequest, "Invalid report_type. Must be: monthly, yearly, or range")
		return
	}

	// 4️ Parse optional filter / sort params
	req.Search = strings.TrimSpace(c.Query("search"))
	req.Role = strings.TrimSpace(c.Query("role"))
	req.SortBy = strings.TrimSpace(c.Query("sort_by"))
	req.SortOrder = strings.TrimSpace(c.Query("sort_order"))

	// Validate role filter if provided
	if req.Role != "" {
		validRoles := map[string]bool{
			accessrole.ROLE_EMPLOYEE:    true,
			accessrole.ROLE_INTERN:      true,
			accessrole.ROLE_HR:          true,
			accessrole.ROLE_ADMIN:       true,
			accessrole.ROLE_SUPER_ADMIN: true,
			accessrole.ROLE_MANAGER:     true,
		}
		if !validRoles[strings.ToUpper(req.Role)] {
			errors.RespondWithError(c, http.StatusBadRequest, "Invalid role filter. Must be: EMPLOYEE, INTERN, HR, ADMIN, SUPERADMIN, MANAGER")
			return
		}
		req.Role = strings.ToUpper(req.Role)
	}

	// Validate sort_by if provided
	if req.SortBy != "" {
		validSortFields := map[string]bool{
			"name": true, "email": true, "role": true,
			"total_leaves": true, "paid_leaves": true,
			"unpaid_leaves": true, "early_leaves": true,
			"accrued_leaves": true, "balance_leaves": true, "used_leaves": true,
		}
		if !validSortFields[req.SortBy] {
			errors.RespondWithError(c, http.StatusBadRequest, "Invalid sort_by. Must be: name, email, role, total_leaves, paid_leaves, unpaid_leaves, early_leaves, accrued_leaves, balance_leaves, used_leaves")
			return
		}
	}

	// 5️ Call service layer
	response, err := h.LeaveReportSvc.GetLeaveReport(&req)
	if err != nil {
		fmt.Printf("GetLeaveReport Service Error: %v\n", err)
		errors.RespondWithError(c, http.StatusInternalServerError, "Failed to fetch leave report: "+err.Error())
		return
	}

	// 6️ Return success
	c.JSON(http.StatusOK, gin.H{
		"message": "Leave report fetched successfully",
		"data":    response,
	})
}
