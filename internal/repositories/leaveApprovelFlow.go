package repositories

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
)

// =====================================================
// INTERFACE
// =====================================================

type LeaveApprovalFlowRepository interface {
	InsertFlow(ctx context.Context, req *models.LeaveApprovalFlowRequest) error
	GetAllFlows(ctx context.Context) ([]models.LeaveApprovalFlow, error)
	GetById(ctx context.Context, id string) (*models.LeaveApprovalFlow, error)
	UpdateLeaveApprovelFlow(ctx context.Context, id string, req *models.LeaveApprovalFlowRequest) error
	DeleteLeaveApprovelFlow(ctx context.Context, id string) error
	GetDefaultFlowID(ctx context.Context) (string, error)
}

// =====================================================
// IMPLEMENTATION
// =====================================================

type leaveApprovalFlowRepo struct {
	db *sqlx.DB
}

func NewLeaveApprovalFlowRepository(db *sqlx.DB) LeaveApprovalFlowRepository {
	return &leaveApprovalFlowRepo{db: db}
}

// =====================================================
// INSERT ONLY (NO DELETE HERE)
// =====================================================
func (r *leaveApprovalFlowRepo) InsertFlow(ctx context.Context, req *models.LeaveApprovalFlowRequest) error {

	query := `
		INSERT INTO leave_approval_flow (name, flow)
		VALUES ($1, $2)
	`

	flowBytes, err := json.Marshal(req.Flow)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query, req.Name, flowBytes)
	if err != nil {
		return err
	}

	return nil
}

// // =====================================================
// // GET FLOW
// // =====================================================
func (r *leaveApprovalFlowRepo) GetAllFlows(ctx context.Context) ([]models.LeaveApprovalFlow, error) {

	query := `
		SELECT id, name, is_system, flow, created_at, updated_at
            FROM leave_approval_flow ORDER BY created_at DESC;
	`

	var flows []models.LeaveApprovalFlow

	err := r.db.SelectContext(ctx, &flows, query)
	if err != nil {
		return nil, err
	}

	return flows, nil
}
func (r *leaveApprovalFlowRepo) GetById(ctx context.Context, id string) (*models.LeaveApprovalFlow, error) {

	query := `
		SELECT 
			id, 
			name, 
			is_system, 
			flow, 
			created_at, 
			updated_at
		FROM leave_approval_flow
		WHERE id = $1
	`

	var flow models.LeaveApprovalFlow

	err := r.db.GetContext(ctx, &flow, query, id)
	if err != nil {
		return nil, err
	}

	return &flow, nil
}
func (r *leaveApprovalFlowRepo) UpdateLeaveApprovelFlow(ctx context.Context, id string, req *models.LeaveApprovalFlowRequest) error {

	query := `
		UPDATE leave_approval_flow
		SET 
			name = $1,
			flow = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND is_system = false
	`

	flowBytes, err := json.Marshal(req.Flow)
	if err != nil {
		return err
	}

	res, err := r.db.ExecContext(ctx, query, req.Name, flowBytes, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	//  IMPORTANT: not found case
	if rowsAffected == 0 {
		return fmt.Errorf("leave approval flow not found")
	}
	return nil
}

func (r *leaveApprovalFlowRepo) DeleteLeaveApprovelFlow(ctx context.Context, id string) error {

	query := `
		DELETE FROM leave_approval_flow
		WHERE id = $1 AND is_system = false
	`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// not found case
	if rowsAffected == 0 {
		return fmt.Errorf("leave approval flow not found")
	}

	return nil
}

func (r *leaveApprovalFlowRepo) GetDefaultFlowID(ctx context.Context) (string, error) {

	query := `
		SELECT id
		FROM leave_approval_flow
		WHERE is_system = true
		LIMIT 1
	`

	var id string

	err := r.db.GetContext(ctx, &id, query)
	if err != nil {
		return "", err
	}

	return id, nil
}
