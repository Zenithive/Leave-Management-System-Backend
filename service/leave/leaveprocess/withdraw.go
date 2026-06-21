package leaveprocess

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

// WithdrawProcessor handles the WITHDRAW action.
//
// Mirrors ApproveProcessor exactly — same stage ordering, same skip logic,
// same "all settled → finalise" pattern. Just runs in the withdrawal direction.
//
// ── How it works ─────────────────────────────────────────────────────────────
//
// Suppose after full approval the log is:
//
//   stage 1: ROLE_A=APPROVED, ROLE_B=SKIPPED
//   stage 2: ROLE_C=APPROVED, ROLE_D=SKIPPED, ROLE_E=SKIPPED
//   stage 3: ROLE_F=APPROVED
//   leave:   APPROVED
//
// ROLE_C (stage 2) initiates a withdrawal:
//
//   resetLogForWithdraw(log, stageNo=2, role="ROLE_C"):
//     stage 1: ROLE_A=SKIPPED,  ROLE_B=SKIPPED    ← ≤ stage 2 → SKIPPED
//     stage 2: ROLE_C=(acting), ROLE_D=SKIPPED, ROLE_E=SKIPPED ← same stage siblings → SKIPPED
//     stage 3: ROLE_F=WAITING                      ← > stage 2 → reset to WAITING
//
//   stampStage(ROLE_C) → WITHDRAWN
//   higherStagesSettledForWithdraw? → NO (stage 3 = WAITING) → WITHDRAWAL_PENDING
//
// ROLE_F (stage 3) must now also confirm withdrawal:
//
//   higherStagesSettledForWithdraw(stageNo=3)? → YES (nothing above 3)
//   isHighestApprovedStage? → YES → restore balance
//   stampStage(ROLE_F) → WITHDRAWN
//   allStagesSettledForWithdraw? → YES → leave = WITHDRAWN
//
// ── Rules ────────────────────────────────────────────────────────────────────
//
//  1. Caller's stage must be APPROVED (or WAITING if a higher stage already
//     reset it back to WAITING during their withdrawal).
//  2. All stages with a HIGHER stage_no must be WITHDRAWN/SKIPPED first.
//  3. On first withdrawal call: resetLogForWithdraw stamps siblings SKIPPED
//     and resets higher stages to WAITING.
//  4. If higher WAITING stages remain → WITHDRAWAL_PENDING.
//  5. When all stages settled → WITHDRAWN.
//  6. Balance restored only by isHighestApprovedStage — non-early only.

type WithdrawProcessor struct{}

func (p *WithdrawProcessor) Process(ctx context.Context, tx *sqlx.Tx, lctx *LeaveActionContext) error {

	// 1. Find caller's stage
	stage := findStage(lctx.Flow, lctx.Role)

	// 2. Reset the log:
	//    - stages ≤ acting stage → SKIPPED  (no longer part of the chain)
	//    - stages > acting stage → WAITING  (must still confirm withdrawal)
	//    The acting stage itself is handled by stampStage below.
	resetLogForWithdraw(lctx.Flow.ApprovalLog, stage.StageNo, lctx.Role)

	// 3. Stamp acting stage → WITHDRAWN
	stampStage(stage, models.WITHDRAWN, lctx.ApproverID, lctx.Remarks)

	// 4. Persist the updated approval log
	if err := lctx.FlowLogRepo.UpdateApprovalLog(ctx, tx, lctx.Leave.ID, lctx.Flow.ApprovalLog); err != nil {
		return utils.CustomErr(nil, http.StatusInternalServerError, "failed to update approval log: "+err.Error())
	}

	// 5. Not all stages settled → higher stages still need to withdraw
	if !allStagesSettledForWithdraw(lctx.Flow.ApprovalLog) {
		if err := lctx.LeaveFlowRepo.UpdateLeaveStatusTx(tx.Tx, lctx.Leave.ID, constant.LEAVE_WITHDRAWAL_PENDING, lctx.ApproverID); err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError,
				"failed to set withdrawal pending: "+err.Error())
		}
		return nil
	}

	// 8. All stages settled → this is the FINAL withdrawal
	//    Set leave status = WITHDRAWN
	if err := lctx.LeaveFlowRepo.UpdateLeaveStatusTx(tx.Tx, lctx.Leave.ID, constant.LEAVE_WITHDRAWN, lctx.ApproverID); err != nil {
		return utils.CustomErr(nil, http.StatusInternalServerError,
			"failed to withdraw leave: "+err.Error())
	}
	// 9. Restore balance ONLY on final withdrawal — non-early leave types only
	if !isEarlyLeave(lctx.LeaveType) {
		if err := lctx.CommRepo.UpdateWidthrowLeaveBalanceByEmployeeId(tx, lctx.Leave.EmployeeID, lctx.Leave.LeaveTypeID, lctx.Leave.Days); err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError,
				"failed to restore leave balance: "+err.Error())
		}
	}

	return nil
}
