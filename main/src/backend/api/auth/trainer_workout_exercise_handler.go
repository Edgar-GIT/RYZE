package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/workout_exercises"
)

// trainerWorkoutExerciseResponse only exposes safe workout exercise structure:
// the assignment identity, its owning workout id, its position and the safe
// catalog data of the assigned exercise. Deletion markers and any internal data
// are never exposed.
type trainerWorkoutExerciseResponse struct {
	ID               string                 `json:"id"`
	ProgramWorkoutID string                 `json:"program_workout_id"`
	Position         int                    `json:"position"`
	Exercise         trainerExerciseSummary `json:"exercise"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// trainerExerciseSummary is the safe public view of the global catalog
// exercise assigned to a workout. It carries only the descriptive metadata
// exposed by the public catalog and never exposes internal data.
type trainerExerciseSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	TargetMuscles string `json:"target_muscles"`
	Equipment     string `json:"equipment"`
	Difficulty    string `json:"difficulty"`
	VideoURL      string `json:"video_url"`
	ImageURL      string `json:"image_url"`
}

func newTrainerWorkoutExerciseResponse(workoutExercise *workout_exercises.WorkoutExercise) trainerWorkoutExerciseResponse {
	return trainerWorkoutExerciseResponse{
		ID:               workoutExercise.ID,
		ProgramWorkoutID: workoutExercise.ProgramWorkoutID,
		Position:         workoutExercise.Position,
		Exercise: trainerExerciseSummary{
			ID:            workoutExercise.Exercise.ID,
			Name:          workoutExercise.Exercise.Name,
			Description:   workoutExercise.Exercise.Description,
			TargetMuscles: workoutExercise.Exercise.TargetMuscles,
			Equipment:     workoutExercise.Exercise.Equipment,
			Difficulty:    workoutExercise.Exercise.Difficulty,
			VideoURL:      workoutExercise.Exercise.VideoURL,
			ImageURL:      workoutExercise.Exercise.ImageURL,
		},
		CreatedAt: workoutExercise.CreatedAt,
		UpdatedAt: workoutExercise.UpdatedAt,
	}
}

// addWorkoutExerciseRequest is the request DTO for POST
// /api/v1/trainer/programs/:programID/weeks/:weekID/workouts/:workoutID/exercises.
// It only accepts the global catalog exercise id; the workout id and the
// trainer ownership never come from the body.
type addWorkoutExerciseRequest struct {
	ExerciseID string `json:"exercise_id"`
}

// TrainerWorkoutExerciseHandler exposes the authenticated trainer's workout
// exercise management operations. It never performs authentication or
// authorization itself: those are enforced by the Authenticate,
// TrainerAuthenticate and RequireTrainerPermission middleware mounted on the
// route. The trainer identity always comes exclusively from the trainer
// context; request parameters, body, headers or any client-supplied identity
// can never influence which program, week, workout or workout exercise is
// created, listed, read, reordered or deleted.
type TrainerWorkoutExerciseHandler struct {
	service workout_exercises.Service
}

func NewTrainerWorkoutExerciseHandler(svc workout_exercises.Service) *TrainerWorkoutExerciseHandler {
	return &TrainerWorkoutExerciseHandler{service: svc}
}

// AddExercise assigns one active catalog exercise to one of the authenticated
// trainer's own program workouts. The workout id in the path is the only source
// of truth; a client-supplied workout_id in the body is never accepted.
func (h *TrainerWorkoutExerciseHandler) AddExercise(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	var req addWorkoutExerciseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	workoutExercise, err := h.service.AddExercise(c.Request.Context(), trainerID,
		c.Param("programID"), c.Param("weekID"), c.Param("workoutID"), req.ExerciseID)
	if err != nil {
		h.respondWorkoutExerciseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Exercise added to workout successfully.",
		"data":    newTrainerWorkoutExerciseResponse(workoutExercise),
	})
}

// ListExercises returns every active workout exercise of one of the
// authenticated trainer's own program workouts, in position order, each with
// its safe exercise data.
func (h *TrainerWorkoutExerciseHandler) ListExercises(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	exercises, err := h.service.ListExercises(c.Request.Context(), trainerID,
		c.Param("programID"), c.Param("weekID"), c.Param("workoutID"))
	if err != nil {
		h.respondWorkoutExerciseError(c, err)
		return
	}

	response := make([]trainerWorkoutExerciseResponse, 0, len(exercises))
	for i := range exercises {
		response = append(response, newTrainerWorkoutExerciseResponse(&exercises[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout exercises retrieved successfully.",
		"data": gin.H{
			"exercises": response,
		},
	})
}

// GetExercise returns one active workout exercise of one of the authenticated
// trainer's own program workouts.
func (h *TrainerWorkoutExerciseHandler) GetExercise(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	workoutExercise, err := h.service.GetExercise(c.Request.Context(), trainerID,
		c.Param("programID"), c.Param("weekID"), c.Param("workoutID"), c.Param("workoutExerciseID"))
	if err != nil {
		h.respondWorkoutExerciseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout exercise retrieved successfully.",
		"data":    newTrainerWorkoutExerciseResponse(workoutExercise),
	})
}

// ReorderExercises replaces the order of every active workout exercise of one
// of the authenticated trainer's own program workouts with the order of the ids
// provided in the body.
func (h *TrainerWorkoutExerciseHandler) ReorderExercises(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	if err := h.service.ReorderExercises(c.Request.Context(), trainerID,
		c.Param("programID"), c.Param("weekID"), c.Param("workoutID"), req.IDs); err != nil {
		h.respondWorkoutExerciseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout exercises reordered successfully.",
		"data":    gin.H{},
	})
}

// DeleteExercise soft-deletes one of the authenticated trainer's own workout
// exercises.
func (h *TrainerWorkoutExerciseHandler) DeleteExercise(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	if err := h.service.RemoveExercise(c.Request.Context(), trainerID,
		c.Param("programID"), c.Param("weekID"), c.Param("workoutID"), c.Param("workoutExerciseID")); err != nil {
		h.respondWorkoutExerciseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout exercise removed successfully.",
		"data":    gin.H{},
	})
}

// respondWorkoutExerciseError maps workout exercise service errors to API
// responses. Internal error details are never exposed to the client.
func (h *TrainerWorkoutExerciseHandler) respondWorkoutExerciseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, workout_exercises.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, workout_exercises.ErrWorkoutNotFound):
		RespondError(c, http.StatusNotFound, "WORKOUT_NOT_FOUND", "Workout not found.", nil)
	case errors.Is(err, workout_exercises.ErrExerciseNotFound):
		RespondError(c, http.StatusNotFound, "EXERCISE_NOT_FOUND", "Exercise not found.", nil)
	case errors.Is(err, workout_exercises.ErrWorkoutExerciseNotFound):
		RespondError(c, http.StatusNotFound, "WORKOUT_EXERCISE_NOT_FOUND", "Workout exercise not found.", nil)
	case errors.Is(err, workout_exercises.ErrReorderConflict):
		RespondError(c, http.StatusConflict, "REORDER_CONFLICT", "The order list does not match the current entries.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
