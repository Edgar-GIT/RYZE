package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/programs"
)

// trainerProgramResponse only exposes safe program metadata: the program
// identity, its owner trainer id, the public product fields and the lifecycle
// timestamps. Deletion markers and any internal data are never exposed.
type trainerProgramResponse struct {
	ID              string    `json:"id"`
	TrainerID       string    `json:"trainer_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	PriceMinorUnits int64     `json:"price_minor_units"`
	Currency        string    `json:"currency"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func newTrainerProgramResponse(program *programs.Program) trainerProgramResponse {
	return trainerProgramResponse{
		ID:              program.ID,
		TrainerID:       program.TrainerID,
		Name:            program.Name,
		Description:     program.Description,
		Type:            program.Type,
		Status:          program.Status,
		PriceMinorUnits: program.PriceMinorUnits,
		Currency:        program.Currency,
		CreatedAt:       program.CreatedAt,
		UpdatedAt:       program.UpdatedAt,
	}
}

// createTrainerProgramRequest is the request DTO for POST /api/v1/trainer/programs.
// It never accepts trainer_id: the owner is always the authenticated trainer.
type createTrainerProgramRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	PriceMinorUnits int64  `json:"price_minor_units"`
	Currency        string `json:"currency"`
}

// updateTrainerProgramRequest is the request DTO for PATCH /api/v1/trainer/programs/:programID.
// Nil values mean "leave unchanged"; trainer_id is immutable and never accepted.
type updateTrainerProgramRequest struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	Type            *string `json:"type"`
	Status          *string `json:"status"`
	PriceMinorUnits *int64  `json:"price_minor_units"`
	Currency        *string `json:"currency"`
}

// TrainerProgramsHandler exposes the authenticated trainer's program
// management operations. It never performs authentication or authorization
// itself: those are enforced by the Authenticate, TrainerAuthenticate and
// RequireTrainerPermission middleware mounted on the route. The trainer
// identity always comes exclusively from the trainer context; request
// parameters, body, headers or any client-supplied identity can never
// influence which program is created, listed, read, updated or deleted.
type TrainerProgramsHandler struct {
	service programs.Service
}

func NewTrainerProgramsHandler(svc programs.Service) *TrainerProgramsHandler {
	return &TrainerProgramsHandler{service: svc}
}

// CreateProgram creates a new program owned by the authenticated trainer. A
// client-supplied trainer_id in the body is deliberately ignored.
func (h *TrainerProgramsHandler) CreateProgram(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	var req createTrainerProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	program, err := h.service.CreateProgram(c.Request.Context(), trainerID, programs.CreateProgramInput{
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		Status:          req.Status,
		PriceMinorUnits: req.PriceMinorUnits,
		Currency:        req.Currency,
	})
	if err != nil {
		h.respondProgramsError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Program created successfully.",
		"data":    newTrainerProgramResponse(program),
	})
}

// ListPrograms returns one page of the authenticated trainer's active programs.
// A client-supplied trainer_id query parameter is deliberately ignored.
func (h *TrainerProgramsHandler) ListPrograms(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

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

	result, err := h.service.ListPrograms(c.Request.Context(), trainerID, page, limit)
	if err != nil {
		h.respondProgramsError(c, err)
		return
	}

	programs := make([]trainerProgramResponse, 0, len(result.Programs))
	for i := range result.Programs {
		programs = append(programs, newTrainerProgramResponse(&result.Programs[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Programs retrieved successfully.",
		"data": gin.H{
			"programs": programs,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// GetProgram returns one of the authenticated trainer's own active programs.
// The program id in the path only identifies the requested resource and never
// proves access: only a program owned by the authenticated trainer is ever
// returned.
func (h *TrainerProgramsHandler) GetProgram(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	program, err := h.service.GetProgram(c.Request.Context(), trainerID, c.Param("programID"))
	if err != nil {
		h.respondProgramsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program retrieved successfully.",
		"data":    newTrainerProgramResponse(program),
	})
}

// UpdateProgram updates the whitelisted fields of one of the authenticated
// trainer's own programs. The owner trainer id is immutable and can never be
// changed by the client.
func (h *TrainerProgramsHandler) UpdateProgram(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	var req updateTrainerProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	program, err := h.service.UpdateProgram(c.Request.Context(), trainerID, c.Param("programID"), programs.UpdateProgramInput{
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		Status:          req.Status,
		PriceMinorUnits: req.PriceMinorUnits,
		Currency:        req.Currency,
	})
	if err != nil {
		h.respondProgramsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program updated successfully.",
		"data":    newTrainerProgramResponse(program),
	})
}

// PublishProgram transitions a draft program to published. The program id in
// the path only identifies the requested resource and never proves access: only
// a draft program owned by the authenticated trainer can be published. A
// client-supplied trainer_id is deliberately ignored.
func (h *TrainerProgramsHandler) PublishProgram(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	program, err := h.service.PublishProgram(c.Request.Context(), trainerID, c.Param("programID"))
	if err != nil {
		h.respondProgramsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program published successfully.",
		"data":    newTrainerProgramResponse(program),
	})
}

// DeleteProgram soft-deletes one of the authenticated trainer's own programs.
// Only the program row is soft-deleted; it is never removed from the database.
func (h *TrainerProgramsHandler) DeleteProgram(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	err := h.service.DeleteProgram(c.Request.Context(), trainerID, c.Param("programID"))
	if err != nil {
		h.respondProgramsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program deleted successfully.",
		"data":    gin.H{},
	})
}

// respondProgramsError maps programs service errors to API responses. Internal
// error details are never exposed to the client.
func (h *TrainerProgramsHandler) respondProgramsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, programs.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, programs.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	case errors.Is(err, programs.ErrProgramAlreadyPublished):
		RespondError(c, http.StatusConflict, "PROGRAM_ALREADY_PUBLISHED", "Program is already published.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
