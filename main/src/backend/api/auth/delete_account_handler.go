package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/delete_account"
	"ryze/backend/services/password"
)

// deleteAccountRequest is the request DTO for POST /api/v1/auth/delete-account.
// It carries no validation tags and never accepts a user identifier: identity
// is resolved exclusively from the authenticated session.
type deleteAccountRequest struct {
	Password string `json:"password"`
}

// DeleteAccountHandler exposes the delete-account service over HTTP and clears
// the access-token cookie after a successful deletion.
type DeleteAccountHandler struct {
	service delete_account.DeleteAccountService
	secure  bool
}

func NewDeleteAccountHandler(svc delete_account.DeleteAccountService, secure bool) *DeleteAccountHandler {
	return &DeleteAccountHandler{service: svc, secure: secure}
}

func (h *DeleteAccountHandler) DeleteAccount(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	var req deleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	if err := h.service.DeleteAccount(c.Request.Context(), delete_account.Input{
		UserID:   userID,
		Password: req.Password,
	}); err != nil {
		switch {
		case errors.Is(err, delete_account.ErrInvalidCredentials):
			RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials.", nil)
		case errors.Is(err, delete_account.ErrInvalidInput), errors.Is(err, password.ErrEmptyPassword):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	http.SetCookie(c.Writer, accessTokenCookie("", -1, h.secure))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Account deleted successfully.",
		"data":    gin.H{},
	})
}
