package leaveflow

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

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
}

type leaveFlow struct {
	DB                  *sqlx.DB
	LeaveValidationSvc  *LeaveValidationService
	Repo                repositories.LeaveFlowRepository
	CommRepo            *repositories.Repository
	LeavePolicyRepo     repositories.LeavePolicyRepository
	LeaveFlowLogService service.LeaveFlowLog
	LeavePolicyService  service.LeavePolicyService
	processor           leaveprocess.LeaveActionProcessor
}

func NewLeaveFlow(db *sqlx.DB, leaveFlowLogService service.LeaveFlowLog, leavePolicyService service.LeavePolicyService, leaveFlowRepo repositories.LeaveFlowRepository, leavePolicyRepo repositories.LeavePolicyRepository, commRepo *repositories.Repository) LeaveFlowService {
	return &leaveFlow{
		DB:                  db,
		Repo:                leaveFlowRepo,
		CommRepo:            commRepo,
		LeaveValidationSvc:  NewLeaveValidationService(commRepo),
		LeavePolicyRepo:     leavePolicyRepo,
		LeaveFlowLogService: leaveFlowLogService,
		LeavePolicyService:  leavePolicyService,
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
	if leave.Status != string(constant.LEAVE_PENDING) {
		return utils.CustomErr(nil, http.StatusBadRequest, "process only pending leave")
	}
	if leave.EmployeeID == empID {
		return utils.CustomErr(nil, http.StatusForbidden, "You cannot approve your own leave request")
	}
	leaveLogFlow, err := s.LeaveFlowLogService.GetByLeaveID(ctx, uuid.MustParse(leaveID))
	if err != nil {
		return err
	}
	// velidate
	if err := s.ActionValidator(ctx, leaveLogFlow, role, req.Action); err != nil {
		return err
	}

	leavePolicy, err := s.LeavePolicyRepo.GetById(ctx, strconv.Itoa(leave.LeaveTypeID))
	if err != nil {
		return utils.CustomErr(nil, 500, "Failed to fetch leave type: "+err.Error())
	}
	err = common.ExecuteTransaction(ctx, s.DB, func(tx *sqlx.Tx) error {

		return nil
	})

	return nil
}

func (s *leaveFlow) GetByID(ctx context.Context, leaveID string) (*models.Leave, error) {
	leave, err := s.Repo.GetByID(ctx, leaveID)
	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusBadRequest, err.Error())
	}
	return leave, err
}

//validare _logic

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

func (s *leaveFlow) ActionValidator(ctx context.Context, flow *models.LeaveFlow, role string, action string) error {

	action = strings.ToUpper(action)

	var stage *models.LeaveFlowStage

	// 1. find role
	for i := range flow.ApprovalLog {
		if string(flow.ApprovalLog[i].ApproverRole) == role {
			stage = &flow.ApprovalLog[i]
			break
		}
	}

	if stage == nil {
		return utils.CustomErr(nil, http.StatusForbidden, "role not allowed for this flow")
	}

	// 2. rules
	switch action {

	case string(models.APPROVE), string(models.REJECT):
		if stage.State != models.WAITING {
			return utils.CustomErr(nil, http.StatusBadRequest, "action allowed only in waiting state")
		}
		return nil

	case "WITHDRAW":
		if stage.State != models.APPROVED {
			return utils.CustomErr(nil, http.StatusBadRequest, "withdraw allowed only after approval")
		}
		return nil

	default:
		return utils.CustomErr(nil, http.StatusBadRequest, "invalid action")
	}
}
