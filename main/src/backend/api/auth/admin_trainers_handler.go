package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/admin_trainers"
	"ryze/backend/services/password"
)

// AdminTrainerHandler exposes administrator trainer-management operations. It
// never performs authentication or authorization itself: those are enforced by
// the AdminAuthenticate and RequireAdminPermission middleware mounted on the
// routes. Read endpoints run under trainers.read (both roles); lifecycle
// endpoints run under trainers.manage (Technical Administrator only).
type AdminTrainerHandler struct {
	service admin_trainers.AdminTrainerService
}

func NewAdminTrainerHandler(svc admin_trainers.AdminTrainerService) *AdminTrainerHandler {
	return &AdminTrainerHandler{service: svc}
}

// createAdminTrainerRequest is the request DTO for creating a trainer. It
// never accepts the trainer id, the user id or deleted_at: those are
// controlled exclusively by the backend.
type createAdminTrainerRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// updateAdminTrainerRequest is the request DTO for updating a trainer. Only
// safe whitelisted user-profile fields are accepted.
type updateAdminTrainerRequest struct {
	Email     *string `json:"email"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

// trainerResponse only exposes safe trainer information: the trainer identity,
// the linked user's profile data and the trainer lifecycle status. Password
// hashes, session versions and deletion markers are never exposed.
type trainerResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListTrainers returns one page of active trainers with pagination metadata.
func (h *AdminTrainerHandler) ListTrainers(c *gin.Context) {
	result, ok := h.listTrainers(c, false)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainers retrieved successfully.",
		"data":    listTrainersData(result),
	})
}

// ListDeletedTrainers returns one page of soft-deleted trainers with
// pagination metadata. It is a clearly separated view for lifecycle
// management; active and deleted trainers are never mixed in a single list.
// The same safe representation is used and deleted_at is never exposed.
func (h *AdminTrainerHandler) ListDeletedTrainers(c *gin.Context) {
	result, ok := h.listTrainers(c, true)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Deleted trainers retrieved successfully.",
		"data":    listTrainersData(result),
	})
}

func (h *AdminTrainerHandler) listTrainers(c *gin.Context, deleted bool) (admin_trainers.ListTrainersResult, bool) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return admin_trainers.ListTrainersResult{}, false
	}
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return admin_trainers.ListTrainersResult{}, false
	}

	var result admin_trainers.ListTrainersResult
	if deleted {
		result, err = h.service.ListDeletedTrainers(c.Request.Context(), page, limit)
	} else {
		result, err = h.service.ListTrainers(c.Request.Context(), page, limit)
	}
	if err != nil {
		switch {
		case errors.Is(err, admin_trainers.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return admin_trainers.ListTrainersResult{}, false
	}
	return result, true
}

func listTrainersData(result admin_trainers.ListTrainersResult) gin.H {
	trainers := make([]trainerResponse, 0, len(result.Trainers))
	for _, trainer := range result.Trainers {
		trainers = append(trainers, newTrainerResponse(trainer))
	}
	return gin.H{
		"trainers": trainers,
		"pagination": gin.H{
			"page":        result.Page,
			"limit":       result.Limit,
			"total":       result.Total,
			"total_pages": totalPages(result.Total, result.Limit),
		},
	}
}

func newTrainerResponse(result admin_trainers.TrainerResult) trainerResponse {
	status := "active"
	if result.Trainer.DeletedAt.Valid {
		status = "disabled"
	}
	return trainerResponse{
		ID:        result.Trainer.ID,
		UserID:    result.Trainer.UserID,
		Email:     result.User.Email,
		FirstName: result.User.FirstName,
		LastName:  result.User.LastName,
		Status:    status,
		CreatedAt: result.Trainer.CreatedAt,
		UpdatedAt: result.Trainer.UpdatedAt,
	}
}

// GetTrainer returns one active trainer by UUID. Soft-deleted trainers are
// never returned.
func (h *AdminTrainerHandler) GetTrainer(c *gin.Context) {
	result, err := h.service.GetTrainer(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, admin_trainers.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_trainers.ErrTrainerNotFound):
			RespondError(c, http.StatusNotFound, "TRAINER_NOT_FOUND", "Trainer not found.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer retrieved successfully.",
		"data":    newTrainerResponse(*result),
	})
}

// CreateTrainer creates a new user account and its trainer profile.
func (h *AdminTrainerHandler) CreateTrainer(c *gin.Context) {
	var req createAdminTrainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	result, err := h.service.CreateTrainer(c.Request.Context(), admin_trainers.CreateTrainerInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin_trainers.ErrInvalidInput), errors.Is(err, password.ErrEmptyPassword):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_trainers.ErrDuplicateEmail):
			RespondError(c, http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "Email already in use.", nil)
		case errors.Is(err, admin_trainers.ErrTrainerAlreadyLinked):
			RespondError(c, http.StatusConflict, "TRAINER_ALREADY_LINKED", "User already has an active trainer.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Trainer created successfully.",
		"data":    newTrainerResponse(*result),
	})
}

// UpdateTrainer updates the whitelisted safe profile fields of an active
// trainer's user.
func (h *AdminTrainerHandler) UpdateTrainer(c *gin.Context) {
	var req updateAdminTrainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	result, err := h.service.UpdateTrainer(c.Request.Context(), c.Param("id"), admin_trainers.UpdateTrainerInput{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		switch {
		case errors.Is(err, admin_trainers.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_trainers.ErrTrainerNotFound):
			RespondError(c, http.StatusNotFound, "TRAINER_NOT_FOUND", "Trainer not found.", nil)
		case errors.Is(err, admin_trainers.ErrUserInactive):
			RespondError(c, http.StatusConflict, "USER_DISABLED", "Linked user is disabled.", nil)
		case errors.Is(err, admin_trainers.ErrDuplicateEmail):
			RespondError(c, http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "Email already in use.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer updated successfully.",
		"data":    newTrainerResponse(*result),
	})
}

// SoftDeleteTrainer soft-deletes an active trainer by UUID. The row is
// preserved and the linked user account is never touched.
func (h *AdminTrainerHandler) SoftDeleteTrainer(c *gin.Context) {
	if err := h.service.SoftDeleteTrainer(c.Request.Context(), c.Param("id")); err != nil {
		switch {
		case errors.Is(err, admin_trainers.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_trainers.ErrTrainerNotFound):
			RespondError(c, http.StatusNotFound, "TRAINER_NOT_FOUND", "Trainer not found.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer disabled successfully.",
		"data":    gin.H{},
	})
}

// ReactivateTrainer restores a soft-deleted trainer: the same trainer UUID and
// user link are preserved and the trainer becomes active again.
func (h *AdminTrainerHandler) ReactivateTrainer(c *gin.Context) {
	result, err := h.service.ReactivateTrainer(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, admin_trainers.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, admin_trainers.ErrTrainerNotFound):
			RespondError(c, http.StatusNotFound, "TRAINER_NOT_FOUND", "Trainer not found.", nil)
		case errors.Is(err, admin_trainers.ErrAlreadyActive):
			RespondError(c, http.StatusConflict, "TRAINER_ALREADY_ACTIVE", "Trainer is already active.", nil)
		case errors.Is(err, admin_trainers.ErrUserInactive):
			RespondError(c, http.StatusConflict, "USER_DISABLED", "Linked user is disabled.", nil)
		case errors.Is(err, admin_trainers.ErrTrainerAlreadyLinked):
			RespondError(c, http.StatusConflict, "TRAINER_ALREADY_LINKED", "User already has an active trainer.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer reactivated successfully.",
		"data":    newTrainerResponse(*result),
	})
}
