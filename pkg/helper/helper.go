package helper

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	utils "github.com/sanjayk-eng/UserMenagmentSystem_Backend/pkg"
)

// extractEmployeeID reads the authenticated employee's UUID from the gin context.
// Writes the appropriate error response and returns false on failure.
func ExtractEmployeeID(c *gin.Context) (uuid.UUID, bool) {
	raw, ok := c.Get("user_id")
	if !ok {
		utils.RespondWithError(c, http.StatusUnauthorized, "employee ID missing")
		return uuid.Nil, false
	}
	str, ok := raw.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee ID format")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(str)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "invalid employee UUID")
		return uuid.Nil, false
	}
	return id, true
}
