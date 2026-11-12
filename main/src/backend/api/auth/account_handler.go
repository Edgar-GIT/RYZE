package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/account_recovery"
	"ryze/backend/services/password"
)

// AccountHandler exposes password recovery, email verification and the
// account language preference.
type AccountHandler struct {
	account account_recovery.Service
}

func NewAccountHandler(account account_recovery.Service) *AccountHandler {
	return &AccountHandler{account: account}
}

type forgotPasswordRequest struct {
	Email    string `json:"email"`
	Language string `json:"language"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type preferencesRequest struct {
	Language string `json:"language"`
}

// ForgotPassword always answers 202 for a well-formed address, whether or not
// it belongs to an account, so the endpoint cannot be used to discover users.
func (h *AccountHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	if err := h.account.RequestPasswordReset(c.Request.Context(), req.Email, req.Language); err != nil {
		if errors.Is(err, account_recovery.ErrInvalidInput) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Enter a valid email address.", nil)
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "If an account uses this address, a reset link is on its way.",
		"data":    gin.H{},
	})
}

func (h *AccountHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	if err := h.account.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		switch {
		case errors.Is(err, account_recovery.ErrInvalidToken):
			RespondError(c, http.StatusBadRequest, "INVALID_OR_EXPIRED_LINK", "This link is invalid or has expired.", nil)
		case password.IsPolicyError(err):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password updated. Every session was signed out.",
		"data":    gin.H{},
	})
}

func (h *AccountHandler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	if err := h.account.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		if errors.Is(err, account_recovery.ErrInvalidToken) {
			RespondError(c, http.StatusBadRequest, "INVALID_OR_EXPIRED_LINK", "This link is invalid or has expired.", nil)
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Email verified.", "data": gin.H{}})
}

// ResendVerification issues a new verification link for the signed-in account.
func (h *AccountHandler) ResendVerification(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	if err := h.account.SendVerification(c.Request.Context(), userID); err != nil {
		switch {
		case errors.Is(err, account_recovery.ErrAlreadyVerified):
			RespondError(c, http.StatusConflict, "EMAIL_ALREADY_VERIFIED", "This email is already verified.", nil)
		case errors.Is(err, account_recovery.ErrUserNotFound):
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "message": "A verification link is on its way.", "data": gin.H{}})
}

// UpdatePreferences stores the signed-in account's language.
func (h *AccountHandler) UpdatePreferences(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var req preferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	if err := h.account.SetLanguage(c.Request.Context(), userID, req.Language); err != nil {
		switch {
		case errors.Is(err, account_recovery.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Language must be en or pt.", nil)
		case errors.Is(err, account_recovery.ErrUserNotFound):
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Preferences saved.", "data": gin.H{"language": req.Language}})
}
