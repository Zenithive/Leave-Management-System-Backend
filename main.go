package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/controllers"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/database"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/routes"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service/leave/leaveflow"
)

func main() {
	// Initialize configuration and database
	env := config.LoadENV()
	db := database.Connection(env)
	validator := models.InitValidator()

	repo := repositories.InitializeRepo(db)

	//leaveApproverFlow
	leaveApproverFlowRepo := repositories.NewLeaveApprovalFlowRepository(db)
	leaveApporverService := service.NewLeaveApprovalFlowService(db, leaveApproverFlowRepo)

	//leavePolicy
	leavePolicyRepo := repositories.NewLeavePolicy(db)
	leavePolicyService := service.NewLeavePolicy(db, leaveApporverService, leavePolicyRepo, repo)

	//leaveFlowLog
	leaveFlowLogRepo := repositories.NewLeaveFlowLog(db)
	leaveFlowLogService := service.NewLeaveFlowLog(db, leavePolicyService, leaveFlowLogRepo)

	//LeaveFlow
	leaveFlowRepo := repositories.NewLeaveFlow(db)
	leaveFlowService := leaveflow.NewLeaveFlow(db, leaveFlowLogService, leavePolicyService, leaveFlowRepo, leavePolicyRepo, repo)

	handlerFunc := controllers.NewHandler(env, repo, validator, leaveApporverService, leavePolicyService, leaveFlowService, leaveFlowLogService)

	// Start birthday cron job (runs daily at 00:01)
	birthdayCron := service.NewBirthdayCronService(repo, env)
	birthdayCron.Start()
	defer birthdayCron.Stop()

	// Start monthly leave accrual cron job (runs on the 1st of every month at 00:05)
	leaveAccrual := service.NewLeaveAccrualService(repo)
	leaveAccrual.Start()
	defer leaveAccrual.Stop()

	// Attach accrual service to handler so the manual trigger endpoint can use it
	handlerFunc.SetLeaveAccrualService(leaveAccrual)

	// Create a new Gin router
	r := gin.Default()
	models.InitValidator()
	routes.SetupRoutes(r, handlerFunc, env)

	fmt.Printf("Starting server on port %s\n", env.APP_PORT)

	if err := r.Run(":" + env.APP_PORT); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
