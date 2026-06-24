package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/service/leave/leaveflow"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/notification"
)

// HandlerFunc holds all dependencies injected at startup.
// handler must not create services or providers directly —
// all external dependencies come through here.
type HandlerFunc struct {
	Env                      *config.ENV
	Query                    *repositories.Repository
	Validator                *validator.Validate
	LeaveAccrual             *service.LeaveAccrualService
	LeaveReportSvc           *service.LeaveReportService
	LeaveTypeSvc             *service.LeaveTypeService
	LeaveApproverFlowService service.LeaveApprovalFlowService
	LeavePolicyService       service.LeavePolicyService
	LeaveFlowService         leaveflow.LeaveFlowService
	LeaveFlowLogService      service.LeaveFlowLog
	NotificationSvc          notification.Service // async event bus — never nil after NewHandler
	Holidayservice           service.HolidayService
}

// NewHandler constructs the handler with all required dependencies.
func NewHandler(
	env *config.ENV,
	query *repositories.Repository,
	validator *validator.Validate,
	leaveApproverFlowService service.LeaveApprovalFlowService,
	leavePolicyService service.LeavePolicyService,
	leaveFlowService leaveflow.LeaveFlowService,
	leaveFlowLogService service.LeaveFlowLog,
	notifSvc notification.Service,
	holidayservice service.HolidayService,
) *HandlerFunc {
	return &HandlerFunc{
		Env:                      env,
		Query:                    query,
		Validator:                validator,
		LeaveReportSvc:           service.NewLeaveReportService(query),
		LeaveTypeSvc:             service.NewLeaveTypeService(query),
		LeaveApproverFlowService: leaveApproverFlowService,
		LeavePolicyService:       leavePolicyService,
		LeaveFlowService:         leaveFlowService,
		LeaveFlowLogService:      leaveFlowLogService,
		NotificationSvc:          notifSvc,
		Holidayservice:           holidayservice,
	}
}

// SetLeaveAccrualService attaches the accrual service after construction.
// Called from main.go after the service is created.
func (h *HandlerFunc) SetLeaveAccrualService(svc *service.LeaveAccrualService) {
	h.LeaveAccrual = svc
}
