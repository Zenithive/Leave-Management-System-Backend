package leaveprocess

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

type ApproveProcessor struct {
}

func (p *ApproveProcessor) Process(ctx context.Context, tx *sqlx.Tx, flow *models.LeaveFlow, leave *models.Leave, leaveType *models.LeaveType) error {

	// Approve logic

	return nil
}
