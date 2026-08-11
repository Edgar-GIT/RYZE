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

// AdminLoginHandler exposes the admin login service over HTTP and issues admin
// access tokens through the JWT token service.
type AdminLoginHandler struct {
	login  admin_login.AdminService
	tokens token.Service
	ttl    time.Duration
	secure bool
}

func NewAdminLoginHandler(loginSvc admin_login.AdminService, tokens token.Service, ttl time.Duration, secure bool) *AdminLoginHandler {
	return &AdminLoginHandler{login: loginSvc, tokens: tokens, ttl: ttl, secure: secure}
}

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

	accessToken, err := h.tokens.GenerateAdminToken(admin.ID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}

	http.SetCookie(c.Writer, adminAccessTokenCookie(accessToken, accessTokenLifetime(h.ttl), h.secure))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Admin login successful.",
		"data":    gin.H{},
	})
}
