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
// program identity, its owner trainer id, the public product fields, the
// generic audience metadata and the lifecycle timestamps. Deletion markers,
// draft programs and any internal data are never exposed.
type publicProgramResponse struct {
	ID               string    `json:"id"`
	TrainerID        string    `json:"trainer_id"`
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
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func newPublicProgramResponse(program *public_programs.Program) publicProgramResponse {
	return publicProgramResponse{
		ID:               program.ID,
		TrainerID:        program.TrainerID,
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
		CreatedAt:        program.CreatedAt,
		UpdatedAt:        program.UpdatedAt,
	}
}

// publicSetResponse exposes one public prescription set.
type publicSetResponse struct {
	SetNumber   int      `json:"set_number"`
	SetType     string   `json:"set_type"`
	Reps        *int     `json:"reps"`
	WeightKg    *float64 `json:"weight_kg"`
	RIR         *int     `json:"rir"`
	RPE         *float64 `json:"rpe"`
	RestSeconds *int     `json:"rest_seconds"`
	Tempo       string   `json:"tempo"`
}

// publicExerciseResponse exposes one public exercise usage with its catalog
// summary and prescription sets. Internal identifiers are never exposed.
type publicExerciseResponse struct {
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	Instructions  string              `json:"instructions"`
	TargetMuscles string              `json:"target_muscles"`
	Equipment     string              `json:"equipment"`
	Difficulty    string              `json:"difficulty"`
	VideoURL      string              `json:"video_url"`
	ImageURL      string              `json:"image_url"`
	Position      int                 `json:"position"`
	Notes         string              `json:"notes"`
	Sets          []publicSetResponse `json:"sets"`
}

type publicWorkoutResponse struct {
	Position  int                     `json:"position"`
	Exercises []publicExerciseResponse `json:"exercises"`
}

type publicWeekResponse struct {
	WeekNumber int                     `json:"week_number"`
	Workouts   []publicWorkoutResponse `json:"workouts"`
}

func newPublicWeekResponse(week public_programs.Week) publicWeekResponse {
	workouts := make([]publicWorkoutResponse, 0, len(week.Workouts))
	for i := range week.Workouts {
		workout := week.Workouts[i]
		exercises := make([]publicExerciseResponse, 0, len(workout.Exercises))
		for k := range workout.Exercises {
			exercise := workout.Exercises[k]
			sets := make([]publicSetResponse, 0, len(exercise.Sets))
			for j := range exercise.Sets {
				set := exercise.Sets[j]
				sets = append(sets, publicSetResponse{
					SetNumber:   set.SetNumber,
					SetType:     set.SetType,
					Reps:        set.Reps,
					WeightKg:    set.WeightKg,
					RIR:         set.RIR,
					RPE:         set.RPE,
					RestSeconds: set.RestSeconds,
					Tempo:       set.Tempo,
				})
			}
			exercises = append(exercises, publicExerciseResponse{
				Name:          exercise.Name,
				Description:   exercise.Description,
				Instructions:  exercise.Instructions,
				TargetMuscles: exercise.TargetMuscles,
				Equipment:     exercise.Equipment,
				Difficulty:    exercise.Difficulty,
				VideoURL:      exercise.VideoURL,
				ImageURL:      exercise.ImageURL,
				Position:      exercise.Position,
				Notes:         exercise.Notes,
				Sets:          sets,
			})
		}
		workouts = append(workouts, publicWorkoutResponse{Position: workout.Position, Exercises: exercises})
	}
	return publicWeekResponse{WeekNumber: week.WeekNumber, Workouts: workouts}
}

func newPublicProgramDetailResponse(detail *public_programs.ProgramDetail) gin.H {
	weeks := make([]publicWeekResponse, 0, len(detail.Weeks))
	for i := range detail.Weeks {
		weeks = append(weeks, newPublicWeekResponse(detail.Weeks[i]))
	}
	return gin.H{
		"id":                detail.ID,
		"trainer_id":        detail.TrainerID,
		"name":              detail.Name,
		"description":       detail.Description,
		"type":              detail.Type,
		"status":            detail.Status,
		"level":             detail.Level,
		"duration_weeks":    detail.DurationWeeks,
		"frequency_per_week": detail.FrequencyPerWeek,
		"training_type":     detail.TrainingType,
		"price_minor_units": detail.PriceMinorUnits,
		"currency":          detail.Currency,
		"created_at":        detail.CreatedAt,
		"updated_at":        detail.UpdatedAt,
		"weeks":             weeks,
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
	frequency, err := queryInt(c, "frequency", 0)
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

	filter := public_programs.ProgramFilter{
		Query:            strings.TrimSpace(c.Query("q")),
		ProgramType:      strings.TrimSpace(c.Query("type")),
		TrainingType:     strings.TrimSpace(c.Query("training_type")),
		Level:            strings.TrimSpace(c.Query("level")),
		FrequencyPerWeek: frequency,
		DurationMin:      durationMin,
		DurationMax:      durationMax,
		SortBy:           strings.TrimSpace(c.Query("sort")),
		Order:            strings.TrimSpace(c.Query("order")),
		ScopeGeneric:     strings.TrimSpace(c.Query("scope")) == "generic",
	}

	hasSearchParams := filter.Query != "" || filter.ProgramType != "" || filter.TrainingType != "" ||
		filter.Level != "" || filter.FrequencyPerWeek > 0 || filter.DurationMin > 0 || filter.DurationMax > 0 ||
		filter.SortBy != "" || filter.Order != "" || filter.ScopeGeneric

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
		"data":    newPublicProgramDetailResponse(program),
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
