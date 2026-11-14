package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/generic_programs"
)

// TrainerStructureHandler reads and replaces the whole structure of one of the
// authenticated trainer's plans (the trainer builder saves weeks, training
// days, exercises and sets in one request). The trainer id always comes from
// the trainer context.
type TrainerStructureHandler struct {
	service generic_programs.TrainerStructureService
}

func NewTrainerStructureHandler(svc generic_programs.TrainerStructureService) *TrainerStructureHandler {
	return &TrainerStructureHandler{service: svc}
}

type trainerStructureRequest struct {
	Weeks []genericWeekRequest `json:"weeks"`
}

func newTrainerStructureResponse(weeks []generic_programs.Week) gin.H {
	response := make([]genericWeekResponse, 0, len(weeks))
	for i := range weeks {
		response = append(response, newGenericWeekResponse(weeks[i]))
	}
	return gin.H{"weeks": response}
}

// GetStructure returns the full structure of one own plan.
func (h *TrainerStructureHandler) GetStructure(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	weeks, err := h.service.GetStructure(c.Request.Context(), trainerID, c.Param("programID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": newTrainerStructureResponse(weeks)})
}

// ReplaceStructure saves the full desired structure of one own plan.
func (h *TrainerStructureHandler) ReplaceStructure(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	var req trainerStructureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	weeks, err := h.service.ReplaceStructure(c.Request.Context(), trainerID, c.Param("programID"), toGenericWeekInputs(req.Weeks))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Program structure saved.", "data": newTrainerStructureResponse(weeks)})
}

func (h *TrainerStructureHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, generic_programs.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	case errors.Is(err, generic_programs.ErrExerciseNotFound):
		RespondError(c, http.StatusNotFound, "EXERCISE_NOT_FOUND", "Exercise not found.", nil)
	case errors.Is(err, generic_programs.ErrDuplicateExercise):
		RespondError(c, http.StatusConflict, "DUPLICATE_EXERCISE", "An exercise is assigned more than once in the same day.", nil)
	case errors.Is(err, generic_programs.ErrTrainerProgramLocked):
		RespondError(c, http.StatusConflict, "PROGRAM_LOCKED", "This program was delivered and can no longer change.", nil)
	case errors.Is(err, generic_programs.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
