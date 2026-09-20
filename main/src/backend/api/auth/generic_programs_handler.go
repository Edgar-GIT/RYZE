package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/generic_programs"
)

// genericProgramResponse exposes only safe program metadata of a
// platform-owned generic program. The owning trainer is never exposed (generic
// programs are always platform-owned) and deletion markers are never returned.
type genericProgramResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Type             string    `json:"type"`
	Status           string    `json:"status"`
	Level            *string   `json:"level"`
	DurationWeeks    *int      `json:"duration_weeks"`
	FrequencyPerWeek *int      `json:"frequency_per_week"`
	PriceMinorUnits  int64     `json:"price_minor_units"`
	Currency         string    `json:"currency"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func newGenericProgramResponse(program generic_programs.Program) genericProgramResponse {
	return genericProgramResponse{
		ID:               program.ID,
		Name:             program.Name,
		Description:      program.Description,
		Type:             program.Type,
		Status:           program.Status,
		Level:            program.Level,
		DurationWeeks:    program.DurationWeeks,
		FrequencyPerWeek: program.FrequencyPerWeek,
		PriceMinorUnits:  program.PriceMinorUnits,
		Currency:         program.Currency,
		CreatedAt:        program.CreatedAt,
		UpdatedAt:        program.UpdatedAt,
	}
}

// genericSetResponse exposes one prescription set of a workout exercise.
type genericSetResponse struct {
	SetNumber   int      `json:"set_number"`
	SetType     string   `json:"set_type"`
	Reps        *int     `json:"reps"`
	WeightKg    *float64 `json:"weight_kg"`
	RIR         *int     `json:"rir"`
	RPE         *float64 `json:"rpe"`
	RestSeconds *int     `json:"rest_seconds"`
	Tempo       string   `json:"tempo"`
}

func newGenericSetResponse(set generic_programs.Set) genericSetResponse {
	return genericSetResponse{
		SetNumber:   set.SetNumber,
		SetType:     set.SetType,
		Reps:        set.Reps,
		WeightKg:    set.WeightKg,
		RIR:         set.RIR,
		RPE:         set.RPE,
		RestSeconds: set.RestSeconds,
		Tempo:       set.Tempo,
	}
}

// genericExerciseResponse exposes one exercise usage including the safe catalog
// summary of the assigned platform-owned exercise. Per-assignment notes and
// prescription sets complete the usage.
type genericExerciseResponse struct {
	ID                  string               `json:"id"`
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	CatalogInstructions string               `json:"catalog_instructions"`
	TargetMuscles       string               `json:"target_muscles"`
	Equipment           string               `json:"equipment"`
	Difficulty          string               `json:"difficulty"`
	VideoURL            string               `json:"video_url"`
	ImageURL            string               `json:"image_url"`
	Position            int                  `json:"position"`
	Instructions        string               `json:"instructions"`
	Notes               string               `json:"notes"`
	Sets                []genericSetResponse `json:"sets"`
}

func newGenericExerciseResponse(exercise generic_programs.WorkoutExercise) genericExerciseResponse {
	sets := make([]genericSetResponse, 0, len(exercise.Sets))
	for i := range exercise.Sets {
		sets = append(sets, newGenericSetResponse(exercise.Sets[i]))
	}
	return genericExerciseResponse{
		ID:                  exercise.Exercise.ID,
		Name:                exercise.Exercise.Name,
		Description:         exercise.Exercise.Description,
		CatalogInstructions: exercise.Exercise.Instructions,
		TargetMuscles:       exercise.Exercise.TargetMuscles,
		Equipment:           exercise.Exercise.Equipment,
		Difficulty:          exercise.Exercise.Difficulty,
		VideoURL:            exercise.Exercise.VideoURL,
		ImageURL:            exercise.Exercise.ImageURL,
		Position:            exercise.Position,
		Instructions:        exercise.Instructions,
		Notes:               exercise.Notes,
		Sets:                sets,
	}
}

// genericWorkoutResponse exposes one workout slot in position order.
type genericWorkoutResponse struct {
	Position  int                       `json:"position"`
	Exercises []genericExerciseResponse `json:"exercises"`
}

func newGenericWorkoutResponse(workout generic_programs.Workout) genericWorkoutResponse {
	exercises := make([]genericExerciseResponse, 0, len(workout.Exercises))
	for i := range workout.Exercises {
		exercises = append(exercises, newGenericExerciseResponse(workout.Exercises[i]))
	}
	return genericWorkoutResponse{Position: workout.Position, Exercises: exercises}
}

// genericWeekResponse exposes one week in week order.
type genericWeekResponse struct {
	WeekNumber int                      `json:"week_number"`
	Workouts   []genericWorkoutResponse `json:"workouts"`
}

func newGenericWeekResponse(week generic_programs.Week) genericWeekResponse {
	workouts := make([]genericWorkoutResponse, 0, len(week.Workouts))
	for i := range week.Workouts {
		workouts = append(workouts, newGenericWorkoutResponse(week.Workouts[i]))
	}
	return genericWeekResponse{WeekNumber: week.WeekNumber, Workouts: workouts}
}

// genericProgramDetailResponse is the full safe representation of a generic
// program including its complete structure.
type genericProgramDetailResponse struct {
	genericProgramResponse
	Weeks []genericWeekResponse `json:"weeks"`
}

func newGenericProgramDetailResponse(program *generic_programs.ProgramDetail) genericProgramDetailResponse {
	weeks := make([]genericWeekResponse, 0, len(program.Weeks))
	for i := range program.Weeks {
		weeks = append(weeks, newGenericWeekResponse(program.Weeks[i]))
	}
	return genericProgramDetailResponse{
		genericProgramResponse: newGenericProgramResponse(program.Program),
		Weeks:                  weeks,
	}
}

// genericSetRequest is one prescription set of a workout exercise. Training
// metrics are optional: nil values leave the corresponding nullable column
// NULL.
type genericSetRequest struct {
	SetType     string   `json:"set_type"`
	Reps        *int     `json:"reps"`
	WeightKg    *float64 `json:"weight_kg"`
	RIR         *int     `json:"rir"`
	RPE         *float64 `json:"rpe"`
	RestSeconds *int     `json:"rest_seconds"`
	Tempo       string   `json:"tempo"`
}

// genericExerciseRequest is one exercise usage inside a workout. Structural
// positions are derived server-side and never accepted.
type genericExerciseRequest struct {
	ExerciseID   string              `json:"exercise_id"`
	Instructions string              `json:"instructions"`
	Notes        string              `json:"notes"`
	Sets         []genericSetRequest `json:"sets"`
}

// genericWorkoutRequest is one workout slot of a week.
type genericWorkoutRequest struct {
	Exercises []genericExerciseRequest `json:"exercises"`
}

// genericWeekRequest is one week of the program.
type genericWeekRequest struct {
	Workouts []genericWorkoutRequest `json:"workouts"`
}

// genericProgramRequest is the full desired state of a generic program for
// create and update. Level is optional; DurationWeeks and FrequencyPerWeek are
// 0 when unset. Weeks are strongly ordered and their numbers/positions are
// derived server-side as 1..n.
type genericProgramRequest struct {
	Name             string               `json:"name"`
	Description      string               `json:"description"`
	Type             string               `json:"type"`
	Status           string               `json:"status"`
	Level            string               `json:"level"`
	DurationWeeks    int                  `json:"duration_weeks"`
	FrequencyPerWeek int                  `json:"frequency_per_week"`
	PriceMinorUnits  int64                `json:"price_minor_units"`
	Currency         string               `json:"currency"`
	Weeks            []genericWeekRequest `json:"weeks"`
}

func toGenericProgramInput(req *genericProgramRequest) generic_programs.ProgramInput {
	input := generic_programs.ProgramInput{
		Name:             req.Name,
		Description:      req.Description,
		Type:             req.Type,
		Status:           req.Status,
		Level:            req.Level,
		DurationWeeks:    req.DurationWeeks,
		FrequencyPerWeek: req.FrequencyPerWeek,
		PriceMinorUnits:  req.PriceMinorUnits,
		Currency:         req.Currency,
		Weeks:            make([]generic_programs.WeekInput, 0, len(req.Weeks)),
	}
	for i := range req.Weeks {
		weekInput := generic_programs.WeekInput{WeekNumber: i + 1, Workouts: make([]generic_programs.WorkoutInput, 0, len(req.Weeks[i].Workouts))}
		for k := range req.Weeks[i].Workouts {
			workoutInput := generic_programs.WorkoutInput{Position: k + 1, Exercises: make([]generic_programs.ExerciseInput, 0, len(req.Weeks[i].Workouts[k].Exercises))}
			for j := range req.Weeks[i].Workouts[k].Exercises {
				exerciseReq := &req.Weeks[i].Workouts[k].Exercises[j]
				exerciseInput := generic_programs.ExerciseInput{
					ExerciseID:   exerciseReq.ExerciseID,
					Instructions: exerciseReq.Instructions,
					Notes:        exerciseReq.Notes,
					Prescription: make([]generic_programs.SetInput, 0, len(exerciseReq.Sets)),
				}
				for l := range exerciseReq.Sets {
					setReq := &exerciseReq.Sets[l]
					exerciseInput.Prescription = append(exerciseInput.Prescription, generic_programs.SetInput{
						SetType:     setReq.SetType,
						Reps:        setReq.Reps,
						WeightKg:    setReq.WeightKg,
						RIR:         setReq.RIR,
						RPE:         setReq.RPE,
						RestSeconds: setReq.RestSeconds,
						Tempo:       setReq.Tempo,
					})
				}
				workoutInput.Exercises = append(workoutInput.Exercises, exerciseInput)
			}
			weekInput.Workouts = append(weekInput.Workouts, workoutInput)
		}
		input.Weeks = append(input.Weeks, weekInput)
	}
	return input
}

// GenericProgramsHandler exposes the platform-managed generic program CRUD and
// lifecycle operations for administrators. Authorization (which administrator
// role may operate) is enforced by the AdminAuthenticate and
// RequireAdminPermission middleware mounted on the routes; this handler never
// performs authorization itself and never accepts a trainer id, because generic
// programs are always platform-owned (trainer_id NULL).
type GenericProgramsHandler struct {
	service generic_programs.Service
}

func NewGenericProgramsHandler(svc generic_programs.Service) *GenericProgramsHandler {
	return &GenericProgramsHandler{service: svc}
}

// CreateProgram creates a new platform-owned generic program with its full
// structure in a single transaction.
func (h *GenericProgramsHandler) CreateProgram(c *gin.Context) {
	var req genericProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	program, err := h.service.CreateProgram(c.Request.Context(), toGenericProgramInput(&req))
	if err != nil {
		h.respondGenericProgramsError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Generic program created successfully.",
		"data":    newGenericProgramDetailResponse(program),
	})
}

// ListPrograms returns one page of active generic programs narrowed by the
// optional search query and type/level/duration/frequency filters.
func (h *GenericProgramsHandler) ListPrograms(c *gin.Context) {
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
	duration, err := queryInt(c, "duration", 0)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	frequency, err := queryInt(c, "frequency", 0)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	result, err := h.service.ListPrograms(c.Request.Context(), generic_programs.ProgramFilter{
		Query:            c.Query("search"),
		Type:             c.Query("type"),
		Level:            c.Query("level"),
		DurationWeeks:    duration,
		FrequencyPerWeek: frequency,
	}, page, limit)
	if err != nil {
		h.respondGenericProgramsError(c, err)
		return
	}

	programs := make([]genericProgramResponse, 0, len(result.Programs))
	for i := range result.Programs {
		programs = append(programs, newGenericProgramResponse(result.Programs[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Generic programs retrieved successfully.",
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

// GetProgram returns one active generic program with its complete structure.
func (h *GenericProgramsHandler) GetProgram(c *gin.Context) {
	program, err := h.service.GetProgram(c.Request.Context(), c.Param("programID"))
	if err != nil {
		h.respondGenericProgramsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Generic program retrieved successfully.",
		"data":    newGenericProgramDetailResponse(program),
	})
}

// UpdateProgram reconciles an existing generic program with the full desired
// structure in a single transaction.
func (h *GenericProgramsHandler) UpdateProgram(c *gin.Context) {
	var req genericProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	program, err := h.service.UpdateProgram(c.Request.Context(), c.Param("programID"), toGenericProgramInput(&req))
	if err != nil {
		h.respondGenericProgramsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Generic program updated successfully.",
		"data":    newGenericProgramDetailResponse(program),
	})
}

// PublishProgram transitions a draft generic program to published.
func (h *GenericProgramsHandler) PublishProgram(c *gin.Context) {
	program, err := h.service.PublishProgram(c.Request.Context(), c.Param("programID"))
	if err != nil {
		h.respondGenericProgramsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Generic program published successfully.",
		"data":    newGenericProgramResponse(program),
	})
}

// DeleteProgram soft-deletes one active generic program.
func (h *GenericProgramsHandler) DeleteProgram(c *gin.Context) {
	err := h.service.DeleteProgram(c.Request.Context(), c.Param("programID"))
	if err != nil {
		h.respondGenericProgramsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Generic program deleted successfully.",
		"data":    gin.H{},
	})
}

// respondGenericProgramsError maps generic programs service errors to API
// responses. Internal error details are never exposed to the client.
func (h *GenericProgramsHandler) respondGenericProgramsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, generic_programs.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, generic_programs.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	case errors.Is(err, generic_programs.ErrExerciseNotFound):
		RespondError(c, http.StatusUnprocessableEntity, "EXERCISE_NOT_FOUND", "Referenced exercise not found.", nil)
	case errors.Is(err, generic_programs.ErrDuplicateExercise):
		RespondError(c, http.StatusConflict, "DUPLICATE_EXERCISE", "The same exercise cannot appear more than once in a workout.", nil)
	case errors.Is(err, generic_programs.ErrProgramAlreadyPublished):
		RespondError(c, http.StatusConflict, "PROGRAM_ALREADY_PUBLISHED", "Generic program is already published.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
