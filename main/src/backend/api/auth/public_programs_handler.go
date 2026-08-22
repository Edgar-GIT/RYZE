package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/public_programs"
)

// publicProgramResponse only exposes safe published program metadata: the
// program identity, its owner trainer id, the public product fields and
// the lifecycle timestamps. Deletion markers, draft programs and any
// internal data are never exposed.
type publicProgramResponse struct {
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

func newPublicProgramResponse(program *public_programs.Program) publicProgramResponse {
	return publicProgramResponse{
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

// PublicProgramsHandler exposes the public, read-only program catalog. These
// endpoints require no authentication and never perform authorization checks:
// the catalog is global and identical for every caller. No write operation is
// exposed in this foundation.
type PublicProgramsHandler struct {
	service public_programs.Service
}

func NewPublicProgramsHandler(svc public_programs.Service) *PublicProgramsHandler {
	return &PublicProgramsHandler{service: svc}
}

// ListPublishedPrograms returns one page of the published program catalog.
// When search, filter, or sort query parameters are provided, the endpoint
// delegates to SearchPublishedPrograms for enhanced discovery. Otherwise it
// returns the default catalog ordered by creation time (newest first).
func (h *PublicProgramsHandler) ListPublishedPrograms(c *gin.Context) {
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

	query := strings.TrimSpace(c.Query("q"))
	programType := strings.TrimSpace(c.Query("type"))
	sortBy := strings.TrimSpace(c.Query("sort"))
	order := strings.TrimSpace(c.Query("order"))

	hasSearchParams := query != "" || programType != "" || sortBy != "" || order != ""

	var result public_programs.ListProgramsResult
	if hasSearchParams {
		result, err = h.service.SearchPublishedPrograms(c.Request.Context(), query, programType, sortBy, order, page, limit)
	} else {
		result, err = h.service.ListPublishedPrograms(c.Request.Context(), page, limit)
	}
	if err != nil {
		h.respondPublicProgramsError(c, err)
		return
	}

	programs := make([]publicProgramResponse, 0, len(result.Programs))
	for i := range result.Programs {
		programs = append(programs, newPublicProgramResponse(&result.Programs[i]))
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

// GetPublishedProgram returns one published program. The program id in the
// path only identifies the requested resource; the catalog is the same for
// every caller.
func (h *PublicProgramsHandler) GetPublishedProgram(c *gin.Context) {
	program, err := h.service.GetPublishedProgram(c.Request.Context(), c.Param("programID"))
	if err != nil {
		h.respondPublicProgramsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program retrieved successfully.",
		"data":    newPublicProgramResponse(program),
	})
}

// respondPublicProgramsError maps public programs service errors to API
// responses. Internal error details are never exposed to the client.
func (h *PublicProgramsHandler) respondPublicProgramsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, public_programs.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, public_programs.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
