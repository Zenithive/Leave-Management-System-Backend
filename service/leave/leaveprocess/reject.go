package leaveprocess

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

type RejectProcessor struct {
}

func (p *RejectProcessor) Process(ctx context.Context, tx *sqlx.Tx, flow *models.LeaveFlow, leave *models.Leave, leaveType *models.LeaveType) error {

	// Reject logic

	return nil
}
