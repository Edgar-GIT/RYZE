package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/free_programs"
	"ryze/backend/services/program_details"
)

type freeActivationResponse struct {
	ID          string                     `json:"id"`
	ActivatedAt string                     `json:"activated_at"`
	ExpiresAt   *string                    `json:"expires_at"`
	Program     freeProgramSummaryResponse `json:"program"`
}

type freeProgramSummaryResponse struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Type             string  `json:"type"`
	Level            *string `json:"level"`
	DurationWeeks    *int    `json:"duration_weeks"`
	FrequencyPerWeek *int    `json:"frequency_per_week"`
	TrainingType     *string `json:"training_type"`
	CoverImageURL    string  `json:"cover_image_url"`
	PriceMinorUnits  int64   `json:"price_minor_units"`
	Currency         string  `json:"currency"`
}

func newFreeActivationResponse(activation *free_programs.Activation) freeActivationResponse {
	return freeActivationResponse{
		ID:          activation.ID,
		ActivatedAt: activation.ActivatedAt,
		ExpiresAt:   activation.ExpiresAt,
		Program: freeProgramSummaryResponse{
			ID:               activation.Program.ID,
			Name:             activation.Program.Name,
			Description:      activation.Program.Description,
			Type:             activation.Program.Type,
			Level:            activation.Program.Level,
			DurationWeeks:    activation.Program.DurationWeeks,
			FrequencyPerWeek: activation.Program.FrequencyPerWeek,
			TrainingType:     activation.Program.TrainingType,
			CoverImageURL:    program_details.CoverURL(activation.Program.CoverImage),
			PriceMinorUnits:  activation.Program.PriceMinorUnits,
			Currency:         activation.Program.Currency,
		},
	}
}

// FreeProgramsHandler exposes Free program activation to authenticated clients.
// Authorization is the authentication context plus the Free product rule
// enforced by the service. The client can never supply a user id.
type FreeProgramsHandler struct {
	service free_programs.Service
}

func NewFreeProgramsHandler(svc free_programs.Service) *FreeProgramsHandler {
	return &FreeProgramsHandler{service: svc}
}

// Activate grants access to one published Free program. Activation is
// idempotent.
func (h *FreeProgramsHandler) Activate(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	activation, err := h.service.Activate(c.Request.Context(), userID, c.Param("programID"))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Free program activated successfully.",
		"data":    newFreeActivationResponse(activation),
	})
}

// ListActive returns the authenticated client's active Free activation.
func (h *FreeProgramsHandler) ListActive(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	activations, err := h.service.ListActive(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	responses := make([]freeActivationResponse, 0, len(activations))
	for i := range activations {
		responses = append(responses, newFreeActivationResponse(&activations[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Free program activations retrieved successfully.",
		"data":    responses,
	})
}

func (h *FreeProgramsHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, free_programs.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, free_programs.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
