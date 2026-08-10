package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/login"
	"ryze/backend/services/password"
	"ryze/backend/services/token"
)

// loginRequest is the request DTO for POST /api/v1/auth/login. It carries no
// validation tags: semantic validation is owned by the login service.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginHandler exposes the login service over HTTP and issues access tokens
// through the JWT token service.
type LoginHandler struct {
	login  login.LoginService
	tokens token.Service
	ttl    time.Duration
}

func NewLoginHandler(loginSvc login.LoginService, tokens token.Service, ttl time.Duration) *LoginHandler {
	return &LoginHandler{login: loginSvc, tokens: tokens, ttl: ttl}
}

func (h *LoginHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	user, err := h.login.Login(c.Request.Context(), login.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, login.ErrInvalidCredentials):
			respondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials.", nil)
		case errors.Is(err, login.ErrInvalidInput), errors.Is(err, password.ErrEmptyPassword):
			respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	accessToken, err := h.tokens.GenerateAccessToken(user.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login successful.",
		"data": gin.H{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   int(h.ttl.Seconds()),
		},
	})
}
