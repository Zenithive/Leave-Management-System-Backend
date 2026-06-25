package leaveprocess

import (
	"context"
	"net/http"

	"github.com/Zenithive/LeaveManagementSystem/internal/models"
	"github.com/Zenithive/LeaveManagementSystem/pkg/common/errors"
	"github.com/Zenithive/LeaveManagementSystem/pkg/constant"
	"github.com/jmoiron/sqlx"
)

type ApproveProcessor struct{}

func (p *ApproveProcessor) Process(ctx context.Context, tx *sqlx.Tx, lctx *LeaveActionContext) error {

	// 1. get caller's stage
	stage := findStage(lctx.Flow, lctx.Role)
	if stage == nil {
		return errors.CustomErr(http.StatusForbidden, "leave alredy process")
	}

	// 2. Stamp this role's stage → APPROVED
	stampStage(stage, models.APPROVED, lctx.ApproverID, lctx.Remarks)

	// 3. Auto-SKIP all other WAITING siblings at the same stage_no or less then stage
	skipStages(lctx.Flow.ApprovalLog, stage.StageNo, lctx.Role)

	// 4. Persist the updated approval log
	if err := lctx.FlowLogRepo.UpdateApprovalLog(ctx, tx, lctx.Leave.ID, lctx.Flow.ApprovalLog); err != nil {
		return errors.CustomErr(http.StatusInternalServerError, "failed to update approval log: "+err.Error())
	}

	// 5. If any stage is still WAITING → more approvers needed, leave stays Pending
	if !allStagesSettled(lctx.Flow.ApprovalLog) {
		return nil
	}

	// 6. Every stage settled → final approval
	if err := lctx.LeaveFlowRepo.UpdateLeaveStatusTx(tx.Tx, lctx.Leave.ID, constant.LEAVE_APPLOVED, lctx.ApproverID); err != nil {
		return errors.CustomErr(http.StatusInternalServerError, "failed to approve leave: "+err.Error())
	}

	// 7. Deduct balance — skip for IsEarly (no balance bucket)
	if !isEarlyLeave(lctx.LeaveType) {
		if err := lctx.CommRepo.UpdateLeaveBalanceByEmployeeId(tx, lctx.Leave.EmployeeID, lctx.Leave.LeaveTypeID, lctx.Leave.Days); err != nil {
			return errors.CustomErr(http.StatusInternalServerError, "failed to deduct leave balance: "+err.Error())
		}
	}

	return nil
}
