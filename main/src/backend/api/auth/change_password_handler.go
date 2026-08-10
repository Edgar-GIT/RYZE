package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/change_password"
	"ryze/backend/services/password"
)

// changePasswordRequest is the request DTO for POST /api/v1/auth/change-password.
// It carries no validation tags and never accepts a user identifier: identity
// is resolved exclusively from the authenticated session.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePasswordHandler exposes the change-password service over HTTP and
// clears the access-token cookie after a successful change.
type ChangePasswordHandler struct {
	service change_password.ChangePasswordService
	secure  bool
}

func NewChangePasswordHandler(svc change_password.ChangePasswordService, secure bool) *ChangePasswordHandler {
	return &ChangePasswordHandler{service: svc, secure: secure}
}

func (h *ChangePasswordHandler) ChangePassword(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	if _, err := h.service.ChangePassword(c.Request.Context(), change_password.Input{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		switch {
		case errors.Is(err, change_password.ErrInvalidCredentials):
			RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials.", nil)
		case errors.Is(err, change_password.ErrInvalidInput), errors.Is(err, password.ErrEmptyPassword):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	http.SetCookie(c.Writer, accessTokenCookie("", -1, h.secure))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password changed successfully.",
		"data":    gin.H{},
	})
}
