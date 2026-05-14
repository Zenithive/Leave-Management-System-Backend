package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/access_role"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
)

// GET /api/employee
func (h *HandlerFunc) GetEmployee(c *gin.Context) {
	role, _ := c.Get("role")
	r := role.(string)

	if err := access_role.Admin_SuperAdmin_Hr(r, "only ADMIN, SUPERADMIN, and HR can view employees"); err != nil {
		utils.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	var params models.EmployeeFilterParams
	if err := c.ShouldBindQuery(&params); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid query parameters: "+err.Error())
		return
	}

	result, err := h.EmployeeSvc.GetAllEmployees(params, r)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Employees fetched successfully",
		"employees":   result.Employees,
		"total_count": result.TotalCount,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total_pages": result.TotalPages,
		"filters": gin.H{
			"search":      params.Search,
			"roles":       params.Roles,
			"designation": params.Designation,
			"status":      params.Status,
			"manager":     params.Manager,
			"sort_by":     params.SortBy,
			"sort_order":  params.SortOrder,
		},
	})
}

// GET /api/employee/my-team
func (h *HandlerFunc) GetMyTeam(c *gin.Context) {
	currentUserID, _ := uuid.Parse(c.GetString("user_id"))
	role := c.GetString("role")

	if err := access_role.Manager(role, "only managers can access team member list"); err != nil {
		utils.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	var params models.TeamFilterParams
	if err := c.ShouldBindQuery(&params); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid query parameters: "+err.Error())
		return
	}

	employees, err := h.EmployeeSvc.GetMyTeam(currentUserID, params)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to fetch team members: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "team members fetched successfully",
		"manager_id":   currentUserID,
		"team_count":   len(employees),
		"team_members": employees,
		"sort_by":      params.SortBy,
		"sort_order":   params.SortOrder,
	})
}

// GET /api/employee/:id
func (h *HandlerFunc) GetEmployeeById(c *gin.Context) {
	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	employee, err := h.EmployeeSvc.GetEmployeeByID(empID)
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "employee not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "employee details fetched successfully",
		"employee": employee,
	})
}

// POST /api/employee
func (h *HandlerFunc) CreateEmployee(c *gin.Context) {
	role := c.GetString("role")
	if err := access_role.Admin_SuperAdmin_Hr(role, "not permitted"); err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	var input models.EmployeeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	var result *service.CreateEmployeeResult
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		var err error
		result, err = h.EmployeeSvc.CreateEmployee(tx, input, role)
		return err
	}); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	go func() {
		if err := utils.SendEmployeeCreationEmail(input.Email, input.FullName, result.GeneratedPassword); err != nil {
			fmt.Printf("Failed to send welcome email to %s: %v\n", input.Email, err)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message":  "employee created successfully",
		"password": result.GeneratedPassword,
	})
}

// PATCH /api/employee/:id/role
func (h *HandlerFunc) UpdateEmployeeRole(c *gin.Context) {
	callerRole := c.GetString("role")
	callerID, _ := uuid.Parse(c.GetString("user_id"))

	if err := access_role.Admin_SuperAdmin_Hr(callerRole, "not permitted"); err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	var input models.UpdateRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	var result *service.UpdateRoleResult
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		var err error
		result, err = h.EmployeeSvc.UpdateEmployeeRole(tx, empID, input.Role, callerRole, callerID)
		return err
	}); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "role updated successfully",
		"employee_id": result.UpdatedID,
		"old_role":    result.OldRole,
		"new_role":    input.Role,
	})
}

// PUT /api/employee/deactivate/:id
func (h *HandlerFunc) DeleteEmployeeStatus(c *gin.Context) {
	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee id")
		return
	}

	callerRole := c.GetString("role")
	if err := access_role.Admin_SuperAdmin_Hr(callerRole, "not permitted"); err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	var newStatus string
	if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
		var err error
		newStatus, err = h.EmployeeSvc.ToggleEmployeeStatus(tx, empID, callerRole)
		return err
	}); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Employee status updated successfully",
		"new_status": newStatus,
	})
}

// PATCH /api/employee/:id/manager
func (h *HandlerFunc) UpdateEmployeeManager(c *gin.Context) {
	callerRole := c.GetString("role")
	callerID, _ := uuid.Parse(c.GetString("user_id"))

	if err := access_role.Admin_SuperAdmin_Hr(callerRole, "not permitted"); err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	var input models.UpdateManagerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}
	managerID, err := uuid.Parse(input.ManagerID)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid manager ID")
		return
	}

	if err := h.EmployeeSvc.UpdateEmployeeManager(empID, managerID, callerID, callerRole); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "manager updated successfully",
		"employee_id": empID,
		"manager_id":  managerID,
	})
}

// PATCH /api/employee/:id
func (h *HandlerFunc) UpdateEmployeeInfo(c *gin.Context) {
	callerID, _ := uuid.Parse(c.GetString("user_id"))
	callerRole := c.GetString("role")

	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	var input models.UpdateEmployeeInfoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	// Use transaction only when joining_date is changing (leave balance recalculation)
	if input.JoiningDate != nil {
		if err := common.ExecuteTransaction(c, h.Query.DB, func(tx *sqlx.Tx) error {
			return h.EmployeeSvc.UpdateEmployeeInfo(tx, empID, callerID, callerRole, input)
		}); err != nil {
			utils.RespondWithError(c, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		if err := h.EmployeeSvc.UpdateEmployeeInfo(nil, empID, callerID, callerRole, input); err != nil {
			utils.RespondWithError(c, http.StatusBadRequest, err.Error())
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "employee information updated successfully",
		"employee_id": empID,
	})
}

// PATCH /api/employee/:id/password
func (h *HandlerFunc) UpdateEmployeePassword(c *gin.Context) {
	callerRole := c.GetString("role")
	callerID := c.GetString("user_id")

	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	var input models.UpdatePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: password is required and must be at least 8 characters")
		return
	}

	result, err := h.EmployeeSvc.UpdateEmployeePassword(empID, input.NewPassword, callerRole, callerID)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	go func() {
		fmt.Printf("Sending password update email to: %s\n", result.EmployeeEmail)
		if err := utils.SendPasswordUpdateEmail(
			result.EmployeeEmail,
			result.EmployeeFullName,
			input.NewPassword,
			result.UpdaterEmail,
			callerRole,
		); err != nil {
			fmt.Printf("Failed to send password update notification: %v\n", err)
		} else {
			fmt.Printf("Password update email sent successfully to: %s\n", result.EmployeeEmail)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message":     "password updated successfully",
		"employee_id": empID,
	})
}

// PATCH /api/employee/:id/designation
func (h *HandlerFunc) UpdateEmployeeDesignation(c *gin.Context) {
	callerRole := c.GetString("role")
	if err := access_role.Admin_SuperAdmin_Hr(callerRole, "only ADMIN, SUPERADMIN, and HR can assign designations"); err != nil {
		utils.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	var input models.UpdateDesignationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	designationID, err := h.EmployeeSvc.UpdateEmployeeDesignation(empID, input.DesignationID, callerRole)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	message := "employee designation updated successfully"
	if designationID == nil {
		message = "employee designation removed successfully"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        message,
		"employee_id":    empID,
		"designation_id": designationID,
	})
}

// GET /api/employee/:id/reports
func (h *HandlerFunc) GetEmployeeReports(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get employee reports"})
}

// GET /api/employee/birthdays/today
func (h *HandlerFunc) GetTodayBirthdays(c *gin.Context) {
	result, err := h.EmployeeSvc.GetTodayBirthdays()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"date":    result.Date,
		"total":   len(result.Entries),
		"data":    result.Entries,
	})
}

// GET /api/employee/birthdays/upcomming
func (h *HandlerFunc) GetBirthdays(c *gin.Context) {
	month, year := 0, 0
	fmt.Sscanf(c.Query("month"), "%d", &month)
	fmt.Sscanf(c.Query("year"), "%d", &year)

	result, err := h.EmployeeSvc.GetBirthdays(month, year)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
