package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/config/database"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/handler"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/service/leave/leaveflow"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/notification"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/notification/handlers"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/notification/providers"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/routes"
)

func main() {
	// ── Structured logger ────────────────────────────────────────────────────
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// ── Config + DB ──────────────────────────────────────────────────────────
	env := config.LoadENV()
	db := database.Connection(env)
	validator := models.InitValidator()
	repo := repositories.InitializeRepo(db)

	// ── Notification stack ───────────────────────────────────────────────────
	// 1. Email transport provider (Resend HTTP API)
	emailProvider := providers.NewResendEmailProvider(logger)

	// 2. Domain handlers — own the email body composition, no transport knowledge
	leaveHandler := handlers.NewLeaveNotificationHandler(emailProvider, logger, env)
	employeeHandler := handlers.NewEmployeeNotificationHandler(emailProvider, logger, env)

	// 3. Event processor — routes Event → correct handler (single dispatch point)
	processor := notification.NewEventProcessor(leaveHandler, employeeHandler, logger)

	// 4. Notification service — channel + worker pool (async, non-blocking)
	notifCfg := notification.DefaultConfig() // Workers:3, Buffer:256, MaxRetries:3
	notifSvc := notification.NewService(processor, notifCfg, logger)

	// Start worker pool — graceful shutdown via context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	notifSvc.Start(ctx)
	defer notifSvc.Stop()

	// ── Domain services ──────────────────────────────────────────────────────
	leaveApproverFlowRepo := repositories.NewLeaveApprovalFlowRepository(db)
	leaveApporverService := service.NewLeaveApprovalFlowService(db, leaveApproverFlowRepo)

	leavePolicyRepo := repositories.NewLeavePolicy(db)
	leavePolicyService := service.NewLeavePolicy(db, leaveApporverService, leavePolicyRepo, repo)

	leaveFlowLogRepo := repositories.NewLeaveFlowLog(db)
	leaveFlowLogService := service.NewLeaveFlowLog(db, leavePolicyService, leaveFlowLogRepo)

	leaveFlowRepo := repositories.NewLeaveFlow(db)
	leaveFlowService := leaveflow.NewLeaveFlow(
		db,
		leaveFlowLogService,
		leavePolicyService,
		leaveFlowRepo,
		leavePolicyRepo,
		leaveFlowLogRepo,
		repo,
		notifSvc, // injected — leaveflow publishes events, never touches email directly
	)
	holidayRepo := repositories.NewHolidayRepository(db)
	holidayservice := service.NewHolidayService(holidayRepo)

	// ── HTTP handler ─────────────────────────────────────────────────────────
	handlerFunc := handler.NewHandler(
		env, repo, validator,
		leaveApporverService, leavePolicyService,
		leaveFlowService, leaveFlowLogService,
		notifSvc, holidayservice,
	)

	// ── Cron jobs ────────────────────────────────────────────────────────────
	birthdayCron := service.NewBirthdayCronService(repo, env, emailProvider)
	birthdayCron.Start()
	defer birthdayCron.Stop()

	leaveAccrual := service.NewLeaveAccrualService(repo)
	leaveAccrual.Start()
	defer leaveAccrual.Stop()
	handlerFunc.SetLeaveAccrualService(leaveAccrual)

	// ── Router ───────────────────────────────────────────────────────────────
	r := gin.Default()
	models.InitValidator()
	routes.SetupRoutes(r, handlerFunc, env)

	fmt.Printf("Starting server on port %s\n", env.APP_PORT)
	if err := r.Run(":" + env.APP_PORT); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
