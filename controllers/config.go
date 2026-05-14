package controllers

import (
	"github.com/go-playground/validator/v10"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
)

// HandlerFunc holds dependencies
type HandlerFunc struct {
	Env             *config.ENV
	Query           *repositories.Repository
	Validator       *validator.Validate
	LeaveAccrual    *service.LeaveAccrualService
	LeaveReportSvc  *service.LeaveReportService
	EmployeeSvc     *service.EmployeeService
	LeaveBalanceSvc *service.LeaveBalanceService
	LeaveTimingSvc  *service.LeaveTimingService
	DesignationSvc  *service.DesignationService
	HolidaySvc      *service.HolidayService
	LeaveTypeSvc    *service.LeaveTypeService
}

// NewHandler initializes and returns a HandlerFunc
func NewHandler(env *config.ENV, query *repositories.Repository, validator *validator.Validate) *HandlerFunc {
	lbSvc := service.NewLeaveBalanceService(query)
	return &HandlerFunc{
		Env:             env,
		Query:           query,
		Validator:       validator,
		LeaveReportSvc:  service.NewLeaveReportService(query),
		EmployeeSvc:     service.NewEmployeeService(query, lbSvc),
		LeaveBalanceSvc: lbSvc,
		LeaveTimingSvc:  service.NewLeaveTimingService(query),
		DesignationSvc:  service.NewDesignationService(query),
		HolidaySvc:      service.NewHolidayService(query),
		LeaveTypeSvc:    service.NewLeaveTypeService(query, lbSvc),
	}
}

// SetLeaveAccrualService attaches the accrual service after construction.
// Called from main.go after the service is created.
func (h *HandlerFunc) SetLeaveAccrualService(svc *service.LeaveAccrualService) {
	h.LeaveAccrual = svc
}
