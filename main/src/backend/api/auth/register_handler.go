package auth

import (
	"errors"
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
}

// registerResponse only exposes safe public user information.
type registerResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RegisterHandler exposes the registration service over HTTP.
type RegisterHandler struct {
	registration registration.RegistrationService
}

func NewRegisterHandler(svc registration.RegistrationService) *RegisterHandler {
	return &RegisterHandler{registration: svc}
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
		case errors.Is(err, registration.ErrInvalidInput), errors.Is(err, password.ErrEmptyPassword):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Account created successfully.",
		"data":    newRegisterResponse(user),
	})
}

func newRegisterResponse(user *models.User) registerResponse {
	return registerResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
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
