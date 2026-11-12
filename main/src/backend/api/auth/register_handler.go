package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
)

// registerRequest is the request DTO for POST /api/v1/auth/register. It carries
// no validation tags: semantic validation is owned by the registration service.
type registerRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	// Language is the interface language at sign-up ("en" or "pt"); it
	// becomes the language of the account's emails. Optional.
	Language string `json:"language"`
}

// userResponse only exposes safe public user information.
type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// accountResponse is the owner's own view of the account (/me and
// registration): the public fields plus the account settings only the owner
// sees.
type accountResponse struct {
	userResponse
	// EmailVerified reports whether the owner confirmed the address.
	EmailVerified bool `json:"email_verified"`
	// PreferredLanguage is the language of the account's emails, when set.
	PreferredLanguage *string `json:"preferred_language"`
	// PasswordSet is false for an account created through Google until the
	// owner sets a password.
	PasswordSet bool `json:"password_set"`
}

// AfterRegister runs after a successful registration (store the language,
// send the verification email). It never fails the registration.
type AfterRegister func(ctx context.Context, userID, language string)

// RegisterHandler exposes the registration service over HTTP.
type RegisterHandler struct {
	registration  registration.RegistrationService
	afterRegister AfterRegister
}

func NewRegisterHandler(svc registration.RegistrationService) *RegisterHandler {
	return &RegisterHandler{registration: svc}
}

// WithAfterRegister installs the post-registration hook.
func (h *RegisterHandler) WithAfterRegister(hook AfterRegister) *RegisterHandler {
	h.afterRegister = hook
	return h
}

func (h *RegisterHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	user, err := h.registration.Register(c.Request.Context(), registration.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrDuplicateEmail):
			RespondError(c, http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "Email already in use.", nil)
		case errors.Is(err, registration.ErrInvalidInput), password.IsPolicyError(err):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	if h.afterRegister != nil {
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					slog.Error("post-registration hook panicked", slog.String("user_id", user.ID))
				}
			}()
			h.afterRegister(c.Request.Context(), user.ID, req.Language)
		}()
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Account created successfully.",
		"data":    newAccountResponse(user),
	})
}

func newUserResponse(user *models.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func newAccountResponse(user *models.User) accountResponse {
	return accountResponse{
		userResponse:      newUserResponse(user),
		EmailVerified:     user.EmailVerifiedAt != nil,
		PreferredLanguage: user.PreferredLanguage,
		PasswordSet:       user.PasswordSet,
	}
}

// RespondError writes an API error response using the project's envelope.
func RespondError(c *gin.Context, status int, code, message string, details []string) {
	if details == nil {
		details = []string{}
	}
	c.JSON(status, gin.H{
		"success": false,
		"message": message,
		"error": gin.H{
			"code":    code,
			"details": details,
		},
	})
}
