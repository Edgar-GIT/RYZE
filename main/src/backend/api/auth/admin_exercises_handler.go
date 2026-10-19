package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/repositories"
	"ryze/backend/services/exercises"
)

// adminExerciseResponse exposes the administrative representation of a
// catalogue entry: additionally to the public fields it carries provenance,
// publication status and the raw stored media references.
type adminExerciseResponse struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	Source                string    `json:"source"`
	SourceID              string    `json:"source_id"`
	BodyPart              string    `json:"body_part"`
	Description           string    `json:"description"`
	Instructions          string    `json:"instructions"`
	InstructionSteps      []string  `json:"instruction_steps"`
	TargetMuscles         string    `json:"target_muscles"`
	PrimaryMuscleGroup    string    `json:"primary_muscle_group"`
	SecondaryMuscleGroups string    `json:"secondary_muscle_groups"`
	Equipment             string    `json:"equipment"`
	Difficulty            string    `json:"difficulty"`
	MovementCategory      string    `json:"movement_category"`
	VideoURL              string    `json:"video_url"`
	ImageURL              string    `json:"image_url"`
	AnimationURL          string    `json:"animation_url"`
	SourceMediaID         string    `json:"source_media_id"`
	MediaAttribution      string    `json:"media_attribution"`
	Status                string    `json:"status"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func newAdminExerciseResponse(exercise *exercises.AdminExercise) adminExerciseResponse {
	return adminExerciseResponse{
		ID:                    exercise.ID,
		Name:                  exercise.Name,
		Source:                exercise.Source,
		SourceID:              exercise.SourceID,
		BodyPart:              exercise.BodyPart,
		Description:           exercise.Description,
		Instructions:          exercise.Instructions,
		InstructionSteps:      exercise.InstructionSteps,
		TargetMuscles:         exercise.TargetMuscles,
		PrimaryMuscleGroup:    exercise.PrimaryMuscleGroup,
		SecondaryMuscleGroups: exercise.SecondaryMuscleGroups,
		Equipment:             exercise.Equipment,
		Difficulty:            exercise.Difficulty,
		MovementCategory:      exercise.MovementCategory,
		VideoURL:              exercise.VideoURL,
		ImageURL:              exercise.ImageURL,
		AnimationURL:          exercise.AnimationURL,
		SourceMediaID:         exercise.SourceMediaID,
		MediaAttribution:      exercise.MediaAttribution,
		Status:                exercise.Status,
		CreatedAt:             exercise.CreatedAt,
		UpdatedAt:             exercise.UpdatedAt,
	}
}

// adminExerciseRequest is the JSON body accepted when creating a catalogue
// entry. Every field except the name is optional.
type adminExerciseRequest struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Instructions          string   `json:"instructions"`
	InstructionSteps      []string `json:"instruction_steps"`
	TargetMuscles         string   `json:"target_muscles"`
	PrimaryMuscleGroup    string   `json:"primary_muscle_group"`
	SecondaryMuscleGroups string   `json:"secondary_muscle_groups"`
	Equipment             string   `json:"equipment"`
	Difficulty            string   `json:"difficulty"`
	MovementCategory      string   `json:"movement_category"`
	BodyPart              string   `json:"body_part"`
	ImageURL              string   `json:"image_url"`
	AnimationURL          string   `json:"animation_url"`
	SourceMediaID         string   `json:"source_media_id"`
	MediaAttribution      string   `json:"media_attribution"`
}

// adminExerciseUpdateRequest is the JSON body accepted when updating a
// catalogue entry. Absent fields stay untouched; a present field is written
// even when it is empty, so media references and classifications can be
// cleared. The name is immutable through this surface: catalogue identity is
// never renamed once program references may point at it.
type adminExerciseUpdateRequest struct {
	Description           *string   `json:"description"`
	Instructions          *string   `json:"instructions"`
	InstructionSteps      *[]string `json:"instruction_steps"`
	TargetMuscles         *string   `json:"target_muscles"`
	PrimaryMuscleGroup    *string   `json:"primary_muscle_group"`
	SecondaryMuscleGroups *string   `json:"secondary_muscle_groups"`
	Equipment             *string   `json:"equipment"`
	Difficulty            *string   `json:"difficulty"`
	MovementCategory      *string   `json:"movement_category"`
	BodyPart              *string   `json:"body_part"`
	ImageURL              *string   `json:"image_url"`
	AnimationURL          *string   `json:"animation_url"`
	SourceMediaID         *string   `json:"source_media_id"`
	MediaAttribution      *string   `json:"media_attribution"`
	Status                *string   `json:"status"`
}

// AdminExercisesHandler exposes the catalogue administration surface. Every
// route is restricted to administrators holding the exercises permissions by
// the route middleware; this handler never performs authorization itself.
type AdminExercisesHandler struct {
	service exercises.AdminService
}

func NewAdminExercisesHandler(svc exercises.AdminService) *AdminExercisesHandler {
	return &AdminExercisesHandler{service: svc}
}

// ListExercises returns one page of catalogue entries including draft and
// archived ones, optionally narrowed by status, body part, source or free
// text.
func (h *AdminExercisesHandler) ListExercises(c *gin.Context) {
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

	filter := repositories.AdminExerciseFilter{
		Query:    c.Query("q"),
		Status:   c.Query("status"),
		BodyPart: c.Query("body_part"),
		Source:   c.Query("source"),
	}

	result, err := h.service.AdminListExercises(c.Request.Context(), filter, page, limit)
	if err != nil {
		h.respondAdminExercisesError(c, err)
		return
	}

	exerciseList := make([]adminExerciseResponse, 0, len(result.Exercises))
	for i := range result.Exercises {
		exerciseList = append(exerciseList, newAdminExerciseResponse(&result.Exercises[i]))
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

// GetExercise returns one catalogue entry regardless of publication status.
func (h *AdminExercisesHandler) GetExercise(c *gin.Context) {
	exercise, err := h.service.AdminGetExercise(c.Request.Context(), c.Param("exerciseID"))
	if err != nil {
		h.respondAdminExercisesError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Exercise retrieved successfully.",
		"data":    gin.H{"exercise": newAdminExerciseResponse(exercise)},
	})
}

// CreateExercise creates one catalogue entry as a draft. Publication is a
// separate, explicit status transition.
func (h *AdminExercisesHandler) CreateExercise(c *gin.Context) {
	var req adminExerciseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	exercise, err := h.service.AdminCreateExercise(c.Request.Context(), exercises.AdminCreateExerciseInput{
		Name:                  req.Name,
		Description:           req.Description,
		Instructions:          req.Instructions,
		InstructionSteps:      req.InstructionSteps,
		TargetMuscles:         req.TargetMuscles,
		PrimaryMuscleGroup:    req.PrimaryMuscleGroup,
		SecondaryMuscleGroups: req.SecondaryMuscleGroups,
		Equipment:             req.Equipment,
		Difficulty:            req.Difficulty,
		MovementCategory:      req.MovementCategory,
		BodyPart:              req.BodyPart,
		ImageURL:              req.ImageURL,
		AnimationURL:          req.AnimationURL,
		SourceMediaID:         req.SourceMediaID,
		MediaAttribution:      req.MediaAttribution,
	})
	if err != nil {
		h.respondAdminExercisesError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Exercise created successfully.",
		"data":    gin.H{"exercise": newAdminExerciseResponse(exercise)},
	})
}

// UpdateExercise applies the provided fields to one catalogue entry. The
// publication status transition is expressed through the same payload.
func (h *AdminExercisesHandler) UpdateExercise(c *gin.Context) {
	var req adminExerciseUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	exercise, err := h.service.AdminUpdateExercise(c.Request.Context(), c.Param("exerciseID"), exercises.AdminUpdateExerciseInput{
		Description:           req.Description,
		Instructions:          req.Instructions,
		InstructionSteps:      req.InstructionSteps,
		TargetMuscles:         req.TargetMuscles,
		PrimaryMuscleGroup:    req.PrimaryMuscleGroup,
		SecondaryMuscleGroups: req.SecondaryMuscleGroups,
		Equipment:             req.Equipment,
		Difficulty:            req.Difficulty,
		MovementCategory:      req.MovementCategory,
		BodyPart:              req.BodyPart,
		ImageURL:              req.ImageURL,
		AnimationURL:          req.AnimationURL,
		SourceMediaID:         req.SourceMediaID,
		MediaAttribution:      req.MediaAttribution,
		Status:                req.Status,
	})
	if err != nil {
		h.respondAdminExercisesError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Exercise updated successfully.",
		"data":    gin.H{"exercise": newAdminExerciseResponse(exercise)},
	})
}

// respondAdminExercisesError maps administration catalogue service errors to
// API responses. Internal error details are never exposed to the client.
func (h *AdminExercisesHandler) respondAdminExercisesError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, exercises.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, exercises.ErrExerciseNotFound):
		RespondError(c, http.StatusNotFound, "EXERCISE_NOT_FOUND", "Exercise not found.", nil)
	case errors.Is(err, exercises.ErrExerciseConflict):
		RespondError(c, http.StatusConflict, "EXERCISE_NAME_TAKEN", "An exercise with this name already exists.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
