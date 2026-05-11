package controllers

import (
	"github.com/go-playground/validator/v10"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/service"
)

// HandlerFunc holds dependencies
type HandlerFunc struct {
	Env          *config.ENV
	Query        *repositories.Repository
	Validator    *validator.Validate
	LeaveAccrual *service.LeaveAccrualService
}

// NewHandler initializes and returns a HandlerFunc
func NewHandler(env *config.ENV, query *repositories.Repository, validator *validator.Validate) *HandlerFunc {
	return &HandlerFunc{
		Env:       env,
		Query:     query,
		Validator: validator,
	}
}

// SetLeaveAccrualService attaches the accrual service after construction.
// Called from main.go after the service is created.
func (h *HandlerFunc) SetLeaveAccrualService(svc *service.LeaveAccrualService) {
	h.LeaveAccrual = svc
}
