package utils

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RespondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

// Removed unused gin.Context parameter
func CustomErr(ctx *gin.Context, code int, message string) error {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func HTTPStatus(err error) (int, string) {
	var appErr *AppError

	if errors.As(err, &appErr) {
		return appErr.Code, appErr.Message
	}

	return http.StatusInternalServerError, "internal server error"
}

func Error(c *gin.Context, err error) {
	status, msg := HTTPStatus(err)

	// Reuse common error response format
	RespondWithError(c, status, msg)
}
