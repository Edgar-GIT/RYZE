package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/exercises"
)

// exerciseResponse only exposes safe exercise catalog metadata: the entry
// identity, its name and the public descriptive fields. Deletion markers and
// any internal data are never exposed.
type exerciseResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	TargetMuscles string    `json:"target_muscles"`
	Equipment     string    `json:"equipment"`
	Difficulty    string    `json:"difficulty"`
	VideoURL      string    `json:"video_url"`
	ImageURL      string    `json:"image_url"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func newExerciseResponse(exercise *exercises.Exercise) exerciseResponse {
	return exerciseResponse{
		ID:            exercise.ID,
		Name:          exercise.Name,
		Description:   exercise.Description,
		TargetMuscles: exercise.TargetMuscles,
		Equipment:     exercise.Equipment,
		Difficulty:    exercise.Difficulty,
		VideoURL:      exercise.VideoURL,
		ImageURL:      exercise.ImageURL,
		CreatedAt:     exercise.CreatedAt,
		UpdatedAt:     exercise.UpdatedAt,
	}
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

// ListExercises returns one page of the exercise catalog in alphabetical order.
func (h *ExercisesHandler) ListExercises(c *gin.Context) {
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

	result, err := h.service.ListExercises(c.Request.Context(), page, limit)
	if err != nil {
		h.respondExercisesError(c, err)
		return
	}

	exercises := make([]exerciseResponse, 0, len(result.Exercises))
	for i := range result.Exercises {
		exercises = append(exercises, newExerciseResponse(&result.Exercises[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Exercises retrieved successfully.",
		"data": gin.H{
			"exercises": exercises,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// GetExercise returns one active exercise catalog entry. The exercise id in
// the path only identifies the requested resource; the catalog is the same for
// every caller.
func (h *ExercisesHandler) GetExercise(c *gin.Context) {
	exercise, err := h.service.GetExercise(c.Request.Context(), c.Param("exerciseID"))
	if err != nil {
		h.respondExercisesError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Exercise retrieved successfully.",
		"data":    newExerciseResponse(exercise),
	})
}

// SearchExercises returns one page of catalog entries whose name contains the
// search query, case-insensitively.
func (h *ExercisesHandler) SearchExercises(c *gin.Context) {
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

	result, err := h.service.SearchExercises(c.Request.Context(), c.Query("q"), page, limit)
	if err != nil {
		h.respondExercisesError(c, err)
		return
	}

	exercises := make([]exerciseResponse, 0, len(result.Exercises))
	for i := range result.Exercises {
		exercises = append(exercises, newExerciseResponse(&result.Exercises[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Exercises retrieved successfully.",
		"data": gin.H{
			"exercises": exercises,
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
