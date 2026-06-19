package controllers

import (
	"github.com/go-playground/validator/v10"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service/leave/leaveflow"
)

// HandlerFunc holds dependencies
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
}

// NewHandler initializes and returns a HandlerFunc
func NewHandler(env *config.ENV, query *repositories.Repository, validator *validator.Validate, leaveApproverFlowService service.LeaveApprovalFlowService, leavePolicyService service.LeavePolicyService, leaveFlowService leaveflow.LeaveFlowService, leaveFlowLogService service.LeaveFlowLog) *HandlerFunc {
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
	}
}

// SetLeaveAccrualService attaches the accrual service after construction.
// Called from main.go after the service is created.
func (h *HandlerFunc) SetLeaveAccrualService(svc *service.LeaveAccrualService) {
	h.LeaveAccrual = svc
}
