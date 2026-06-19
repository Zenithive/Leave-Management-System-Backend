package leaveprocess

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

type WithdrawProcessor struct {
}

func (p *WithdrawProcessor) Process(ctx context.Context, tx *sqlx.Tx, flow *models.LeaveFlow, leave *models.Leave, leaveType *models.LeaveType) error {

	// Withdraw logic

	return nil
}
