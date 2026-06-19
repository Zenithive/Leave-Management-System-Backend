package leaveprocess

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/models"
)

type CancelProcessor struct {
}

func (p *CancelProcessor) Process(ctx context.Context, tx *sqlx.Tx, flow *models.LeaveFlow, leave *models.Leave, leaveType *models.LeaveType) error {

	// Cancel logic

	return nil
}
