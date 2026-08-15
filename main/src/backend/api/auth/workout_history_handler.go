package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/workout_history"
)

// workoutHistoryEntryResponse is the safe view of one completed workout. It
// exposes only the completed workout reference and the lifecycle timestamps; the
// owning user id, parent identifiers and deletion markers are never exposed.
type workoutHistoryEntryResponse struct {
	ID               string    `json:"id"`
	ProgramWorkoutID string    `json:"program_workout_id"`
	CompletedAt      time.Time `json:"completed_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func newWorkoutHistoryEntryResponse(entry *workout_history.HistoryEntry) workoutHistoryEntryResponse {
	return workoutHistoryEntryResponse{
		ID:               entry.ID,
		ProgramWorkoutID: entry.ProgramWorkoutID,
		CompletedAt:      entry.CompletedAt,
		CreatedAt:        entry.CreatedAt,
		UpdatedAt:        entry.UpdatedAt,
	}
}

// WorkoutHistoryHandler exposes the authenticated client's workout completion
// and history read operations. It never performs authentication or
// authorization itself: those are enforced by the Authenticate middleware
// mounted on the route. The user identity always comes exclusively from the
// authentication context; query parameters, body, headers or any client-supplied
// identity can never influence which workout is completed or which history is
// returned.
type WorkoutHistoryHandler struct {
	service workout_history.Service
}

func NewWorkoutHistoryHandler(svc workout_history.Service) *WorkoutHistoryHandler {
	return &WorkoutHistoryHandler{service: svc}
}

// CompleteWorkout records that the authenticated user completed one workout of
// their currently assigned program. The workout id in the path only identifies
// the requested resource and never proves access: the workout is only
// executable when it belongs to the active structure of the assigned program. A
// client-supplied user_id, client_id or trainer_id is deliberately ignored.
func (h *WorkoutHistoryHandler) CompleteWorkout(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	entry, err := h.service.CompleteWorkout(c.Request.Context(), userID, c.Param("workoutID"))
	if err != nil {
		h.respondHistoryError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Workout completed successfully.",
		"data":    newWorkoutHistoryEntryResponse(entry),
	})
}

// ListHistory returns one page of the authenticated user's own completed
// workouts, newest first. A client-supplied user_id is deliberately ignored: the
// history is always scoped to the identity from the authentication context.
func (h *WorkoutHistoryHandler) ListHistory(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
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

	result, err := h.service.ListHistory(c.Request.Context(), userID, page, limit)
	if err != nil {
		h.respondHistoryError(c, err)
		return
	}

	entries := make([]workoutHistoryEntryResponse, 0, len(result.Entries))
	for i := range result.Entries {
		entries = append(entries, newWorkoutHistoryEntryResponse(&result.Entries[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout history retrieved successfully.",
		"data": gin.H{
			"entries": entries,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// respondHistoryError maps workout history service errors to API responses.
// Internal error details are never exposed to the client.
func (h *WorkoutHistoryHandler) respondHistoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, workout_history.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, workout_history.ErrWorkoutNotFound):
		RespondError(c, http.StatusNotFound, "WORKOUT_NOT_FOUND", "Workout not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
