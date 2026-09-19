package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/repositories"
	"ryze/backend/services/exercises"
)

// exerciseResponse only exposes safe exercise catalog metadata: the entry
// identity, its name and the public descriptive fields. Deletion markers and
// any internal data are never exposed.
type exerciseResponse struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Description           string    `json:"description"`
	Instructions          string    `json:"instructions"`
	TargetMuscles         string    `json:"target_muscles"`
	PrimaryMuscleGroup    string    `json:"primary_muscle_group"`
	SecondaryMuscleGroups string    `json:"secondary_muscle_groups"`
	Equipment             string    `json:"equipment"`
	Difficulty            string    `json:"difficulty"`
	MovementCategory      string    `json:"movement_category"`
	VideoURL              string    `json:"video_url"`
	ImageURL              string    `json:"image_url"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func newExerciseResponse(exercise *exercises.Exercise) exerciseResponse {
	return exerciseResponse{
		ID:                    exercise.ID,
		Name:                  exercise.Name,
		Description:           exercise.Description,
		Instructions:          exercise.Instructions,
		TargetMuscles:         exercise.TargetMuscles,
		PrimaryMuscleGroup:    exercise.PrimaryMuscleGroup,
		SecondaryMuscleGroups: exercise.SecondaryMuscleGroups,
		Equipment:             exercise.Equipment,
		Difficulty:            exercise.Difficulty,
		MovementCategory:      exercise.MovementCategory,
		VideoURL:              exercise.VideoURL,
		ImageURL:              exercise.ImageURL,
		CreatedAt:             exercise.CreatedAt,
		UpdatedAt:             exercise.UpdatedAt,
	}
}

// alternativeResponse exposes one directed alternative link with the alternative
// name already resolved from the catalog.
type alternativeResponse struct {
	ID                    string `json:"id"`
	ExerciseID            string `json:"exercise_id"`
	AlternativeExerciseID string `json:"alternative_exercise_id"`
	AlternativeName       string `json:"alternative_name"`
}

// ExercisesHandler exposes the public, read-only exercise catalog. These
// endpoints require no authentication and never perform authorization checks:
// the catalog is platform-owned and identical for every caller. No write
// operation is exposed in this foundation.
type ExercisesHandler struct {
	service exercises.Service
}

func NewExercisesHandler(svc exercises.Service) *ExercisesHandler {
	return &ExercisesHandler{service: svc}
}

// ListExercises returns one page of the exercise catalog in alphabetical order,
// optionally narrowed by the muscle, equipment, difficulty and category filters.
func (h *ExercisesHandler) ListExercises(c *gin.Context) {
	page, limit, filter, ok := h.parseListParams(c)
	if !ok {
		return
	}

	result, err := h.service.BrowseExercises(c.Request.Context(), filter, page, limit)
	if err != nil {
		h.respondExercisesError(c, err)
		return
	}

	h.respondExercisePage(c, result)
}

// GetExercise returns one active exercise catalog entry together with its
// curated alternatives. The exercise id in the path only identifies the
// requested resource; the catalog is the same for every caller.
func (h *ExercisesHandler) GetExercise(c *gin.Context) {
	detail, err := h.service.GetExercise(c.Request.Context(), c.Param("exerciseID"))
	if err != nil {
		h.respondExercisesError(c, err)
		return
	}

	alternatives := make([]alternativeResponse, 0, len(detail.Alternatives))
	for i := range detail.Alternatives {
		link := detail.Alternatives[i]
		alternatives = append(alternatives, alternativeResponse{
			ID:                    link.ID,
			ExerciseID:            link.ExerciseID,
			AlternativeExerciseID: link.AlternativeExerciseID,
			AlternativeName:       link.AlternativeName,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Exercise retrieved successfully.",
		"data": gin.H{
			"exercise":     newExerciseResponse(&detail.Exercise),
			"alternatives": alternatives,
		},
	})
}

// SearchExercises returns one page of catalog entries whose name contains the
// search query, case-insensitively, optionally narrowed by the same filters
// accepted by the full list. An empty query is search-specific validation and
// is rejected here before reaching the service.
func (h *ExercisesHandler) SearchExercises(c *gin.Context) {
	page, limit, filter, ok := h.parseListParams(c)
	if !ok {
		return
	}
	filter.Query = c.Query("q")
	if strings.TrimSpace(filter.Query) == "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	result, err := h.service.BrowseExercises(c.Request.Context(), filter, page, limit)
	if err != nil {
		h.respondExercisesError(c, err)
		return
	}

	h.respondExercisePage(c, result)
}

// parseListParams reads the shared pagination and browse-filter query
// parameters. On failure it responds directly and reports that the caller must
// abort.
func (h *ExercisesHandler) parseListParams(c *gin.Context) (int, int, repositories.ExerciseSearchFilter, bool) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return 0, 0, repositories.ExerciseSearchFilter{}, false
	}
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return 0, 0, repositories.ExerciseSearchFilter{}, false
	}

	filter := repositories.ExerciseSearchFilter{
		Muscle:     c.Query("muscle"),
		Equipment:  c.Query("equipment"),
		Difficulty: c.Query("difficulty"),
		Category:   c.Query("category"),
	}
	return page, limit, filter, true
}

func (h *ExercisesHandler) respondExercisePage(c *gin.Context, result exercises.ListExercisesResult) {
	exerciseList := make([]exerciseResponse, 0, len(result.Exercises))
	for i := range result.Exercises {
		exerciseList = append(exerciseList, newExerciseResponse(&result.Exercises[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Exercises retrieved successfully.",
		"data": gin.H{
			"exercises": exerciseList,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// respondExercisesError maps exercises service errors to API responses.
// Internal error details are never exposed to the client.
func (h *ExercisesHandler) respondExercisesError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, exercises.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, exercises.ErrExerciseNotFound):
		RespondError(c, http.StatusNotFound, "EXERCISE_NOT_FOUND", "Exercise not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}