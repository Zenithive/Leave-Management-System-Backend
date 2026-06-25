package leaveprocess

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/internal/repositories"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ─────────────────────────────────────────────────────────────────────────────
// LeaveActionContext
//
// Single object passed into every processor.
// Adding new fields here never changes the Process() signature — OCP.
// ─────────────────────────────────────────────────────────────────────────────

type LeaveActionContext struct {
	// Who is acting
	ApproverID uuid.UUID
	Role       string
	Remarks    string

	// Data fetched before the transaction opened
	Leave     *models.Leave
	Flow      *models.LeaveFlow
	LeaveType *models.LeaveType

	// Repos — injected by leaveFlow so processors are DB-agnostic
	FlowLogRepo   repositories.LeaveFlowLog // UpdateApprovalLog
	CommRepo      *repositories.Repository  // updarte balance
	LeaveFlowRepo repositories.LeaveFlowRepository
}

// ─────────────────────────────────────────────────────────────────────────────
// LeaveActionProcessor — one method, one context object
// ─────────────────────────────────────────────────────────────────────────────

type LeaveActionProcessor interface {
	Process(ctx context.Context, tx *sqlx.Tx, lctx *LeaveActionContext) error
}

// ─────────────────────────────────────────────────────────────────────────────
// ProcessorRegistry — maps action string → processor (no switch needed)
// ─────────────────────────────────────────────────────────────────────────────

type ProcessorRegistry struct {
	processors map[string]LeaveActionProcessor
}

// NewProcessorRegistry wires the default action set.
func NewProcessorRegistry() *ProcessorRegistry {
	r := &ProcessorRegistry{processors: make(map[string]LeaveActionProcessor)}
	r.Register("APPROVE", &ApproveProcessor{})
	r.Register("REJECT", &RejectProcessor{})
	r.Register("WITHDRAW", &WithdrawProcessor{})
	return r
}

// Register adds or replaces a processor (case-insensitive key).
func (r *ProcessorRegistry) Register(action string, p LeaveActionProcessor) {
	r.processors[strings.ToUpper(action)] = p
}

// Resolve returns the processor or an error for unknown actions.
func (r *ProcessorRegistry) Resolve(action string) (LeaveActionProcessor, error) {
	p, ok := r.processors[strings.ToUpper(action)]
	if !ok {
		return nil, errors.CustomErr(http.StatusBadRequest,
			fmt.Sprintf("unsupported leave action: %s", action))
	}
	return p, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Shared helpers — used by ApproveProcessor, RejectProcessor, WithdrawProcessor
// ─────────────────────────────────────────────────────────────────────────────

// findStage returns a pointer to the caller's stage in the approval log, or nil.
func findStage(flow *models.LeaveFlow, role string) *models.LeaveFlowStage {
	for i := range flow.ApprovalLog {
		if string(flow.ApprovalLog[i].ApproverRole) == role {
			return &flow.ApprovalLog[i]
		}
	}
	return nil
}

// skipSiblings marks every other WAITING stage at the same stage_no as SKIPPED.
// Called after one role at a stage_no approves — siblings are no longer needed.
func skipStages(log []models.LeaveFlowStage, stageNo int, actingRole string) {
	for i := range log {
		s := &log[i]

		// Skip previous waiting stages
		if s.StageNo < stageNo && s.State == models.WAITING {
			s.State = models.SKIPPED
			continue
		}

		// Skip sibling approvers in the same stage
		if s.StageNo == stageNo &&
			string(s.ApproverRole) != actingRole &&
			s.State == models.WAITING {
			s.State = models.SKIPPED
		}
	}
}
func skipAllWaitingStages(log []models.LeaveFlowStage, stageNo int, _ string) {
	for i := range log {
		s := &log[i]

		if s.State == models.WAITING &&
			(s.StageNo <= stageNo || s.StageNo >= stageNo) {
			s.State = models.SKIPPED
		}
	}
}

// allStagesSettled returns true when every stage in the log is APPROVED or SKIPPED.
// This means the final approval stage has been completed.
func allStagesSettled(log []models.LeaveFlowStage) bool {
	for _, s := range log {
		if s.State != models.APPROVED && s.State != models.SKIPPED {
			return false
		}
	}
	return true
}

// isEarlyLeave reports whether the leave type has no balance bucket.
func isEarlyLeave(lt *models.LeaveType) bool {
	return lt.IsEarly != nil && *lt.IsEarly
}

// stampStage mutates the stage in-place: sets state, actor, remarks, timestamp.
func stampStage(stage *models.LeaveFlowStage, state models.State, actorID uuid.UUID, remarks string) {
	now := time.Now()
	stage.State = state
	stage.ApprovedBy = &actorID
	stage.Remarks = remarks
	stage.ActionAt = &now
}

// ── Withdraw helpers ──────────────────────────────────────────────────────────

// resetLogForWithdraw prepares the approval log when a withdrawal starts.
// Mirrors skipStages used in approve but in the withdrawal direction:
//   - All stages with stage_no <= actingStageNo → SKIPPED  (already acted, not needed)
//   - All stages with stage_no >  actingStageNo → WAITING  (must confirm the withdrawal)
//   - The acting stage itself is stamped separately by stampStage.
func resetLogForWithdraw(log []models.LeaveFlowStage, actingStageNo int, actingRole string) {
	for i := range log {
		s := &log[i]

		// The acting stage itself — skip, handled by stampStage
		if s.StageNo == actingStageNo && string(s.ApproverRole) == actingRole {
			continue
		}
		if s.StageNo <= actingStageNo && s.State != models.WITHDRAWN {
			// Same stage siblings or lower stages — no longer needed
			s.State = models.SKIPPED
		} else if s.State != models.WITHDRAWN {
			// Higher stages — must also withdraw to complete the chain
			s.State = models.WAITING
		}
	}
}

// allStagesSettledForWithdraw returns true when every stage is WITHDRAWN or SKIPPED.
func allStagesSettledForWithdraw(log []models.LeaveFlowStage) bool {
	for _, s := range log {
		if s.State != models.WITHDRAWN && s.State != models.SKIPPED {
			return false
		}
	}
	return true
}
