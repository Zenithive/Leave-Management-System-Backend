package routes

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/controllers"
	middleware "github.com/sanjayk-eng/UserMenagmentSystem_Backend/middlewere"
)

func SetupRoutes(r *gin.Engine, h *controllers.HandlerFunc) {

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Authorization", "token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ----------------- Auth -----------------
	auth := r.Group("/api/auth")
	{
		auth.POST("/login", h.Login)
		auth.GET("/verify", h.VerifyToken)                           // Verify token validity
		auth.GET("/status", h.CheckAuthStatus)                       // Check auth status without requiring auth
		auth.POST("/logout", middleware.AuthMiddleware(h), h.Logout) // Logout (requires valid token)
	}

	// ----------------- Employees -----------------
	employees := r.Group("/api/employee")
	employees.Use(middleware.AuthMiddleware(h)) // Protect employee routes
	{
		employees.GET("/", h.GetEmployee)                                // List all employees (SUPER_ADMIN, ADMIN/HR)
		employees.GET("/my-team", h.GetMyTeam)                           // Get manager's team members (MANAGER only)
		employees.GET("/:id", h.GetEmployeeById)                         // Get employee details (Self/Manager/Admin)
		employees.POST("/", h.CreateEmployee)                            // Create employee (SUPER_ADMIN, ADMIN/HR)
		employees.PATCH("/:id", h.UpdateEmployeeInfo)                    // Update employee info (SUPER_ADMIN, ADMIN/HR)
		employees.PATCH("/:id/password", h.UpdateEmployeePassword)       // Update employee password (SUPER_ADMIN, ADMIN, HR)
		employees.PATCH("/:id/role", h.UpdateEmployeeRole)               // Change employee role (SUPER_ADMIN, ADMIN/HR)
		employees.PATCH("/:id/manager", h.UpdateEmployeeManager)         // Set/change manager (SUPER_ADMIN, ADMIN/HR)
		employees.PATCH("/:id/designation", h.UpdateEmployeeDesignation) // Assign/update designation (SUPER_ADMIN, ADMIN, HR)
		employees.PUT("/deactivate/:id", h.DeleteEmployeeStatus)         // Deactivate/Activate employee (SUPER_ADMIN, ADMIN/HR)
		employees.GET("/:id/reports", h.GetEmployeeReports)              // Get direct reports (Self/Manager/Admin)
	}

	// ----------------- Leaves -----------------
	leaves := r.Group("/api/leaves")
	leaves.Use(middleware.AuthMiddleware(h))
	{
		leaves.POST("/apply", h.ApplyLeave) // Employee applies for leave

		leaves.PUT("/edit/:id", h.EditMyLeave)                         // New Route
		leaves.POST("/admin-add/policy", h.AdminAddLeavePolicy)        // Admin creates leave policy
		leaves.PUT("/admin-update/policy/:id", h.UpdateLeavePolicy)    // Admin, SuperAdmin, HR update leave policy
		leaves.DELETE("/admin-delete/policy/:id", h.DeleteLeavePolicy) // Admin, SuperAdmin, HR delete leave policy
		leaves.GET("/Get-All-Leave-Policy", h.GetAllLeavePolicies)     // Get all leave policies
		leaves.GET("/manager/history", h.GetManagerLeaveHistory)       // Manager gets team leave history
		leaves.POST("/:id/action", h.ActionLeave)                      // Approve/Reject leave
		leaves.DELETE("/:id/cancel", h.CancelLeave)                    // Cancel pending leave (Employee/Admin)
		leaves.POST("/:id/withdraw", h.WithdrawLeave)                  // Withdraw approved leave (Admin/Manager)
		leaves.GET("/all", h.GetAllLeaves)                             // Get all leaves (filtered by role)
		leaves.GET("/my-leaves", h.GetAllMyLeave)                      // Get current user's own leaves with month/year filtering
		leaves.GET("/:id", h.GetLeaveByID)                             // Get leave by ID (role-based access)
		leaves.GET("/timming", h.GetLeaveTiming)                       // Get all Leave Timing
		leaves.PUT("/timming", h.UpdateLeaveTiming)                    // Update leave timing by super admin and admin
	}

	// ----------------- Leave Balances -----------------
	leaveBalances := r.Group("/api/leave-balances")
	leaveBalances.Use(middleware.AuthMiddleware(h))
	{

		leaveBalances.GET("/employee/:id", h.GetLeaveBalances)  // GET /api/employees/:id/leave-balances
		leaveBalances.POST("/:id/adjust", h.AdjustLeaveBalance) // POST /api/leave-balances/:id/adjust
	}

	// ----------------- Payroll -----------------
	payroll := r.Group("/api/payroll")
	payroll.Use(middleware.AuthMiddleware(h))
	{
		// Run payroll for a given month & year
		payroll.POST("/run", h.RunPayroll)
		// POST /api/payroll/run

		// Finalize payroll for a specific payroll run ID
		payroll.POST("/:id/finalize", h.FinalizePayroll)
		// POST /api/payroll/{id}/finalize

		payroll.GET("/payslip", h.GetFinalizedPayslips)

		// Download payslip PDF for a specific employee payslip ID
		payroll.GET("/payslips/:id/pdf", h.GetPayslipPDF)
		// GET /api/payroll/payslips/{id}/pdf

	}

	// ----------------- Settings -----------------
	settings := r.Group("/api/settings")
	settings.Use(middleware.AuthMiddleware(h)) // Only admin/superadmin
	{
		settings.GET("/", h.GetCompanySettings)    // Get current settings
		settings.PUT("/", h.UpdateCompanySettings) // Update settings
	}
	holidays := r.Group("/api/settings/holidays")
	holidays.Use(middleware.AuthMiddleware(h))
	{
		holidays.POST("/", h.AddHoliday)         // SUPERADMIN adds holiday
		holidays.GET("/", h.GetHolidays)         // List all holidays
		holidays.DELETE("/:id", h.DeleteHoliday) // Remove holiday
	}

	// ----------------- Designations -----------------
	designations := r.Group("/api/designations")
	designations.Use(middleware.AuthMiddleware(h))
	{
		designations.POST("/", h.CreateDesignation)      // Create designation (ADMIN, SUPERADMIN, HR)
		designations.GET("/", h.GetAllDesignations)      // Get all designations (All authenticated users)
		designations.GET("/:id", h.GetDesignationByID)   // Get designation by ID (All authenticated users)
		designations.PATCH("/:id", h.UpdateDesignation)  // Update designation (ADMIN, SUPERADMIN, HR)
		designations.DELETE("/:id", h.DeleteDesignation) // Delete designation (ADMIN, SUPERADMIN, HR)
	}
	logs := r.Group("/api/logs")
	logs.Use((middleware.AuthMiddleware(h)))
	{
		logs.GET("/", h.GetLogs) // Get logs filtered by days (SUPERADMIN only)
	}
	// Category routes
	catagory := r.Group("/api/catagory")
	catagory.Use(middleware.AuthMiddleware(h))
	{
		// ======================
		// Category CRUD
		// ======================
		catagory.POST("/", h.CreateCategory)      // Create category (ADMIN, SUPERADMIN, HR)
		catagory.GET("/", h.GetAllCategory)       // Get all categories (ADMIN, SUPERADMIN, HR)
		catagory.DELETE("/:id", h.DeleteCategory) // Delete category (ADMIN, SUPERADMIN, HR)
		catagory.PUT("/:id", h.UpdateCategory)    // Update category (ADMIN, SUPERADMIN, HR)

		// ======================
		// Equipment under category
		// ======================
		equipment := catagory.Group("/equipment")
		{
			equipment.POST("/", h.CreateEquipment)                  // Create equipment (ADMIN, SUPERADMIN, HR)
			equipment.GET("/", h.GetAllEquipment)                   // Get all equipment (ADMIN, SUPERADMIN, HR)
			equipment.GET("/by-category", h.GetEquipmentByCategory) // Get equipment by category ID (query param)
			equipment.PUT("/:id", h.UpdateEquipment)                // Update equipment (ADMIN, SUPERADMIN, HR)
			equipment.DELETE("/:id", h.DeleteEquipment)             // Delete equipment (ADMIN, SUPERADMIN, HR)
		}
		// Equipment assignment routes
		assign := equipment.Group("/assign")
		{
			assign.POST("/", h.AssignEquipment)                           // Assign equipment
			assign.GET("/", h.GetAllAssignedEquipment)                    // Get all assignments
			assign.GET("/employee/:id", h.GetAssignedEquipmentByEmployee) // Get by employee id
			assign.DELETE("/remove", h.RemoveEquipment)                   // Remove/return equipment
			assign.PUT("/update", h.UpdateAssignment)                     // Update assignment (quantity or reassign)
		}

	}
}
