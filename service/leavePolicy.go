package service

// LeaveTypeService handles all business logic for leave type (policy) management.
//
// Responsibilities:
//   - Creating a new leave type and bulk-allocating balances for all active employees
//   - Updating a leave type and recalculating affected employee balances
//   - Deleting a leave type (with referential-integrity guard)
//   - Reading all leave types
//
// This layer sits between controllers (HTTP) and repositories (DB).
// It does not know about gin.Context — it works with plain Go types and returns errors.

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"

	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/common"
)

// LeaveTypeService provides business-logic operations for leave types.

type LeavePolicyService interface {
	Create(ctx context.Context, input *models.LeaveTypeInput) (CreateLeaveTypeResult, error)
	GetByID(ctx context.Context, leaveTypeID int) (*models.LeaveTypeResponse, error)
	Get(ctx context.Context) (*[]models.LeaveTypeResponse, error)
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

// ─────────────────────────────────────────────────────────────────────────────
// CreateLeaveTypeResult is returned by CreateLeaveType.
// ─────────────────────────────────────────────────────────────────────────────
type CreateLeaveTypeResult struct {
	LeaveType *models.LeaveType
}

// CreateLeaveType creates a new leave type and allocates a balance row for every
// active employee (unless the leave type is an early-leave type).
//
// Business rules:
//  1. is_paid, default_entitlement, leave_count, and is_early all have sensible defaults.
//  2. leave_count must be > 0.
//  3. Early leave types (is_early = true) do NOT get a balance bucket — they are
//     tracked per-application, one per employee per month.
//  4. For non-early types, every active employee gets an opening balance row:
//     - INTERN employees → intern_entitlement if set, else default_entitlement
//     - Other employees  → default_entitlement
//     - Employees who joined in the current year get a prorated amount.
//  5. The entire operation runs inside a single caller-provided transaction so
//     the controller can attach logging inside the same transaction boundary.

func (s *LeavePolicy) Create(ctx context.Context, input *models.LeaveTypeInput) (CreateLeaveTypeResult, error) {

	// 1. Normalize input
	if err := s.NormalizeLeaveTypeInput(input); err != nil {
		return CreateLeaveTypeResult{}, err
	}

	var result CreateLeaveTypeResult
	// 2. Transaction wrapper
	err := common.ExecuteTransaction(ctx, s.DB, func(tx *sqlx.Tx) error {

		// 3. Insert leave type
		leaveType, err := s.LeavePolicyRepo.Create(ctx, tx, input)
		if err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError, "failed to create leave type")
		}

		// Populate display fields not returned by RETURNING clause
		leaveType.Name = input.Name
		leaveType.IsPaid = *input.IsPaid
		leaveType.DefaultEntitlement = *input.DefaultEntitlement
		leaveType.InternEntitlement = input.InternEntitlement

		// 4. Bulk allocation (inside transaction)
		if !*input.IsEarly {

			activeEmployees, err := s.CommRepo.GetAllActiveEmployeesWithRoles(tx)
			if err != nil {
				return utils.CustomErr(nil, http.StatusInternalServerError, "failed to fetch active employees")
			}

			err = s.CommRepo.BulkAllocateLeaveBalanceForNewLeaveType(tx, leaveType.ID, *input.DefaultEntitlement, input.InternEntitlement, activeEmployees)
			if err != nil {
				return utils.CustomErr(nil, http.StatusInternalServerError, "failed to allocate leave balances")
			}
		}

		result = CreateLeaveTypeResult{
			LeaveType: leaveType,
		}

		return nil
	})

	if err != nil {
		return CreateLeaveTypeResult{}, err
	}

	return result, nil
}

func (s *LeavePolicy) GetByID(ctx context.Context, leaveTypeID int) (*models.LeaveTypeResponse, error) {

	leaveType, err := s.LeavePolicyRepo.GetById(ctx, strconv.Itoa(leaveTypeID))
	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "failed to get leave policy")
	}
	leaveApproverFlow, err := s.LeaveApporverService.GetLeaveApprovalFlowById(ctx, strconv.Itoa(leaveTypeID))
	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "failed to load Leave Approvel flow")
	}

	res := models.MappPayload(leaveType, leaveApproverFlow)

	return res, err
}

func (s *LeavePolicy) Get(ctx context.Context) (*[]models.LeaveTypeResponse, error) {

	leaveType, err := s.LeavePolicyRepo.Get(ctx)
	if err != nil {
		fmt.Println("================", err.Error())
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

// ─────────────────────────────────────────────────────────────────────────────
// UpdateLeaveTypeResult is returned by UpdateLeaveType.
// ─────────────────────────────────────────────────────────────────────────────
type UpdateLeaveTypeResult struct {
	LeaveTypeID int
}

// UpdateLeaveType updates an existing leave type and recalculates employee balances
// when entitlement values change.
//
// Business rules:
//  1. Roles allowed to update: SUPERADMIN, ADMIN, HR (enforced by caller / middleware).
//  2. default_entitlement cannot be negative.
//  3. When default_entitlement changes for a non-early leave type, all non-INTERN employees'
//     opening and closing balances are recalculated.
//  4. When intern_entitlement changes (or is cleared), all INTERN employees' balances
//     are recalculated using the resolved new effective entitlement.
//  5. The entire operation runs inside a single caller-provided transaction.
func (s *LeaveTypeService) UpdateLeaveType(tx *sqlx.Tx, leaveTypeID int, input models.LeaveTypeInput) (UpdateLeaveTypeResult, error) {

	// ── Apply defaults ────────────────────────────────────────────────────────
	if input.IsPaid == nil {
		f := false
		input.IsPaid = &f
	}
	if input.DefaultEntitlement == nil {
		zero := 0
		input.DefaultEntitlement = &zero
	}
	if input.IsWorkFromHome == nil {
		f := false
		input.IsWorkFromHome = &f
	}

	// ── Validate ─────────────────────────────────────────────────────────────
	if *input.DefaultEntitlement < 0 {
		return UpdateLeaveTypeResult{}, fmt.Errorf("default entitlement cannot be negative")
	}

	// ── Fetch current leave type ──────────────────────────────────────────────
	oldLeaveType, err := s.repo.GetLeaveTypeByIdTx(tx, leaveTypeID)
	if err == sql.ErrNoRows {
		return UpdateLeaveTypeResult{}, fmt.Errorf("leave type not found")
	}
	if err != nil {
		return UpdateLeaveTypeResult{}, fmt.Errorf("failed to fetch leave type: %w", err)
	}

	// ── Persist the leave type changes ────────────────────────────────────────
	if err := s.repo.UpdateLeaveType(tx, leaveTypeID, input); err != nil {
		return UpdateLeaveTypeResult{}, fmt.Errorf("failed to update leave type: %w", err)
	}

	currentYear := time.Now().Year()
	oldDefaultEntitlement := oldLeaveType.DefaultEntitlement
	newDefaultEntitlement := *input.DefaultEntitlement

	// ── Recalculate non-INTERN balances when default_entitlement changed ───────
	// Skip for early leave types — they have no balance bucket.
	isEarly := oldLeaveType.IsEarly != nil && *oldLeaveType.IsEarly
	if !isEarly {
		if err := s.repo.UpdateLeaveBalancesForEntitlementChange(
			tx, leaveTypeID, oldDefaultEntitlement, newDefaultEntitlement, currentYear,
		); err != nil {
			return UpdateLeaveTypeResult{}, fmt.Errorf("failed to update leave balances: %w", err)
		}
	}

	// ── Recalculate INTERN balances ───────────────────────────────────────────
	// Resolve the effective intern entitlement:
	//   - If intern_entitlement is provided in the request → use it
	//   - If intern_entitlement was cleared (nil) → fall back to newDefaultEntitlement
	newEffectiveIntern := newDefaultEntitlement
	if input.InternEntitlement != nil {
		newEffectiveIntern = *input.InternEntitlement
	}
	if err := s.repo.UpdateInternLeaveBalancesForEntitlementChange(
		tx, leaveTypeID, newEffectiveIntern, currentYear,
	); err != nil {
		return UpdateLeaveTypeResult{}, fmt.Errorf("failed to update intern leave balances: %w", err)
	}

	return UpdateLeaveTypeResult{LeaveTypeID: leaveTypeID}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// DeleteLeaveType removes a leave type.
// ─────────────────────────────────────────────────────────────────────────────

// DeleteLeaveType deletes a leave type if it is not referenced by any leave application.
//
// Business rules:
//  1. If the leave type is used in any existing leave application → return a descriptive error.
//  2. The operation runs inside a caller-provided transaction.
func (s *LeaveTypeService) DeleteLeaveType(tx *sqlx.Tx, leaveTypeID int) error {

	// ── Verify existence ──────────────────────────────────────────────────────
	_, err := s.repo.GetLeaveTypeByIdTx(tx, leaveTypeID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("leave type not found")
	}
	if err != nil {
		return fmt.Errorf("failed to fetch leave type: %w", err)
	}

	// ── Delete (repo returns sql.ErrNoRows when the type is in use) ───────────
	if err := s.repo.DeleteLeaveType(tx, leaveTypeID); err == sql.ErrNoRows {
		return fmt.Errorf("cannot delete leave type: it is being used in existing leave applications")
	} else if err != nil {
		return fmt.Errorf("failed to delete leave type: %w", err)
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// GetAllLeaveTypes returns all leave type definitions.
// ─────────────────────────────────────────────────────────────────────────────

// GetAllLeaveTypes returns all leave types ordered by id.
func (s *LeaveTypeService) GetAllLeaveTypes() ([]models.LeaveType, error) {
	leaveTypes, err := s.repo.GetAllLeaveType()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leave types: %w", err)
	}
	return leaveTypes, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// LeaveTypeLogMeta is used by the controller to record an audit log entry
// inside the same transaction, after the service call succeeds.
// ─────────────────────────────────────────────────────────────────────────────

// LeaveTypeLogMeta carries the context needed to write an audit log entry.
type LeaveTypeLogMeta struct {
	Action     string    // constant.ActionCreate / ActionUpdate / ActionDelete
	FromUserID uuid.UUID // employee who performed the operation
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
	if *input.DefaultEntitlement < 0 {
		return fmt.Errorf("default entitlement cannot be negative")
	}
	return nil
}
