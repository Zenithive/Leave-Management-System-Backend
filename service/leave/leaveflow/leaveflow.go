package leaveflow

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service/leave/leaveprocess"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
)

type LeaveFlowService interface {
	Create(ctx context.Context, leave *models.LeaveInput, role string) error
	GetByID(ctx context.Context, leaveID string) (*models.Leave, error)
	ActionLeave(ctx context.Context, req models.ActionLeaveReq, leaveID string, empID uuid.UUID, role string) error
	GetLeaves(ctx context.Context, empID uuid.UUID, role string, month int, year int) (gin.H, error)
	GetMyLeave(empID uuid.UUID, month int, year int) (gin.H, error)
	CancleLeave(c context.Context, leaveId string) (string, error)
}

type leaveFlow struct {
	DB                  *sqlx.DB
	LeaveValidationSvc  *LeaveValidationService
	Repo                repositories.LeaveFlowRepository
	CommRepo            *repositories.Repository
	LeavePolicyRepo     repositories.LeavePolicyRepository
	LeaveFlowLogRepo    repositories.LeaveFlowLog
	LeaveFlowLogService service.LeaveFlowLog
	LeavePolicyService  service.LeavePolicyService
	registry            *leaveprocess.ProcessorRegistry
}

func NewLeaveFlow(db *sqlx.DB, leaveFlowLogService service.LeaveFlowLog, leavePolicyService service.LeavePolicyService, leaveFlowRepo repositories.LeaveFlowRepository, leavePolicyRepo repositories.LeavePolicyRepository, leaveFlowLogRepo repositories.LeaveFlowLog, commRepo *repositories.Repository) LeaveFlowService {
	return &leaveFlow{
		DB:                  db,
		Repo:                leaveFlowRepo,
		CommRepo:            commRepo,
		LeaveValidationSvc:  NewLeaveValidationService(commRepo),
		LeavePolicyRepo:     leavePolicyRepo,
		LeaveFlowLogRepo:    leaveFlowLogRepo,
		LeaveFlowLogService: leaveFlowLogService,
		LeavePolicyService:  leavePolicyService,
		registry:            leaveprocess.NewProcessorRegistry(),
	}
}

func (s *leaveFlow) Create(ctx context.Context, leave *models.LeaveInput, role string) error {
	LeaveTypeInfo, leaveTiming, err := s.ValidateLeave(ctx, leave)
	if err != nil {
		return err
	}
	leaveTypeRres, err := s.LeavePolicyService.GetByID(ctx, leave.LeaveTypeID)
	if err != nil {
		return err
	}
	var Days float64
	err = common.ExecuteTransaction(ctx, s.DB, func(tx *sqlx.Tx) error {
		// Calculate working days with timing consideration
		leaveDays, err := service.CalculateWorkingDaysWithTiming(s.CommRepo, tx, leave.StartDate, leave.EndDate, LeaveTypeInfo.TimingID, leaveTiming)
		if err != nil {
			return utils.CustomErr(nil, http.StatusBadRequest, err.Error())
		}
		if leaveDays <= 0 {
			return utils.CustomErr(nil, http.StatusBadRequest, "Calculated leave days must be greater than zero. Please check the dates and timing")
		}
		leave.Days = &Days
		Days = leaveDays

		// Validate unpaid leave application: Cannot apply for unpaid leave if paid balance > 0
		if err := service.ValidateUnpaidLeaveApplication(s.CommRepo, tx, leave.EmployeeID, leave.LeaveTypeID); err != nil {
			return utils.CustomErr(nil, http.StatusBadRequest, err.Error())
		}

		// Comprehensive leave validation (balance, early leave limit, overlapping)
		validationParams := ValidateLeaveApplicationParams{
			EmployeeID:     leave.EmployeeID,
			LeaveTypeID:    leave.LeaveTypeID,
			StartDate:      leave.StartDate,
			EndDate:        leave.EndDate,
			LeaveDays:      leaveDays,
			ExcludeLeaveID: nil,
		}
		if err := s.LeaveValidationSvc.ValidateLeaveApplication(tx, validationParams, LeaveTypeInfo.LeaveType); err != nil {
			return utils.CustomErr(nil, http.StatusBadRequest, err.Error())
		}

		// Insert Leave
		// For IsEarly leave types, pass the leave_timing string
		var leaveTimingStr *string
		if LeaveTypeInfo.LeaveType.IsEarly != nil && *LeaveTypeInfo.LeaveType.IsEarly && leave.LeaveTiming != nil {
			leaveTimingStr = leave.LeaveTiming
		}
		id, err := s.Repo.InsertLeave(tx, leave, leaveTimingStr)
		if err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError, "Failed to apply leave: "+err.Error())
		}
		if err := s.LeaveFlowLogService.Create(ctx, tx, id, leaveTypeRres, role); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (s *leaveFlow) ActionLeave(ctx context.Context, req models.ActionLeaveReq, leaveID string, empID uuid.UUID, role string) error {
	leave, err := s.GetByID(ctx, leaveID)
	if err != nil {
		return err
	}
	if leave.EmployeeID == empID {
		return utils.CustomErr(nil, http.StatusForbidden, "You cannot process your own leave request")
	}
	leaveLogFlow, err := s.LeaveFlowLogService.GetByLeaveID(ctx, uuid.MustParse(leaveID))
	if err != nil {
		return err
	}
	// velidate
	if err := s.ActionValidator(ctx, leaveLogFlow, role, req.Action, leave.Status); err != nil {
		return err
	}

	leavePolicy, err := s.LeavePolicyRepo.GetById(ctx, strconv.Itoa(leave.LeaveTypeID))
	if err != nil {
		return utils.CustomErr(nil, 500, "Failed to fetch leave type: "+err.Error())
	}

	// Resolve the processor for the requested action via the registry
	processor, err := s.registry.Resolve(strings.ToUpper(req.Action))
	if err != nil {
		return err
	}

	// Build the context object — single place where all data is assembled
	lctx := &leaveprocess.LeaveActionContext{
		ApproverID:    empID,
		Role:          role,
		Remarks:       req.Remarks,
		Leave:         leave,
		Flow:          leaveLogFlow,
		LeaveType:     leavePolicy,
		FlowLogRepo:   s.LeaveFlowLogRepo,
		CommRepo:      s.CommRepo,
		LeaveFlowRepo: s.Repo,
	}

	return common.ExecuteTransaction(ctx, s.DB, func(tx *sqlx.Tx) error {
		return processor.Process(ctx, tx, lctx)
	})
}

func (s *leaveFlow) GetByID(ctx context.Context, leaveID string) (*models.Leave, error) {
	leave, err := s.Repo.GetByID(ctx, leaveID)
	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusBadRequest, err.Error())
	}
	return leave, err
}

func (s *leaveFlow) GetLeaves(ctx context.Context, empID uuid.UUID, role string, month int, year int) (gin.H, error) {

	var (
		result []models.LeaveResponse
		err    error
	)
	switch role {

	case constant.ROLE_EMPLOYEE, constant.ROLE_INTERN:
		result, err = s.Repo.GetAllEmployeeLeaveByMonthYear(empID, month, year)
	case constant.ROLE_MANAGER:

		result, err = s.Repo.GetAllleavebaseonassignManagerByMonthYear(empID, month, year)

	case constant.ROLE_ADMIN, constant.ROLE_HR, constant.ROLE_SUPER_ADMIN:
		result, err = s.Repo.GetAllLeaveByMonthYear(month, year)

	default:
		return nil, utils.CustomErr(nil, http.StatusForbidden, "invalid role")
	}

	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "failed to fetch leaves")
	}

	if result == nil {
		result = []models.LeaveResponse{}
	}
	for i := range result {
		flow, err := s.LeaveFlowLogService.GetByLeaveID(ctx, uuid.MustParse(result[i].ID))
		if err != nil {
			return nil, err
		}

		if flow != nil {
			result[i].ApprovalLog = flow.ApprovalLog
		}
	}

	return gin.H{
		"message": "Leaves fetched successfully",
		"role":    role,
		"month":   month,
		"year":    year,
		"summary": models.BuildLeaveCountSummary(result),
		"data":    result,
	}, nil
}

//validare _logic

func (s *leaveFlow) GetMyLeave(empID uuid.UUID, month int, year int) (gin.H, error) {

	result, err := s.Repo.GetMyLeavesByMonthYear(empID, month, year)

	if err != nil {
		fmt.Printf("GetAllMyLeave DB Error: %v\n", err)
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "Failed to fetch my leaves: "+err.Error())
	}

	if result == nil {
		result = []models.LeaveResponse{}
	}

	return gin.H{
		"message": "My leaves fetched successfully",
		"user_id": empID,
		"month":   month,
		"year":    year,
		"summary": models.BuildLeaveCountSummary(result),
		"data":    result,
	}, nil
}

func (s *leaveFlow) CancleLeave(ctx context.Context, leaveId string) (string, error) {

	leave, err := s.Repo.GetByID(ctx, leaveId)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", utils.CustomErr(nil, http.StatusNotFound, "Leave not found")
		}
		return "", utils.CustomErr(nil, http.StatusInternalServerError, "Failed to fetch leave: "+err.Error())
	}
	// Status validation using switch
	switch leave.Status {

	case constant.LEAVE_APPLOVED:
		return "", utils.CustomErr(nil, http.StatusBadRequest, "Cannot cancel approved leave. Please contact your manager or admin")

	case constant.LEAVE_REJECTED:
		return "", utils.CustomErr(nil, http.StatusBadRequest, "Leave is already rejected")

	case constant.LEAVE_CANCELLED:
		return "", utils.CustomErr(nil, http.StatusBadRequest, "Leave is already cancelled")
	}
	if err := s.Repo.UpdateLeaveStatus(leaveId, constant.LEAVE_CANCELLED); err != nil {
		return "", utils.CustomErr(nil, http.StatusInternalServerError, "Failed to cancel leave: "+err.Error())
	}
	return leaveId, nil
}

func (s *leaveFlow) ValidateLeave(ctx context.Context, leave *models.LeaveInput) (*LeaveTypeInfo, time.Time, error) {

	var (
		leaveTiming time.Time
		err         error
	)

	// Validate leave timing value if provided
	if leave.LeaveTiming != nil {
		leaveTiming, err = service.ValidateLeaveTiming(*leave.LeaveTiming)
		if err != nil {
			return nil, time.Time{}, utils.CustomErr(nil, http.StatusBadRequest, err.Error())
		}
	}

	// Validate leave timing ID
	if err := s.LeaveValidationSvc.ValidateLeaveTimingID(leave.LeaveTimingID); err != nil {
		return nil, time.Time{}, utils.CustomErr(nil, http.StatusBadRequest, err.Error())
	}

	// Validate reason
	if err := s.LeaveValidationSvc.ValidateLeaveReason(leave.Reason); err != nil {
		return nil, time.Time{}, utils.CustomErr(nil, http.StatusBadRequest, err.Error())
	}

	// Validate start and end dates
	if err := s.LeaveValidationSvc.ValidateLeaveDates(leave.StartDate, leave.EndDate); err != nil {
		return nil, time.Time{}, utils.CustomErr(nil, http.StatusBadRequest, err.Error())
	}

	// Get leave type and resolve timing
	leaveTypeInfo, err := s.LeaveValidationSvc.GetLeaveTypeAndResolveTimingID(leave.LeaveTypeID, leave.LeaveTimingID)
	if err != nil {
		return nil, time.Time{}, utils.CustomErr(nil, http.StatusBadRequest, err.Error())
	}

	return leaveTypeInfo, leaveTiming, nil
}

func (s *leaveFlow) ActionValidator(ctx context.Context, flow *models.LeaveFlow, role string, action string, status string) error {

	action = strings.ToUpper(action)

	var stage *models.LeaveFlowStage

	// find this role's stage
	for i := range flow.ApprovalLog {
		if string(flow.ApprovalLog[i].ApproverRole) == role {
			stage = &flow.ApprovalLog[i]
			break
		}
	}

	if stage == nil {
		return utils.CustomErr(nil, http.StatusForbidden, "role not allowed for this flow")
	}

	switch action {

	case string(models.APPROVE):
		// APPROVE requires ordered processing — stage must be WAITING
		if status != string(constant.LEAVE_PENDING) {
			return utils.CustomErr(nil, http.StatusBadRequest, "process only pending leave")
		}
		if stage.State != models.WAITING {
			if status != string(constant.LEAVE_PENDING) {
				return utils.CustomErr(nil, http.StatusBadRequest, "process only pending leave")
			}
			return utils.CustomErr(nil, http.StatusBadRequest, "approve allowed only in waiting state")
		}
		return nil

	case string(models.REJECT):
		// REJECT is a single final action — only check that stage is WAITING,
		// no ordering constraint applies
		if stage.State != models.WAITING {
			return utils.CustomErr(nil, http.StatusBadRequest, "reject allowed only in waiting state")
		}
		return nil

	case "WITHDRAW":
		// Stage must be APPROVED (original approver) or WAITING
		// (reset to WAITING by a lower-stage withdrawal that needs higher confirmation)
		if status != string(constant.LEAVE_APPLOVED) && status != string(constant.LEAVE_WITHDRAWAL_PENDING) {
			return utils.CustomErr(nil, http.StatusBadRequest, "withdraw allowed only after approval")
		}
		return nil
	default:
		return utils.CustomErr(nil, http.StatusBadRequest, "invalid action")
	}
}
