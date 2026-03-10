package controllers

import (
	"github.com/go-playground/validator/v10"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg/config"
	"github.com/sanjayk-eng/UserMenagmentSystem_Backend/repositories"
)

// HandlerFunc holds dependencies
type HandlerFunc struct {
	Env       *config.ENV
	Query     *repositories.Repository
	Validator *validator.Validate
}

// NewHandler initializes and returns a HandlerFunc
func NewHandler(env *config.ENV, query *repositories.Repository, validator *validator.Validate) *HandlerFunc {
	return &HandlerFunc{
		Env:       env,
		Query:     query,
		Validator: validator,
	}
}
