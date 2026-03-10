package controllers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/access_role"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

type UpdateRoleInput struct {
	Role string `json:"role" validate:"required"` // Only valid roles
}
type UpdateManagerInput struct {
	ManagerID string `json:"manager_id" validate:"required"` // UUID of new manager
}

// GetEmployees - GET /api/employees
// Query params: ?role=EMPLOYEE&designation=Senior Developer
func (h *HandlerFunc) GetEmployee(c *gin.Context) {
	role, _ := c.Get("role")
	r := role.(string)

	if err := access_role.Admin_SuperAdmin_Hr(r, "only ADMIN, SUPERADMIN, and HR can update designations"); err != nil {
		utils.RespondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	// Get filter parameters from query string
	roleFilter := c.Query("role")               // e.g., ?role=EMPLOYEE
	designationFilter := c.Query("designation") // e.g., ?designation=Senior Developer

	employees, err := h.Query.GetAllEmployees(roleFilter, designationFilter, r)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(200, gin.H{
		"message":   "Employees fetched",
		"employees": employees,
		"filters": gin.H{
			"role":        roleFilter,
			"designation": designationFilter,
		},
	})
}

// GetEmployeeById - GET /api/employee/:id
// Simple endpoint - just fetch and return employee data
func (h *HandlerFunc) GetEmployeeById(c *gin.Context) {
	// 1️ Parse Employee ID
	empIDStr := c.Param("id")
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, 400, "invalid employee ID")
		return
	}

	// 2️ Fetch employee details (EmployeeResponse has no password)
	employee, err := h.Query.GetEmployeeByID(empID)
	if err != nil {
		utils.RespondWithError(c, 404, "employee not found")
		return
	}

	// 3️ Response
	c.JSON(200, gin.H{
		"message":  "employee details fetched successfully",
		"employee": employee,
	})
}

func (h *HandlerFunc) CreateEmployee(c *gin.Context) {
	role := c.GetString("role")
	if role != constant.ROLE_SUPER_ADMIN && role != constant.ROLE_ADMIN && role != constant.ROLE_HR {
		utils.RespondWithError(c, http.StatusUnauthorized, "not permitted")
		return
	}

	var input models.EmployeeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// HR and ADMIN cannot create SUPERADMIN users
	if (role == "ADMIN" || role == "HR") && input.Role == "SUPERADMIN" {
		utils.RespondWithError(c, 403, "HR and ADMIN cannot create SUPERADMIN users")
		return
	}

	if !strings.HasSuffix(input.Email, "@zenithive.com") {
		utils.RespondWithError(c, 400, "email must end with @zenithive.com")
		return
	}

	// EMAIL EXIST CHECK
	exists, err := h.Query.CheckEmailExists(input.Email)
	if err != nil {
		utils.RespondWithError(c, 500, err.Error())
		return
	}
	if exists {
		utils.RespondWithError(c, 400, "email already exists")
		return
	}

	// GET ROLE ID
	roleID, err := h.Query.GetRoleID(input.Role)
	if err != nil {
		utils.RespondWithError(c, 400, "role not found")
		return
	}

	// GENERATE SECURE PASSWORD (combination format)
	generatedPassword, err := utils.GenerateSecurePassword()
	if err != nil {
		utils.RespondWithError(c, 500, "failed to generate secure password")
		return
	}

	// HASH PASSWORD
	hash, err := utils.HashPassword(generatedPassword)
	if err != nil {
		utils.RespondWithError(c, 500, "failed to hash password")
		return
	}

	// SET DEFAULT SALARY TO 0 IF NOT PROVIDED
	if input.Salary == nil {
		zeroSalary := 0.0
		input.Salary = &zeroSalary
	}

	// INSERT
	err = h.Query.InsertEmployee(
		input.FullName, input.Email,
		roleID, hash,
		input.Salary, input.JoiningDate,
	)
	if err != nil {
		utils.RespondWithError(c, 500, "failed to create employee")
		return
	}

	// Send welcome email with generated credentials (async to not block response)
	go func() {
		if err := utils.SendEmployeeCreationEmail(input.Email, input.FullName, generatedPassword); err != nil {
			fmt.Printf("Failed to send welcome email to %s: %v\n", input.Email, err)
		}
	}()

	c.JSON(201, gin.H{
		"message":  "employee created successfully",
		"password": generatedPassword, // Return generated password in response
	})
}
func (h *HandlerFunc) UpdateEmployeeRole(c *gin.Context) {
	// ---------------------------
	// 1️ Check permission
	// ---------------------------
	role := c.GetString("role")
	currentUserID, _ := uuid.Parse(c.GetString("user_id"))

	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, 401, "not permitted")
		return
	}

	// ---------------------------
	// 2️ Parse Employee ID
	// ---------------------------
	empIDStr := c.Param("id")
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, 400, "invalid employee ID")
		return
	}

	// ---------------------------
	// 2.5️ ADMIN and HR cannot change their own role
	// ---------------------------
	if (role == "ADMIN" || role == "HR") && currentUserID == empID {
		utils.RespondWithError(c, 403, "ADMIN and HR cannot change their own role. Only SUPERADMIN can change roles.")
		return
	}

	// ---------------------------
	// 3️ Bind input JSON
	// ---------------------------
	var input UpdateRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, 400, "invalid input: "+err.Error())
		return
	}

	// ---------------------------
	// 4️ Fetch current role & status
	// ---------------------------
	currentRole, isManager, err := h.Query.GetEmployeeCurrentRoleAndManagerStatus(empID)
	if err != nil {
		utils.RespondWithError(c, 500, "failed to fetch employee role: "+err.Error())
		return
	}

	// ---------------------------
	// 4.5️ HR and ADMIN cannot edit SUPERADMIN
	// ---------------------------
	if (role == "ADMIN" || role == "HR") && currentRole == "SUPERADMIN" {
		utils.RespondWithError(c, 403, "HR and ADMIN cannot modify SUPERADMIN users")
		return
	}

	// HR and ADMIN cannot promote to SUPERADMIN
	if (role == "ADMIN" || role == "HR") && input.Role == "SUPERADMIN" {
		utils.RespondWithError(c, 403, "HR and ADMIN cannot promote users to SUPERADMIN")
		return
	}

	// ---------------------------
	// 5️ Check if role is unchanged
	// ---------------------------
	if currentRole == input.Role {
		utils.RespondWithError(c, 400, "employee already has this role")
		return
	}

	// ---------------------------
	// 6️ Edge Case: Employee is a Manager
	// ---------------------------
	if isManager && input.Role != "MANAGER" {
		// Employee manages other employees → cannot remove MANAGER role
		utils.RespondWithError(c, 403, "cannot change role of employee who is a manager with subordinates")
		return
	}

	// ---------------------------
	// 7️ Update Role
	// ---------------------------
	updatedID, err := h.Query.UpdateEmployeeRole(empID, input.Role)
	if err != nil {
		utils.RespondWithError(c, 500, "failed to update role: "+err.Error())
		return
	}

	// ---------------------------
	// 8️ Response
	// ---------------------------
	c.JSON(200, gin.H{
		"message":     "role updated successfully",
		"employee_id": updatedID,
		"old_role":    currentRole,
		"new_role":    input.Role,
	})
}

func (h *HandlerFunc) DeleteEmployeeStatus(c *gin.Context) {

	// Read ID
	idParam := c.Param("id")
	empID, err := uuid.Parse(idParam)
	if err != nil {
		utils.RespondWithError(c, 400, "invalid employee id")
		return
	}

	// Role check
	role, _ := c.Get("role")
	r := role.(string)

	if r != "SUPERADMIN" && r != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, 401, "not permitted")
		return
	}

	// Check if target employee is SUPERADMIN
	targetEmp, err := h.Query.GetEmployeeByID(empID)
	if err != nil {
		utils.RespondWithError(c, 404, "employee not found")
		return
	}

	// HR and ADMIN cannot deactivate SUPERADMIN
	if (r == "ADMIN") && targetEmp.Role == "SUPERADMIN" {
		utils.RespondWithError(c, 403, "HR and ADMIN cannot modify SUPERADMIN users")
		return
	}

	// Toggle using same method name
	newStatus, err := h.Query.DeleteEmployeeStatus(empID)
	if err != nil {
		utils.RespondWithError(c, 500, err.Error())
		return
	}

	c.JSON(200, gin.H{
		"message":    "Employee status updated successfully",
		"new_status": newStatus,
	})
}
func (h *HandlerFunc) UpdateEmployeeManager(c *gin.Context) {
	// 1️ Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, 401, "not permitted")
		return
	}

	// 2️ Parse Employee ID
	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondWithError(c, 400, "invalid employee ID")
		return
	}

	// 2.5️ Check if target employee is SUPERADMIN
	targetEmp, err := h.Query.GetEmployeeByID(empID)
	if err != nil {
		utils.RespondWithError(c, 404, "employee not found")
		return
	}

	// HR and ADMIN cannot assign manager to SUPERADMIN
	if (role == "ADMIN" || role == "HR") && targetEmp.Role == "SUPERADMIN" {
		utils.RespondWithError(c, 403, "HR and ADMIN cannot modify SUPERADMIN users")
		return
	}

	// 3️ Parse Manager ID
	var input UpdateManagerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, 400, "invalid input: "+err.Error())
		return
	}
	managerID, err := uuid.Parse(input.ManagerID)
	if err != nil {
		utils.RespondWithError(c, 400, "invalid manager ID")
		return
	}

	// 4️ Self assignment check
	if empID == managerID {
		utils.RespondWithError(c, 400, "cannot assign employee as their own manager")
		return
	}

	// 4.5️ Prevent manager from assigning themselves to others
	currentUserID, _ := uuid.Parse(c.GetString("user_id"))
	if currentUserID == managerID && role != "SUPERADMIN" {
		utils.RespondWithError(c, 403, "you cannot assign yourself as a manager to others. Only SUPERADMIN can do this.")
		return
	}

	// 5️ Check if employee already has a manager
	// var existingManager uuid.UUID
	// err = h.Query.DB.Get(&existingManager, "SELECT manager_id FROM Tbl_Employee WHERE id=$1", empID)
	// if err != nil {
	// 	utils.RespondWithError(c, 404, "employee not found")
	// 	return
	// }

	// if existingManager != uuid.Nil {
	// 	utils.RespondWithError(c, 400, "employee already has a manager assigned")
	// 	return
	// }

	// 6️ Validate Manager exists, active and role = MANAGER
	var mgrRole, mgrStatus string
	err = h.Query.DB.Get(&mgrRole, "SELECT r.type FROM Tbl_Employee e JOIN Tbl_Role r ON e.role_id = r.id WHERE e.id=$1", managerID)
	if err != nil {
		utils.RespondWithError(c, 404, "manager not found")
		return
	}
	err = h.Query.DB.Get(&mgrStatus, "SELECT status FROM Tbl_Employee WHERE id=$1", managerID)
	if err != nil || mgrStatus != "active" {
		utils.RespondWithError(c, 403, "manager is deactivated")
		return
	}
	if mgrRole != "MANAGER" {
		utils.RespondWithError(c, 400, "assigned employee is not a manager")
		return
	}

	// 7️ Update manager
	err = h.Query.UpdateManager(empID, managerID)
	if err != nil {
		utils.RespondWithError(c, 500, "failed to update manager: "+err.Error())
		return
	}

	// 8️ Success response
	c.JSON(200, gin.H{
		"message":     "manager updated successfully",
		"employee_id": empID,
		"manager_id":  managerID,
	})
}

// UpdateEmployeeInfo - PATCH /api/employee/:id
// Anyone can update their own name
// Only SUPERADMIN and ADMIN can update email and salary
func (h *HandlerFunc) UpdateEmployeeInfo(c *gin.Context) {
	// 1️ Get current user info
	currentUserID, _ := uuid.Parse(c.GetString("user_id"))
	role := c.GetString("role")

	// 2️ Parse Employee ID
	empIDStr := c.Param("id")
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, 400, "invalid employee ID")
		return
	}

	// 3️ Check if employee exists
	existingEmp, err := h.Query.GetEmployeeByID(empID)
	if err != nil {
		utils.RespondWithError(c, 404, "employee not found")
		return
	}

	// 3.5️ HR and ADMIN cannot edit SUPERADMIN
	if (role == "ADMIN" || role == "HR") && existingEmp.Role == "SUPERADMIN" {
		utils.RespondWithError(c, 403, "HR and ADMIN cannot modify SUPERADMIN users")
		return
	}

	// 4️ Bind input JSON
	var input struct {
		FullName    *string    `json:"full_name"`
		Email       *string    `json:"email"`
		Salary      *float64   `json:"salary"`
		JoiningDate *time.Time `json:"joining_date"`
		EndingDate  *time.Time `json:"ending_date"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, 400, "invalid input: "+err.Error())
		return
	}

	// 5️ Permission checks
	isAdmin := role == "SUPERADMIN" || role == "ADMIN"
	isSelf := currentUserID == empID

	// Check if trying to update email, salary, joining_date, or ending_date
	if (input.Email != nil || input.Salary != nil || input.JoiningDate != nil || input.EndingDate != nil) && !isAdmin {
		utils.RespondWithError(c, 403, "only SUPERADMIN and ADMIN can update email, salary, joining date, and ending date")
		return
	}

	// Check if trying to update someone else's name
	if input.FullName != nil && !isSelf && !isAdmin {
		utils.RespondWithError(c, 403, "you can only update your own name")
		return
	}

	// 6️ Validate and update email if provided
	var finalEmail string
	if input.Email != nil {
		if !strings.HasSuffix(*input.Email, "@zenithive.com") {
			utils.RespondWithError(c, 400, "email must end with @zenithive.com")
			return
		}

		// Check if email is being changed and if new email already exists
		if existingEmp.Email != *input.Email {
			exists, err := h.Query.CheckEmailExists(*input.Email)
			if err != nil {
				utils.RespondWithError(c, 500, "failed to check email: "+err.Error())
				return
			}
			if exists {
				utils.RespondWithError(c, 400, "email already exists")
				return
			}
		}
		finalEmail = *input.Email
	} else {
		finalEmail = existingEmp.Email
	}

	// 7️⃣ Prepare final values
	finalName := existingEmp.FullName
	if input.FullName != nil {
		finalName = *input.FullName
	}

	finalSalary := existingEmp.Salary
	if input.Salary != nil {
		finalSalary = input.Salary
	}

	finalJoiningDate := existingEmp.JoiningDate
	if input.JoiningDate != nil {
		finalJoiningDate = input.JoiningDate
	}

	finalEndingDate := existingEmp.EndingDate
	if input.EndingDate != nil {
		finalEndingDate = input.EndingDate
	}

	// 8️ Update employee info
	err = h.Query.UpdateEmployeeInfo(empID, finalName, finalEmail, finalSalary, finalJoiningDate, finalEndingDate)
	if err != nil {
		utils.RespondWithError(c, 500, "failed to update employee: "+err.Error())
		return
	}

	// 9️ Response
	c.JSON(200, gin.H{
		"message":     "employee information updated successfully",
		"employee_id": empID,
	})
}

// GetEmployeeReports - GET /api/employees/:id/reports
func (s *HandlerFunc) GetEmployeeReports(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get employee reports"})
}

// UpdateEmployeePassword - PATCH /api/employee/:id/password
func (h *HandlerFunc) UpdateEmployeePassword(c *gin.Context) {
	// 1️ Permission check - Only SUPERADMIN, ADMIN, and HR
	role := c.GetString("role")
	// if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
	// 	utils.RespondWithError(c, 401, "not permitted to update password")
	// 	return
	// }

	// 2️ Parse Employee ID
	empIDStr := c.Param("id")
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, 400, "invalid employee ID")
		return
	}

	// 3️ Bind input JSON
	var input struct {
		NewPassword string `json:"new_password" validate:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, 400, "invalid input: password is required and must be at least 8 characters")
		return
	}

	// 4️ Validate password length
	if len(input.NewPassword) < 8 {
		utils.RespondWithError(c, 400, "password must be at least 8  characters long")
		return
	}

	// 5️ Check if employee exists
	existingEmp, err := h.Query.GetEmployeeByID(empID)
	if err != nil {
		utils.RespondWithError(c, 404, "employee not found")
		return
	}

	// 5.5️ HR and ADMIN cannot change SUPERADMIN password
	if (role == "ADMIN" || role == "HR") && existingEmp.Role == "SUPERADMIN" {
		utils.RespondWithError(c, 403, "HR and ADMIN cannot modify SUPERADMIN users")
		return
	}

	// 6️ Hash the new password
	hashedPassword, err := utils.HashPassword(input.NewPassword)
	if err != nil {
		utils.RespondWithError(c, 500, "failed to hash password: "+err.Error())
		return
	}

	// 7️ Update password in database
	err = h.Query.UpdateEmployeePassword(empID, hashedPassword)
	if err != nil {
		utils.RespondWithError(c, 500, "failed to update password: "+err.Error())
		return
	}

	// 8️ Send notification email to employee with new password
	go func() {
		var empDetails struct {
			Email    string `db:"email"`
			FullName string `db:"full_name"`
		}
		err := h.Query.DB.Get(&empDetails, "SELECT email, full_name FROM Tbl_Employee WHERE id=$1", empID)
		if err != nil {
			fmt.Printf("Failed to fetch employee details for email notification: %v\n", err)
			return
		}

		var updatedByEmail string
		currentUserID := c.GetString("user_id")
		err = h.Query.DB.Get(&updatedByEmail, "SELECT email FROM Tbl_Employee WHERE id=$1", currentUserID)
		if err != nil {
			fmt.Printf("Failed to fetch updater email for email notification: %v\n", err)
			updatedByEmail = "admin@zenithive.com" // Fallback email
		}

		fmt.Printf("Sending password update email to: %s\n", empDetails.Email)
		err = utils.SendPasswordUpdateEmail(
			empDetails.Email,
			empDetails.FullName,
			input.NewPassword,
			updatedByEmail,
			role,
		)
		if err != nil {
			fmt.Printf("Failed to send password update notification: %v\n", err)
			fmt.Printf("Error details: %+v\n", err)
		} else {
			fmt.Printf("Password update email sent successfully to: %s\n", empDetails.Email)
		}
	}()

	// 9️ Response
	c.JSON(200, gin.H{
		"message":     "password updated successfully",
		"employee_id": empID,
	})
}

// GetMyTeam - GET /api/employee/my-team
// Manager gets list of employees who report to them
func (h *HandlerFunc) GetMyTeam(c *gin.Context) {
	// 1️ Get current user info
	currentUserID, _ := uuid.Parse(c.GetString("user_id"))
	role := c.GetString("role")

	// 2️ Permission check - Only MANAGER can use this endpoint
	if role != "MANAGER" {
		utils.RespondWithError(c, 403, "only managers can access team member list")
		return
	}

	// 3️ Fetch team members
	employees, err := h.Query.GetEmployeesByManagerID(currentUserID)
	if err != nil {
		utils.RespondWithError(c, 500, "failed to fetch team members: "+err.Error())
		return
	}

	// 4️ Response
	c.JSON(200, gin.H{
		"message":      "team members fetched successfully",
		"manager_id":   currentUserID,
		"team_count":   len(employees),
		"team_members": employees,
	})
}

// UpdateEmployeeDesignation - PATCH /api/employee/:id/designation
// Only ADMIN, SUPERADMIN, and HR can assign/update employee designation
func (h *HandlerFunc) UpdateEmployeeDesignation(c *gin.Context) {
	// 1️⃣ Permission check
	role := c.GetString("role")
	if role != "SUPERADMIN" && role != "ADMIN" && role != "HR" {
		utils.RespondWithError(c, http.StatusForbidden, "only ADMIN, SUPERADMIN, and HR can assign designations")
		return
	}

	// 2️⃣ Parse Employee ID
	empIDStr := c.Param("id")
	empID, err := uuid.Parse(empIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid employee ID")
		return
	}

	// 3️⃣ Check if employee exists
	targetEmp, err := h.Query.GetEmployeeByID(empID)
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "employee not found")
		return
	}

	// 4️⃣ HR and ADMIN cannot modify SUPERADMIN
	if (role == "ADMIN" || role == "HR") && targetEmp.Role == "SUPERADMIN" {
		utils.RespondWithError(c, http.StatusForbidden, "HR and ADMIN cannot modify SUPERADMIN users")
		return
	}

	// 5️⃣ Bind input JSON
	var input struct {
		DesignationID *string `json:"designation_id"` // Can be null to remove designation
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "invalid input: "+err.Error())
		return
	}

	// 6️⃣ Parse and validate designation ID if provided
	var designationID *uuid.UUID
	if input.DesignationID != nil && *input.DesignationID != "" {
		parsedID, err := uuid.Parse(*input.DesignationID)
		if err != nil {
			utils.RespondWithError(c, http.StatusBadRequest, "invalid designation ID")
			return
		}

		// Check if designation exists
		_, err = h.Query.GetDesignationByID(parsedID)
		if err != nil {
			utils.RespondWithError(c, http.StatusNotFound, "designation not found")
			return
		}
		designationID = &parsedID
	}

	// 7️⃣ Update employee designation
	err = h.Query.UpdateEmployeeDesignation(empID, designationID)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "failed to update designation: "+err.Error())
		return
	}

	// 8️⃣ Response
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
