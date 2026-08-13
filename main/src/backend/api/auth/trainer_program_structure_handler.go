package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/program_structure"
)

// trainerWeekResponse only exposes safe week structure: the week identity, its
// owning program id, the week number and the week's workouts in order. Deletion
// markers and any internal data are never exposed.
type trainerWeekResponse struct {
	ID         string                   `json:"id"`
	ProgramID  string                   `json:"program_id"`
	WeekNumber int                      `json:"week_number"`
	Workouts   []trainerWorkoutResponse `json:"workouts"`
	CreatedAt  time.Time                `json:"created_at"`
	UpdatedAt  time.Time                `json:"updated_at"`
}

func newTrainerWeekResponse(week *program_structure.Week) trainerWeekResponse {
	workouts := make([]trainerWorkoutResponse, 0, len(week.Workouts))
	for i := range week.Workouts {
		workouts = append(workouts, newTrainerWorkoutResponse(&week.Workouts[i]))
	}
	return trainerWeekResponse{
		ID:         week.ID,
		ProgramID:  week.ProgramID,
		WeekNumber: week.WeekNumber,
		Workouts:   workouts,
		CreatedAt:  week.CreatedAt,
		UpdatedAt:  week.UpdatedAt,
	}
}

// trainerWorkoutResponse only exposes safe workout structure: the workout
// identity, its owning week id and its position. Deletion markers and any
// internal data are never exposed.
type trainerWorkoutResponse struct {
	ID            string    `json:"id"`
	ProgramWeekID string    `json:"program_week_id"`
	Position      int       `json:"position"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func newTrainerWorkoutResponse(workout *program_structure.Workout) trainerWorkoutResponse {
	return trainerWorkoutResponse{
		ID:            workout.ID,
		ProgramWeekID: workout.ProgramWeekID,
		Position:      workout.Position,
		CreatedAt:     workout.CreatedAt,
		UpdatedAt:     workout.UpdatedAt,
	}
}

// reorderRequest carries the ordered list of identifiers for a reorder
// operation. It never accepts trainer, program or week identities: those always
// come from the authenticated trainer context and the URL path.
type reorderRequest struct {
	IDs []string `json:"ids"`
}

// TrainerProgramStructureHandler exposes the authenticated trainer's program
// structure management operations (weeks and workouts). It never performs
// authentication or authorization itself: those are enforced by the
// Authenticate, TrainerAuthenticate and RequireTrainerPermission middleware
// mounted on the route. The trainer identity always comes exclusively from the
// trainer context; request parameters, body, headers or any client-supplied
// identity can never influence which program, week or workout is created,
// listed, read, reordered or deleted.
type TrainerProgramStructureHandler struct {
	service program_structure.Service
}

func NewTrainerProgramStructureHandler(svc program_structure.Service) *TrainerProgramStructureHandler {
	return &TrainerProgramStructureHandler{service: svc}
}

// CreateWeek appends a new empty week to one of the authenticated trainer's own
// programs.
func (h *TrainerProgramStructureHandler) CreateWeek(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	week, err := h.service.CreateWeek(c.Request.Context(), trainerID, c.Param("programID"))
	if err != nil {
		h.respondStructureError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Week created successfully.",
		"data":    newTrainerWeekResponse(week),
	})
}

// ListWeeks returns every active week of one of the authenticated trainer's own
// programs, in week order, with each week's workouts in position order.
func (h *TrainerProgramStructureHandler) ListWeeks(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	weeks, err := h.service.ListWeeks(c.Request.Context(), trainerID, c.Param("programID"))
	if err != nil {
		h.respondStructureError(c, err)
		return
	}

	response := make([]trainerWeekResponse, 0, len(weeks))
	for i := range weeks {
		response = append(response, newTrainerWeekResponse(&weeks[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Weeks retrieved successfully.",
		"data": gin.H{
			"weeks": response,
		},
	})
}

// GetWeek returns one active week of one of the authenticated trainer's own
// programs, with its workouts in position order.
func (h *TrainerProgramStructureHandler) GetWeek(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	week, err := h.service.GetWeek(c.Request.Context(), trainerID, c.Param("programID"), c.Param("weekID"))
	if err != nil {
		h.respondStructureError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Week retrieved successfully.",
		"data":    newTrainerWeekResponse(week),
	})
}

// ReorderWeeks replaces the order of every active week of one of the
// authenticated trainer's own programs with the order of the ids provided in
// the body.
func (h *TrainerProgramStructureHandler) ReorderWeeks(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	if err := h.service.ReorderWeeks(c.Request.Context(), trainerID, c.Param("programID"), req.IDs); err != nil {
		h.respondStructureError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Weeks reordered successfully.",
		"data":    gin.H{},
	})
}

// DeleteWeek soft-deletes one of the authenticated trainer's own program weeks.
func (h *TrainerProgramStructureHandler) DeleteWeek(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	if err := h.service.DeleteWeek(c.Request.Context(), trainerID, c.Param("programID"), c.Param("weekID")); err != nil {
		h.respondStructureError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Week deleted successfully.",
		"data":    gin.H{},
	})
}

// CreateWorkout appends a new empty workout to one of the authenticated
// trainer's own program weeks.
func (h *TrainerProgramStructureHandler) CreateWorkout(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	workout, err := h.service.CreateWorkout(c.Request.Context(), trainerID, c.Param("programID"), c.Param("weekID"))
	if err != nil {
		h.respondStructureError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Workout created successfully.",
		"data":    newTrainerWorkoutResponse(workout),
	})
}

// ListWorkouts returns every active workout of one of the authenticated
// trainer's own program weeks, in position order.
func (h *TrainerProgramStructureHandler) ListWorkouts(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	workouts, err := h.service.ListWorkouts(c.Request.Context(), trainerID, c.Param("programID"), c.Param("weekID"))
	if err != nil {
		h.respondStructureError(c, err)
		return
	}

	response := make([]trainerWorkoutResponse, 0, len(workouts))
	for i := range workouts {
		response = append(response, newTrainerWorkoutResponse(&workouts[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workouts retrieved successfully.",
		"data": gin.H{
			"workouts": response,
		},
	})
}

// GetWorkout returns one active workout of one of the authenticated trainer's
// own program weeks.
func (h *TrainerProgramStructureHandler) GetWorkout(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	workout, err := h.service.GetWorkout(c.Request.Context(), trainerID, c.Param("programID"), c.Param("weekID"), c.Param("workoutID"))
	if err != nil {
		h.respondStructureError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout retrieved successfully.",
		"data":    newTrainerWorkoutResponse(workout),
	})
}

// ReorderWorkouts replaces the order of every active workout of one of the
// authenticated trainer's own program weeks with the order of the ids provided
// in the body.
func (h *TrainerProgramStructureHandler) ReorderWorkouts(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	if err := h.service.ReorderWorkouts(c.Request.Context(), trainerID, c.Param("programID"), c.Param("weekID"), req.IDs); err != nil {
		h.respondStructureError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workouts reordered successfully.",
		"data":    gin.H{},
	})
}

// DeleteWorkout soft-deletes one of the authenticated trainer's own program
// workouts.
func (h *TrainerProgramStructureHandler) DeleteWorkout(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	if err := h.service.DeleteWorkout(c.Request.Context(), trainerID, c.Param("programID"), c.Param("weekID"), c.Param("workoutID")); err != nil {
		h.respondStructureError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Workout deleted successfully.",
		"data":    gin.H{},
	})
}

// respondStructureError maps program structure service errors to API responses.
// Internal error details are never exposed to the client.
func (h *TrainerProgramStructureHandler) respondStructureError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, program_structure.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, program_structure.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	case errors.Is(err, program_structure.ErrWeekNotFound):
		RespondError(c, http.StatusNotFound, "WEEK_NOT_FOUND", "Week not found.", nil)
	case errors.Is(err, program_structure.ErrWorkoutNotFound):
		RespondError(c, http.StatusNotFound, "WORKOUT_NOT_FOUND", "Workout not found.", nil)
	case errors.Is(err, program_structure.ErrReorderConflict):
		RespondError(c, http.StatusConflict, "REORDER_CONFLICT", "The order list does not match the current entries.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
