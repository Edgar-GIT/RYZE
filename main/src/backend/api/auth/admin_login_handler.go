package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/admin_login"
	"ryze/backend/services/token"
)

// adminLoginRequest is the request DTO for POST /api/v1/admin/auth/login. It
// carries no validation tags: semantic validation is owned by the service.
type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// adminVerifyRequest is the request DTO for POST /api/v1/admin/auth/verify.
type adminVerifyRequest struct {
	AccessCode string `json:"access_code"`
}

// AdminLoginHandler exposes the two-stage admin authentication flow over HTTP:
// Login completes the username/password stage and Verify completes the access
// code stage before the final admin session cookie is issued.
type AdminLoginHandler struct {
	login  admin_login.AdminService
	tokens token.Service
	ttl    time.Duration
	secure bool
}

func NewAdminLoginHandler(loginSvc admin_login.AdminService, tokens token.Service, ttl time.Duration, secure bool) *AdminLoginHandler {
	return &AdminLoginHandler{login: loginSvc, tokens: tokens, ttl: ttl, secure: secure}
}

// Login validates the username/password stage. On success it issues a
// short-lived temporary authentication state as an HttpOnly stage cookie; the
// final admin session cookie is only issued by Verify after the access code is
// confirmed.
func (h *AdminLoginHandler) Login(c *gin.Context) {
	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	admin, err := h.login.Login(c.Request.Context(), admin_login.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin_login.ErrInvalidCredentials):
			RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials.", nil)
		case errors.Is(err, admin_login.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	stageToken, err := h.tokens.GenerateAdminStageToken(admin.ID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}

	http.SetCookie(c.Writer, adminStageTokenCookie(stageToken, accessTokenLifetime(token.AdminStageTokenTTL), h.secure))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Admin access code required.",
		"data":    gin.H{},
	})
}

// Verify validates the access code for the administrator identified in the
// temporary stage state. Only after the code is confirmed does it clear the
// stage cookie and issue the final ryze_admin_access_token admin session
// cookie. Every failure (missing, invalid or expired stage state, wrong access
// code) maps to the same generic authentication error.
func (h *AdminLoginHandler) Verify(c *gin.Context) {
	stageToken, err := c.Cookie(AdminStageTokenCookieName)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials.", nil)
		return
	}

	adminID, err := h.tokens.ValidateAdminStageToken(stageToken)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials.", nil)
		return
	}

	var req adminVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	admin, err := h.login.VerifyAccessCode(c.Request.Context(), admin_login.VerifyInput{
		AdminID:    adminID,
		AccessCode: req.AccessCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin_login.ErrInvalidCredentials):
			RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials.", nil)
		case errors.Is(err, admin_login.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	accessToken, err := h.tokens.GenerateAdminToken(admin.ID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}

	http.SetCookie(c.Writer, adminStageTokenCookie("", -1, h.secure))
	http.SetCookie(c.Writer, adminAccessTokenCookie(accessToken, accessTokenLifetime(h.ttl), h.secure))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Admin login successful.",
		"data":    gin.H{},
	})
}
