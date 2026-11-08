package auth

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/models"
	"ryze/backend/services/public_programs"
)

// publicProgramResponse only exposes safe published program metadata: the
// program identity, its public product fields, the generic audience metadata
// and the lifecycle timestamps. The owning trainer id and the programme
// structure are never exposed publicly; structure is only available through an
// authenticated access grant.
type publicProgramResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Type             string    `json:"type"`
	Status           string    `json:"status"`
	Level            *string   `json:"level"`
	DurationWeeks    *int      `json:"duration_weeks"`
	FrequencyPerWeek *int      `json:"frequency_per_week"`
	TrainingType     *string   `json:"training_type"`
	PriceMinorUnits  int64     `json:"price_minor_units"`
	Currency         string    `json:"currency"`
	ProductType      string    `json:"product_type"`
	Objective        *string   `json:"objective"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func newPublicProgramResponse(program *public_programs.Program) publicProgramResponse {
	return publicProgramResponse{
		ID:               program.ID,
		Name:             program.Name,
		Description:      program.Description,
		Type:             program.Type,
		Status:           program.Status,
		Level:            program.Level,
		DurationWeeks:    program.DurationWeeks,
		FrequencyPerWeek: program.FrequencyPerWeek,
		TrainingType:     program.TrainingType,
		PriceMinorUnits:  program.PriceMinorUnits,
		Currency:         program.Currency,
		ProductType:      program.ProductType,
		Objective:        program.Objective,
		CreatedAt:        program.CreatedAt,
		UpdatedAt:        program.UpdatedAt,
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
// When search, filter, scope, or sort query parameters are provided, the
// endpoint delegates to SearchPublishedPrograms for enhanced discovery.
// Otherwise it returns the default catalog ordered by creation time (newest
// first). scope=generic restricts the catalog to platform-owned (trainer_id
// NULL) programs.
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
	frequencies, err := queryIntList(c, "frequency")
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	durationMin, err := queryInt(c, "duration_min", 0)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	durationMax, err := queryInt(c, "duration_max", 0)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	durationRanges, err := queryDurationRanges(c, "duration_ranges")
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	scope := strings.TrimSpace(c.Query("scope"))
	if scope != "" && scope != "generic" && scope != models.ProgramProductTypePremiumLevel1 && scope != models.ProgramProductTypePremiumLevel2 {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	filter := public_programs.ProgramFilter{
		Query:              strings.TrimSpace(c.Query("q")),
		ProgramType:        strings.TrimSpace(c.Query("type")),
		TrainingTypes:      queryCSV(c, "training_type"),
		Levels:             queryCSV(c, "level"),
		Frequencies:        frequencies,
		DurationMin:        durationMin,
		DurationMax:        durationMax,
		DurationRanges:     durationRanges,
		SortBy:             strings.TrimSpace(c.Query("sort")),
		Order:              strings.TrimSpace(c.Query("order")),
		ScopeGeneric:       scope == "generic",
		ScopePremiumLevel1: scope == models.ProgramProductTypePremiumLevel1,
		ScopePremiumLevel2: scope == models.ProgramProductTypePremiumLevel2,
	}

	hasSearchParams := filter.Query != "" || filter.ProgramType != "" || len(filter.TrainingTypes) > 0 ||
		len(filter.Levels) > 0 || len(filter.Frequencies) > 0 || filter.DurationMin > 0 || filter.DurationMax > 0 ||
		len(filter.DurationRanges) > 0 || filter.SortBy != "" || filter.Order != "" || filter.ScopeGeneric || filter.ScopePremiumLevel1 || filter.ScopePremiumLevel2

	var result public_programs.ListProgramsResult
	if hasSearchParams {
		result, err = h.service.SearchPublishedPrograms(c.Request.Context(), filter, page, limit)
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

// GetPublishedProgram returns one published program with its client-safe
// structure. The program id in the path only identifies the requested
// resource; the catalog is the same for every caller.
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

// queryCSV parses a comma-separated query parameter into trimmed, non-empty
// values. Empty entries are ignored; validation of each value belongs to the
// service.
func queryCSV(c *gin.Context, key string) []string {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}
	return values
}

// queryIntList parses a comma-separated integer query parameter.
func queryIntList(c *gin.Context, key string) ([]int, error) {
	values := queryCSV(c, key)
	parsed := make([]int, 0, len(values))
	for _, value := range values {
		number, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, number)
	}
	return parsed, nil
}

// queryDurationRanges parses "min-max,min-max" inclusive duration ranges.
func queryDurationRanges(c *gin.Context, key string) ([]public_programs.DurationRange, error) {
	values := queryCSV(c, key)
	ranges := make([]public_programs.DurationRange, 0, len(values))
	for _, value := range values {
		parts := strings.Split(value, "-")
		if len(parts) != 2 {
			return nil, errors.New("invalid duration range")
		}
		minWeeks, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}
		maxWeeks, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, public_programs.DurationRange{MinWeeks: minWeeks, MaxWeeks: maxWeeks})
	}
	return ranges, nil
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
