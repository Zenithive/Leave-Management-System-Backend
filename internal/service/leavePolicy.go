package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/common"
)

// LeaveTypeService provides business-logic operations for leave types.

type LeavePolicyService interface {
	Create(ctx context.Context, input *models.LeaveTypeInput) (*models.LeaveType, error)
	GetByID(ctx context.Context, leaveTypeID int) (*models.LeaveTypeResponse, error)
	Get(ctx context.Context) (*[]models.LeaveTypeResponse, error)
	Update(ctx context.Context, leaveTypeID int, input *models.LeaveTypeInput) (*models.LeaveType, error)
	Delete(ctx context.Context, leaveTypeID int) error
}

type LeavePolicy struct {
	DB                   *sqlx.DB
	LeaveApporverService LeaveApprovalFlowService
	LeavePolicyRepo      repositories.LeavePolicyRepository
	CommRepo             *repositories.Repository
}

func NewLeavePolicy(db *sqlx.DB, leaveApporverService LeaveApprovalFlowService, leavePolicyRepo repositories.LeavePolicyRepository, commRepo *repositories.Repository) LeavePolicyService {
	return &LeavePolicy{
		DB:                   db,
		LeaveApporverService: leaveApporverService,
		LeavePolicyRepo:      leavePolicyRepo,
		CommRepo:             commRepo,
	}
}

type LeaveTypeService struct {
	repo *repositories.Repository
}

// NewLeaveTypeService creates a new LeaveTypeService.
func NewLeaveTypeService(repo *repositories.Repository) *LeaveTypeService {
	return &LeaveTypeService{repo: repo}
}

func (s *LeavePolicy) Create(ctx context.Context, input *models.LeaveTypeInput) (*models.LeaveType, error) {

	var res *models.LeaveType
	var err error

	// 1. Normalize input
	if err := s.NormalizeLeaveTypeInput(input); err != nil {
		return nil, err
	}

	// 2. Transaction wrapper
	err = common.ExecuteTransaction(ctx, s.DB, func(tx *sqlx.Tx) error {

		// 3. Insert leave type
		res, err = s.LeavePolicyRepo.Create(ctx, tx, input)
		if err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError, "failed to create leave type")
		}

		// Populate display fields not returned by RETURNING clause
		res.Name = input.Name
		res.IsPaid = *input.IsPaid
		res.DefaultEntitlement = *input.DefaultEntitlement
		res.InternEntitlement = input.InternEntitlement

		// 4. Bulk allocation (inside transaction)
		if !*input.IsEarly {

			activeEmployees, err := s.CommRepo.GetAllActiveEmployeesWithRoles(tx)
			if err != nil {
				return utils.CustomErr(nil, http.StatusInternalServerError, "failed to fetch active employees")
			}

			err = s.CommRepo.BulkAllocateLeaveBalanceForNewLeaveType(tx, res.ID, *input.DefaultEntitlement, input.InternEntitlement, activeEmployees)
			if err != nil {
				return utils.CustomErr(nil, http.StatusInternalServerError, "failed to allocate leave balances")
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *LeavePolicy) GetByID(ctx context.Context, leaveTypeID int) (*models.LeaveTypeResponse, error) {

	leaveType, err := s.LeavePolicyRepo.GetById(ctx, strconv.Itoa(leaveTypeID))
	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "failed to get leave policy")
	}

	// No approval flow assigned — return leave type with nil flow
	if leaveType.ApprovalFlowID == nil || *leaveType.ApprovalFlowID == "" {
		return models.MappPayload(leaveType, nil), nil
	}

	// Fetch the approval flow using its own UUID (not the leave type ID)
	leaveApproverFlow, err := s.LeaveApporverService.GetLeaveApprovalFlowById(ctx, *leaveType.ApprovalFlowID)
	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "failed to load Leave Approvel flow")
	}

	return models.MappPayload(leaveType, leaveApproverFlow), nil
}

func (s *LeavePolicy) Get(ctx context.Context) (*[]models.LeaveTypeResponse, error) {

	leaveType, err := s.LeavePolicyRepo.Get(ctx)
	if err != nil {

		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "failed to get leave policy")
	}
	leaveApproverFlow, err := s.LeaveApporverService.GetAllLeaveApprovalFlows(ctx)
	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "failed to load Leave Approvel flow")
	}

	// Build a lookup map for O(1) flow matching by ID
	flowMap := make(map[string]models.LeaveApprovalFlowResponse, len(leaveApproverFlow))
	for _, f := range leaveApproverFlow {
		flowMap[f.ID] = f
	}

	var res []models.LeaveTypeResponse

	for _, l := range *leaveType {
		lCopy := l // avoid loop-variable aliasing
		if lCopy.ApprovalFlowID != nil && *lCopy.ApprovalFlowID != "" {
			if flow, ok := flowMap[*lCopy.ApprovalFlowID]; ok {
				res = append(res, *models.MappPayload(&lCopy, &flow))
				continue
			}
		}
		// Leave type has no approval flow — still include it with a nil flow
		res = append(res, models.LeaveTypeResponse{
			ID:                 lCopy.ID,
			Name:               lCopy.Name,
			IsPaid:             lCopy.IsPaid,
			DefaultEntitlement: lCopy.DefaultEntitlement,
			InternEntitlement:  lCopy.InternEntitlement,
			IsEarly:            lCopy.IsEarly,
			IsWorkFromHome:     lCopy.IsWorkFromHome,
			ApprovalFlowID:     lCopy.ApprovalFlowID,
			CreatedAt:          lCopy.CreatedAt,
			UpdatedAt:          lCopy.UpdatedAt,
			ApprovalFlow:       nil,
		})
	}
	return &res, err
}

func (s *LeavePolicy) Update(ctx context.Context, leaveTypeID int, input *models.LeaveTypeInput) (*models.LeaveType, error) {
	// 1. Normalize input
	if err := s.NormalizeLeaveTypeInput(input); err != nil {
		return nil, err
	}
	var res *models.LeaveType
	oldLeaveType, err := s.LeavePolicyRepo.GetById(ctx, strconv.Itoa(leaveTypeID))
	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusBadRequest, err.Error())
	}
	err = common.ExecuteTransaction(ctx, s.DB, func(tx *sqlx.Tx) error {
		res, err = s.LeavePolicyRepo.Update(ctx, tx, strconv.Itoa(leaveTypeID), input)
		if err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError, "failed to update leave policy")
		}

		currentYear := time.Now().Year()
		oldDefaultEntitlement := oldLeaveType.DefaultEntitlement
		newDefaultEntitlement := *input.DefaultEntitlement

		isEarly := oldLeaveType.IsEarly != nil && *oldLeaveType.IsEarly
		if !isEarly {
			if err := s.CommRepo.UpdateLeaveBalancesForEntitlementChange(
				tx, leaveTypeID, oldDefaultEntitlement, newDefaultEntitlement, currentYear,
			); err != nil {
				return utils.CustomErr(nil, http.StatusInternalServerError, "failed to update leave Balance")
			}
		}

		newEffectiveIntern := newDefaultEntitlement
		if input.InternEntitlement != nil {
			newEffectiveIntern = *input.InternEntitlement
		}
		if err := s.CommRepo.UpdateInternLeaveBalancesForEntitlementChange(
			tx, leaveTypeID, newEffectiveIntern, currentYear,
		); err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError, "failed to update intern leave balances")
		}
		return nil
	})

	return res, err
}

func (s *LeavePolicy) Delete(ctx context.Context, leaveTypeID int) error {
	err := common.ExecuteTransaction(ctx, s.DB, func(tx *sqlx.Tx) error {
		if err := s.LeavePolicyRepo.Delete(tx, leaveTypeID); err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError, err.Error())
		}
		return nil
	})
	return err
}

func (s *LeavePolicy) NormalizeLeaveTypeInput(input *models.LeaveTypeInput) error {
	if input.IsPaid == nil {
		v := false
		input.IsPaid = &v
	}

	if input.DefaultEntitlement == nil {
		v := 0
		input.DefaultEntitlement = &v
	}

	if input.LeaveCount == nil {
		v := 2
		input.LeaveCount = &v
	}

	if input.IsEarly == nil {
		v := false
		input.IsEarly = &v
	}

	if input.IsWorkFromHome == nil {
		v := false
		input.IsWorkFromHome = &v
	}
	if input.ApprovalFlowID != nil && *input.ApprovalFlowID == "" {
		input.ApprovalFlowID = nil
	}
	if *input.DefaultEntitlement < 0 {
		return fmt.Errorf("default entitlement cannot be negative")
	}

	return nil
}
