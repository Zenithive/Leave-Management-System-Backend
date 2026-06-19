package leaveprocess

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils/constant"
)

type WithdrawProcessor struct{}

func (p *WithdrawProcessor) Process(ctx context.Context, tx *sqlx.Tx, lctx *LeaveActionContext) error {

	// 1. Find caller's stage
	stage := findStage(lctx.Flow, lctx.Role)

	// 2. Stamp this stage → WITHDRAWN
	stampStage(stage, models.WITHDRAWN, lctx.ApproverID, lctx.Remarks)

	// 5. Skip any sibling APPROVED stages at the same stage_no (safety, mirrors approve)
	skipSiblingsForWithdraw(lctx.Flow.ApprovalLog, stage.StageNo, lctx.Role)

	// 6. Persist the updated approval log
	if err := lctx.FlowLogRepo.UpdateApprovalLog(ctx, tx, lctx.Leave.ID, lctx.Flow.ApprovalLog); err != nil {
		return utils.CustomErr(nil, http.StatusInternalServerError, "failed to update approval log: "+err.Error())
	}

	// 7. If higher APPROVED stages still exist → intermediate, not fully withdrawn yet
	if !allStagesSettledForWithdraw(lctx.Flow.ApprovalLog) {
		if err := lctx.CommRepo.UpdateLeaveStatusWithApprover(
			tx.Tx, lctx.Leave.ID, constant.LEAVE_WITHDRAWAL_PENDING, lctx.ApproverID,
		); err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError, "failed to set withdrawal pending: "+err.Error())
		}
		return nil
	}

	// 8. All stages settled → fully withdrawn
	if err := lctx.CommRepo.UpdateLeaveStatusWithApprover(
		tx.Tx, lctx.Leave.ID, constant.LEAVE_WITHDRAWN, lctx.ApproverID,
	); err != nil {
		return utils.CustomErr(nil, http.StatusInternalServerError, "failed to withdraw leave: "+err.Error())
	}

	// 9. Restore balance ONLY for the final (highest stage_no) approver — non-early only
	if isFinalWithdrawStage(lctx.Flow.ApprovalLog, stage.StageNo) && !isEarlyLeave(lctx.LeaveType) {
		if err := lctx.CommRepo.UpdateWidthrowLeaveBalanceByEmployeeId(tx, lctx.Leave.EmployeeID, lctx.Leave.LeaveTypeID, lctx.Leave.Days); err != nil {
			return utils.CustomErr(nil, http.StatusInternalServerError, "failed to restore leave balance: "+err.Error())
		}
	}

	return nil
}
