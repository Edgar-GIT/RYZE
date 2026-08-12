package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/services/trainer_applications"
)

// applicationResponse only exposes safe application information.
type applicationResponse struct {
	ID        string       `json:"id"`
	Status    string       `json:"status"`
	UserID    string       `json:"user_id"`
	User      userResponse `json:"user"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func newApplicationResponse(application *models.TrainerApplication) applicationResponse {
	return applicationResponse{
		ID:        application.ID,
		Status:    application.Status,
		UserID:    application.UserID,
		User:      newUserResponse(&application.User),
		CreatedAt: application.CreatedAt,
		UpdatedAt: application.UpdatedAt,
	}
}

// AdminTrainerApplicationHandler exposes administrator trainer-application
// review operations. It never performs authentication or authorization itself:
// those are enforced by the AdminAuthenticate and RequireAdminPermission
// middleware mounted on the routes. All endpoints run under
// trainer_applications.read and trainer_applications.manage, both shared by
// every admin role.
type AdminTrainerApplicationHandler struct {
	service trainer_applications.Service
}

func NewAdminTrainerApplicationHandler(svc trainer_applications.Service) *AdminTrainerApplicationHandler {
	return &AdminTrainerApplicationHandler{service: svc}
}

// ListApplications returns one page of trainer applications, optionally
// filtered by status, with pagination metadata.
func (h *AdminTrainerApplicationHandler) ListApplications(c *gin.Context) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	result, err := h.service.ListApplications(c.Request.Context(), page, limit, c.Query("status"))
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	applications := make([]applicationResponse, 0, len(result.Applications))
	for i := range result.Applications {
		applications = append(applications, newApplicationResponse(&result.Applications[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer applications retrieved successfully.",
		"data": gin.H{
			"applications": applications,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// GetApplication returns one trainer application by UUID.
func (h *AdminTrainerApplicationHandler) GetApplication(c *gin.Context) {
	application, err := h.service.GetApplication(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application retrieved successfully.",
		"data":    newApplicationResponse(application),
	})
}

// ApproveApplication approves a PENDING application and creates the trainer
// profile for the applicant in the same database transaction.
func (h *AdminTrainerApplicationHandler) ApproveApplication(c *gin.Context) {
	application, err := h.service.ApproveApplication(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationStateConflict):
			RespondError(c, http.StatusConflict, "APPLICATION_STATE_CONFLICT", "Trainer application is not pending.", nil)
		case errors.Is(err, trainer_applications.ErrAlreadyTrainer):
			RespondError(c, http.StatusConflict, "USER_ALREADY_TRAINER", "User is already a trainer.", nil)
		case errors.Is(err, trainer_applications.ErrUserNotFound):
			RespondError(c, http.StatusConflict, "USER_NOT_ACTIVE", "User is not active.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application approved successfully.",
		"data":    newApplicationResponse(application),
	})
}

// RejectApplication rejects a PENDING application. The application stays in
// history so the user can apply again.
func (h *AdminTrainerApplicationHandler) RejectApplication(c *gin.Context) {
	if err := h.service.RejectApplication(c.Request.Context(), c.Param("id")); err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationNotFound):
			RespondError(c, http.StatusNotFound, "APPLICATION_NOT_FOUND", "Trainer application not found.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationStateConflict):
			RespondError(c, http.StatusConflict, "APPLICATION_STATE_CONFLICT", "Trainer application is not pending.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer application rejected successfully.",
		"data":    gin.H{},
	})
}

// TrainerApplicationHandler exposes the trainer-application flow to regular
// authenticated users. The user identity always comes from the authenticated
// session; the request body can never influence it.
type TrainerApplicationHandler struct {
	service trainer_applications.Service
}

func NewTrainerApplicationHandler(svc trainer_applications.Service) *TrainerApplicationHandler {
	return &TrainerApplicationHandler{service: svc}
}

// Apply creates a PENDING trainer application for the authenticated user. The
// user ID is resolved exclusively from the authenticated session; any
// client-provided identity is ignored.
func (h *TrainerApplicationHandler) Apply(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	application, err := h.service.Apply(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, trainer_applications.ErrUserNotFound):
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		case errors.Is(err, trainer_applications.ErrAlreadyTrainer):
			RespondError(c, http.StatusConflict, "ALREADY_TRAINER", "You are already a trainer.", nil)
		case errors.Is(err, trainer_applications.ErrApplicationAlreadyActive):
			RespondError(c, http.StatusConflict, "APPLICATION_ALREADY_EXISTS", "You already have an active trainer application.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Trainer application submitted successfully.",
		"data":    newApplicationResponse(application),
	})
}
