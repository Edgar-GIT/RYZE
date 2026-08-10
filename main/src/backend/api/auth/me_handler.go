package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/repositories"
)

// MeHandler returns the authenticated user's safe information.
type MeHandler struct {
	users repositories.UserRepository
}

func NewMeHandler(users repositories.UserRepository) *MeHandler {
	return &MeHandler{users: users}
}

// GetMe resolves the authenticated user UUID from the context (set by the
// authentication middleware) and returns the user through the repository.
func (h *MeHandler) GetMe(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	user, err := h.users.FindByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User retrieved successfully.",
		"data":    newUserResponse(user),
	})
}
