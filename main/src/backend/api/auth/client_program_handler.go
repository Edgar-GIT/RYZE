package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/client_programs"
)

// clientProgramResponse only exposes safe assigned-program information: the
// public product metadata, the lifecycle timestamps and the full active
// structure. The owning trainer, parent identifiers, deletion markers and any
// internal data are never exposed.
type clientProgramResponse struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	Type            string               `json:"type"`
	Status          string               `json:"status"`
	PriceMinorUnits int64                `json:"price_minor_units"`
	Currency        string               `json:"currency"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Weeks           []clientWeekResponse `json:"weeks"`
}

// clientWeekResponse is the safe view of one week of the assigned program.
type clientWeekResponse struct {
	ID         string                  `json:"id"`
	WeekNumber int                     `json:"week_number"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  time.Time               `json:"updated_at"`
	Workouts   []clientWorkoutResponse `json:"workouts"`
}

// clientWorkoutResponse is the safe view of one workout of the assigned
// program.
type clientWorkoutResponse struct {
	ID        string                          `json:"id"`
	Position  int                             `json:"position"`
	CreatedAt time.Time                       `json:"created_at"`
	UpdatedAt time.Time                       `json:"updated_at"`
	Exercises []clientWorkoutExerciseResponse `json:"exercises"`
}

// clientWorkoutExerciseResponse is the safe view of one exercise usage inside
// an assigned workout, reusing the safe public exercise summary.
type clientWorkoutExerciseResponse struct {
	ID        string                 `json:"id"`
	Position  int                    `json:"position"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Exercise  trainerExerciseSummary `json:"exercise"`
}

func newClientProgramResponse(program *client_programs.Program) clientProgramResponse {
	response := clientProgramResponse{
		ID:              program.ID,
		Name:            program.Name,
		Description:     program.Description,
		Type:            program.Type,
		Status:          program.Status,
		PriceMinorUnits: program.PriceMinorUnits,
		Currency:        program.Currency,
		CreatedAt:       program.CreatedAt,
		UpdatedAt:       program.UpdatedAt,
		Weeks:           make([]clientWeekResponse, 0, len(program.Weeks)),
	}
	for i := range program.Weeks {
		response.Weeks = append(response.Weeks, newClientWeekResponse(&program.Weeks[i]))
	}
	return response
}

func newClientWeekResponse(week *client_programs.Week) clientWeekResponse {
	response := clientWeekResponse{
		ID:         week.ID,
		WeekNumber: week.WeekNumber,
		CreatedAt:  week.CreatedAt,
		UpdatedAt:  week.UpdatedAt,
		Workouts:   make([]clientWorkoutResponse, 0, len(week.Workouts)),
	}
	for i := range week.Workouts {
		response.Workouts = append(response.Workouts, newClientWorkoutResponse(&week.Workouts[i]))
	}
	return response
}

func newClientWorkoutResponse(workout *client_programs.Workout) clientWorkoutResponse {
	response := clientWorkoutResponse{
		ID:        workout.ID,
		Position:  workout.Position,
		CreatedAt: workout.CreatedAt,
		UpdatedAt: workout.UpdatedAt,
		Exercises: make([]clientWorkoutExerciseResponse, 0, len(workout.Exercises)),
	}
	for i := range workout.Exercises {
		response.Exercises = append(response.Exercises, newClientWorkoutExerciseResponse(&workout.Exercises[i]))
	}
	return response
}

func newClientWorkoutExerciseResponse(workoutExercise *client_programs.WorkoutExercise) clientWorkoutExerciseResponse {
	return clientWorkoutExerciseResponse{
		ID:        workoutExercise.ID,
		Position:  workoutExercise.Position,
		CreatedAt: workoutExercise.CreatedAt,
		UpdatedAt: workoutExercise.UpdatedAt,
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
	}
}

// ClientProgramHandler exposes the authenticated client's assigned program read
// operation. It never performs authentication or authorization itself: those
// are enforced by the Authenticate middleware mounted on the route. The user
// identity always comes exclusively from the authentication context; query
// parameters, body, headers or any client-supplied identity can never influence
// which program is returned.
type ClientProgramHandler struct {
	service client_programs.Service
}

func NewClientProgramHandler(svc client_programs.Service) *ClientProgramHandler {
	return &ClientProgramHandler{service: svc}
}

// GetProgram returns the full active structure of the program currently
// assigned to the authenticated user. A client-supplied user_id or trainer_id
// is deliberately ignored: the requested program is always resolved from the
// authentication context.
func (h *ClientProgramHandler) GetProgram(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	program, err := h.service.GetProgram(c.Request.Context(), userID)
	if err != nil {
		h.respondProgramError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Assigned program retrieved successfully.",
		"data":    newClientProgramResponse(program),
	})
}

// respondProgramError maps client program service errors to API responses.
// Internal error details are never exposed to the client.
func (h *ClientProgramHandler) respondProgramError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, client_programs.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, client_programs.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
