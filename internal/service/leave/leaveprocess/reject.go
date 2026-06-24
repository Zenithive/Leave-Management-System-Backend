package leaveprocess

import (
	"context"
	"net/http"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/Zenithive/LeaveManagementSystem/pkg/constant"
	"github.com/jmoiron/sqlx"
)

type RejectProcessor struct{}

func (p *RejectProcessor) Process(ctx context.Context, tx *sqlx.Tx, lctx *LeaveActionContext) error {

	// 1. Find caller's stage
	stage := findStage(lctx.Flow, lctx.Role)
	if stage == nil {
		return errors.CustomErr(nil, http.StatusForbidden, "leave alredy process")
	}

	// 2. Stamp this stage → REJECTED
	stampStage(stage, models.REJECTED, lctx.ApproverID, lctx.Remarks)

	// 3. Auto-SKIP all other WAITING siblings at the same stage_no or less then stage
	skipAllWaitingStages(lctx.Flow.ApprovalLog, stage.StageNo, lctx.Role)

	// 3. Persist the updated approval log
	if err := lctx.FlowLogRepo.UpdateApprovalLog(ctx, tx, lctx.Leave.ID, lctx.Flow.ApprovalLog); err != nil {
		return errors.CustomErr(nil, http.StatusInternalServerError, "failed to update approval log: "+err.Error())
	}

	// 4. Rejection is always final — set leave status immediately
	if err := lctx.LeaveFlowRepo.UpdateLeaveStatusTx(tx.Tx, lctx.Leave.ID, constant.LEAVE_REJECTED, lctx.ApproverID); err != nil {
		return errors.CustomErr(nil, http.StatusInternalServerError, "failed to reject leave: "+err.Error())
	}

	return nil
}
