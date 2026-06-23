package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/models"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/internal/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/utils"
)

// =====================================================
// SERVICE INTERFACE
// =====================================================

type LeaveApprovalFlowService interface {
	CreateLeaveApproverFlow(ctx context.Context, req *models.LeaveApprovalFlowRequest) error
	GetLeaveApprovalFlowById(ctx context.Context, id string) (*models.LeaveApprovalFlowResponse, error)
	GetAllLeaveApprovalFlows(ctx context.Context) ([]models.LeaveApprovalFlowResponse, error)
	UpdateLeaveApprovelFlow(ctx context.Context, id string, req *models.LeaveApprovalFlowRequest) error
	DeleteLeaveApprovelFlow(ctx context.Context, id string) error
}

// =====================================================
// SERVICE STRUCT
// =====================================================

type leaveApprovalFlowService struct {
	DB   *sqlx.DB
	Repo repositories.LeaveApprovalFlowRepository
}

// constructor
func NewLeaveApprovalFlowService(db *sqlx.DB, repo repositories.LeaveApprovalFlowRepository) LeaveApprovalFlowService {
	return &leaveApprovalFlowService{
		DB:   db,
		Repo: repo,
	}
}

func (s *leaveApprovalFlowService) CreateLeaveApproverFlow(ctx context.Context, req *models.LeaveApprovalFlowRequest) error {

	// 1. Validate request
	if err := s.AllowToCreateLeaveApprovelFlow(req); err != nil {
		return err
	}
	if err := s.Repo.InsertFlow(ctx, req); err != nil {
		return utils.CustomErr(nil, http.StatusInternalServerError, "failed to insert approval flow")
	}

	return nil
}

func (s *leaveApprovalFlowService) GetAllLeaveApprovalFlows(ctx context.Context) ([]models.LeaveApprovalFlowResponse, error) {

	// 1. Fetch from repo
	flows, err := s.Repo.GetAllFlows(ctx)
	if err != nil {
		fmt.Println("errError", err.Error())
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "failed to get approval flows")
	}

	// 2. Convert DB model → Response model
	var response []models.LeaveApprovalFlowResponse

	for _, flow := range flows {

		var stages []models.ApprovalStage

		if err := json.Unmarshal(flow.Flow, &stages); err != nil {
			return nil, utils.CustomErr(nil, http.StatusInternalServerError, "invalid flow data")
		}

		response = append(response, models.LeaveApprovalFlowResponse{
			ID:       flow.ID,
			Name:     flow.Name,
			IsSystem: flow.IsSystem,
			Flow:     stages,
		})
	}

	return response, nil
}
func (s *leaveApprovalFlowService) GetLeaveApprovalFlowById(ctx context.Context, id string) (*models.LeaveApprovalFlowResponse, error) {

	// 1. Fetch from repo
	flow, err := s.Repo.GetById(ctx, id)
	if err != nil {
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "failed to get approval flow")
	}

	// 2. Convert JSON → struct
	var stages []models.ApprovalStage

	if err := json.Unmarshal(flow.Flow, &stages); err != nil {
		return nil, utils.CustomErr(nil, http.StatusInternalServerError, "invalid flow data")
	}

	// 3. Build response
	response := &models.LeaveApprovalFlowResponse{
		ID:       flow.ID,
		Name:     flow.Name,
		IsSystem: flow.IsSystem,
		Flow:     stages,
	}

	return response, nil
}

func (s *leaveApprovalFlowService) UpdateLeaveApprovelFlow(ctx context.Context, id string, req *models.LeaveApprovalFlowRequest) error {
	// 1. Validate request
	if err := s.AllowToCreateLeaveApprovelFlow(req); err != nil {
		return err
	}
	if err := s.Repo.UpdateLeaveApprovelFlow(ctx, id, req); err != nil {
		return utils.CustomErr(nil, http.StatusInternalServerError, err.Error())
	}
	return nil
}

func (s *leaveApprovalFlowService) DeleteLeaveApprovelFlow(ctx context.Context, id string) error {
	if err := s.Repo.DeleteLeaveApprovelFlow(ctx, id); err != nil {
		return utils.CustomErr(nil, http.StatusInternalServerError, err.Error())
	}
	return nil
}

// VelidateLeaveApproverFLow
func (s *leaveApprovalFlowService) AllowToCreateLeaveApprovelFlow(req *models.LeaveApprovalFlowRequest) error {

	if req == nil {
		return utils.CustomErr(nil, http.StatusBadRequest, "request is nil")
	}

	if len(req.Flow) == 0 {
		return utils.CustomErr(nil, http.StatusBadRequest, "approval flow cannot be empty")
	}

	roleStages := make(map[models.ApproverRole]int)

	validRoles := map[models.ApproverRole]struct{}{
		models.ApproverManager:    {},
		models.ApproverHR:         {},
		models.ApproverAdmin:      {},
		models.ApproverSuperAdmin: {},
	}

	for _, stage := range req.Flow {

		role := models.ApproverRole(stage.ApproverRole)

		if _, ok := validRoles[role]; !ok {
			return utils.CustomErr(
				nil,
				http.StatusBadRequest,
				fmt.Sprintf(
					"invalid approver role: %s",
					stage.ApproverRole,
				),
			)
		}

		if _, exists := roleStages[role]; exists {
			return utils.CustomErr(nil, http.StatusBadRequest, fmt.Sprintf("approver role %s already exists", stage.ApproverRole))
		}

		if stage.StageNo <= 0 {
			return utils.CustomErr(nil, http.StatusBadRequest, fmt.Sprintf("invalid stage number: %d", stage.StageNo))
		}

		roleStages[role] = stage.StageNo
	}

	// MANAGER -> HR -> ADMIN -> SUPERADMIN
	hierarchy := []models.ApproverRole{
		models.ApproverManager,
		models.ApproverHR,
		models.ApproverAdmin,
		models.ApproverSuperAdmin,
	}

	for i := 0; i < len(hierarchy)-1; i++ {

		currentRole := hierarchy[i]
		currentStage, exists := roleStages[currentRole]

		if !exists {
			continue
		}

		for j := i + 1; j < len(hierarchy); j++ {

			nextRole := hierarchy[j]
			nextStage, exists := roleStages[nextRole]

			if !exists {
				continue
			}

			// Same stage allowed
			if currentStage > nextStage {
				return utils.CustomErr(nil, http.StatusBadRequest, fmt.Sprintf("%s cannot be after %s", currentRole, nextRole))
			}
		}
	}

	return nil
}
