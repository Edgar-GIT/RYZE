package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/workout_sessions"
)

type workoutSessionSetResponse struct {
	ID                    string     `json:"id"`
	SetNumber             int        `json:"set_number"`
	SetType               string     `json:"set_type"`
	PrescribedReps        *int       `json:"prescribed_reps"`
	PrescribedWeightKg    *float64   `json:"prescribed_weight_kg"`
	PrescribedRIR         *int       `json:"prescribed_rir"`
	PrescribedRPE         *float64   `json:"prescribed_rpe"`
	PrescribedRestSeconds *int       `json:"prescribed_rest_seconds"`
	PrescribedTempo       string     `json:"prescribed_tempo"`
	ActualReps            *int       `json:"actual_reps"`
	ActualWeightKg        *float64   `json:"actual_weight_kg"`
	ActualRIR             *int       `json:"actual_rir"`
	ActualRPE             *float64   `json:"actual_rpe"`
	Status                string     `json:"status"`
	CompletedAt           *time.Time `json:"completed_at"`
	RestEndsAt            *time.Time `json:"rest_ends_at"`
}

type workoutSessionExerciseResponse struct {
	ID           string                      `json:"id"`
	ExerciseID   string                      `json:"exercise_id"`
	Position     int                         `json:"position"`
	Name         string                      `json:"name"`
	Instructions string                      `json:"instructions"`
	Notes        string                      `json:"notes"`
	Sets         []workoutSessionSetResponse `json:"sets"`
}

type workoutSessionResponse struct {
	ID               string                           `json:"id"`
	ProgramID        string                           `json:"program_id"`
	ProgramWorkoutID string                           `json:"program_workout_id"`
	ProgramName      string                           `json:"program_name"`
	WeekNumber       int                              `json:"week_number"`
	WorkoutPosition  int                              `json:"workout_position"`
	Status           string                           `json:"status"`
	StartedAt        time.Time                        `json:"started_at"`
	PausedAt         *time.Time                       `json:"paused_at"`
	CompletedAt      *time.Time                       `json:"completed_at"`
	AbandonedAt      *time.Time                       `json:"abandoned_at"`
	ElapsedSeconds   int                              `json:"elapsed_seconds"`
	TotalSets        int                              `json:"total_sets"`
	CompletedSets    int                              `json:"completed_sets"`
	SkippedSets      int                              `json:"skipped_sets"`
	Exercises        []workoutSessionExerciseResponse `json:"exercises"`
}

type updateWorkoutSetRequest struct {
	ActualReps     *int     `json:"actual_reps"`
	ActualWeightKg *float64 `json:"actual_weight_kg"`
	ActualRIR      *int     `json:"actual_rir"`
	ActualRPE      *float64 `json:"actual_rpe"`
	Status         string   `json:"status"`
}

func newWorkoutSessionResponse(session *workout_sessions.Session) workoutSessionResponse {
	response := workoutSessionResponse{
		ID:               session.ID,
		ProgramID:        session.ProgramID,
		ProgramWorkoutID: session.ProgramWorkoutID,
		ProgramName:      session.ProgramName,
		WeekNumber:       session.WeekNumber,
		WorkoutPosition:  session.WorkoutPosition,
		Status:           session.Status,
		StartedAt:        session.StartedAt,
		PausedAt:         session.PausedAt,
		CompletedAt:      session.CompletedAt,
		AbandonedAt:      session.AbandonedAt,
		ElapsedSeconds:   session.ElapsedSeconds,
		TotalSets:        session.TotalSets,
		CompletedSets:    session.CompletedSets,
		SkippedSets:      session.SkippedSets,
		Exercises:        make([]workoutSessionExerciseResponse, 0, len(session.Exercises)),
	}
	for i := range session.Exercises {
		exercise := session.Exercises[i]
		exerciseResponse := workoutSessionExerciseResponse{
			ID:           exercise.ID,
			ExerciseID:   exercise.ExerciseID,
			Position:     exercise.Position,
			Name:         exercise.Name,
			Instructions: exercise.Instructions,
			Notes:        exercise.Notes,
			Sets:         make([]workoutSessionSetResponse, 0, len(exercise.Sets)),
		}
		for k := range exercise.Sets {
			set := exercise.Sets[k]
			exerciseResponse.Sets = append(exerciseResponse.Sets, workoutSessionSetResponse{
				ID:                    set.ID,
				SetNumber:             set.SetNumber,
				SetType:               set.SetType,
				PrescribedReps:        set.PrescribedReps,
				PrescribedWeightKg:    set.PrescribedWeightKg,
				PrescribedRIR:         set.PrescribedRIR,
				PrescribedRPE:         set.PrescribedRPE,
				PrescribedRestSeconds: set.PrescribedRestSeconds,
				PrescribedTempo:       set.PrescribedTempo,
				ActualReps:            set.ActualReps,
				ActualWeightKg:        set.ActualWeightKg,
				ActualRIR:             set.ActualRIR,
				ActualRPE:             set.ActualRPE,
				Status:                set.Status,
				CompletedAt:           set.CompletedAt,
				RestEndsAt:            set.RestEndsAt,
			})
		}
		response.Exercises = append(response.Exercises, exerciseResponse)
	}
	return response
}

// WorkoutSessionsHandler exposes the authenticated workout execution lifecycle.
type WorkoutSessionsHandler struct {
	service workout_sessions.Service
}

func NewWorkoutSessionsHandler(svc workout_sessions.Service) *WorkoutSessionsHandler {
	return &WorkoutSessionsHandler{service: svc}
}

func (h *WorkoutSessionsHandler) Start(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	session, err := h.service.Start(c.Request.Context(), userID, c.Param("programID"), c.Param("workoutID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout session started successfully.",
		"data":    newWorkoutSessionResponse(session),
	})
}

func (h *WorkoutSessionsHandler) Active(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	session, err := h.service.Active(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	var data any
	if session != nil {
		data = newWorkoutSessionResponse(session)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Active workout session retrieved successfully.",
		"data":    data,
	})
}

func (h *WorkoutSessionsHandler) Get(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	session, err := h.service.Get(c.Request.Context(), userID, c.Param("sessionID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout session retrieved successfully.",
		"data":    newWorkoutSessionResponse(session),
	})
}

func (h *WorkoutSessionsHandler) UpdateSet(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var request updateWorkoutSetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	session, err := h.service.UpdateSet(c.Request.Context(), userID, c.Param("sessionID"), c.Param("setID"), workout_sessions.SetInput{
		ActualReps:     request.ActualReps,
		ActualWeightKg: request.ActualWeightKg,
		ActualRIR:      request.ActualRIR,
		ActualRPE:      request.ActualRPE,
		Status:         request.Status,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout set updated successfully.",
		"data":    newWorkoutSessionResponse(session),
	})
}

func (h *WorkoutSessionsHandler) Pause(c *gin.Context) {
	h.transition(c, h.service.Pause, "Workout session paused successfully.")
}

func (h *WorkoutSessionsHandler) Resume(c *gin.Context) {
	h.transition(c, h.service.Resume, "Workout session resumed successfully.")
}

func (h *WorkoutSessionsHandler) Complete(c *gin.Context) {
	h.transition(c, h.service.Complete, "Workout session completed successfully.")
}

func (h *WorkoutSessionsHandler) Abandon(c *gin.Context) {
	h.transition(c, h.service.Abandon, "Workout session abandoned successfully.")
}

type sessionTransition func(context.Context, string, string) (*workout_sessions.Session, error)

func (h *WorkoutSessionsHandler) transition(c *gin.Context, fn sessionTransition, message string) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	session, err := fn(c.Request.Context(), userID, c.Param("sessionID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"data":    newWorkoutSessionResponse(session),
	})
}

func (h *WorkoutSessionsHandler) History(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	sessions, err := h.service.History(c.Request.Context(), userID, limit)
	if err != nil {
		h.respondError(c, err)
		return
	}
	responses := make([]workoutSessionResponse, 0, len(sessions))
	for i := range sessions {
		responses = append(responses, newWorkoutSessionResponse(&sessions[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout history retrieved successfully.",
		"data":    responses,
	})
}

func (h *WorkoutSessionsHandler) userID(c *gin.Context) (string, bool) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return "", false
	}
	return userID, true
}

func (h *WorkoutSessionsHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, workout_sessions.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, workout_sessions.ErrWorkoutNotFound):
		RespondError(c, http.StatusNotFound, "WORKOUT_NOT_FOUND", "Workout not found.", nil)
	case errors.Is(err, workout_sessions.ErrSessionNotFound):
		RespondError(c, http.StatusNotFound, "WORKOUT_SESSION_NOT_FOUND", "Workout session not found.", nil)
	case errors.Is(err, workout_sessions.ErrSessionConflict):
		RespondError(c, http.StatusConflict, "WORKOUT_SESSION_ACTIVE", "Another workout session is already active.", nil)
	case errors.Is(err, workout_sessions.ErrSessionNotActive):
		RespondError(c, http.StatusConflict, "WORKOUT_SESSION_NOT_ACTIVE", "Workout session is not active.", nil)
	case errors.Is(err, workout_sessions.ErrSetNotFound):
		RespondError(c, http.StatusNotFound, "WORKOUT_SET_NOT_FOUND", "Workout set not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
