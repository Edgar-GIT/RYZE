package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/program_assignments"
)

// trainerClientProgramResponse only exposes safe assignment information: the
// assignment identity, the owning trainer id, the program and client ids, the
// safe program metadata and the lifecycle timestamps. Deletion markers and any
// internal data are never exposed.
type trainerClientProgramResponse struct {
	ID        string                 `json:"id"`
	TrainerID string                 `json:"trainer_id"`
	ProgramID string                 `json:"program_id"`
	UserID    string                 `json:"user_id"`
	Program   trainerProgramResponse `json:"program"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

func newTrainerClientProgramResponse(assignment *program_assignments.Assignment) trainerClientProgramResponse {
	return trainerClientProgramResponse{
		ID:        assignment.ID,
		TrainerID: assignment.TrainerID,
		ProgramID: assignment.ProgramID,
		UserID:    assignment.UserID,
		Program: trainerProgramResponse{
			ID:          assignment.Program.ID,
			TrainerID:   assignment.Program.TrainerID,
			Name:        assignment.Program.Name,
			Description: assignment.Program.Description,
			Type:        assignment.Program.Type,
			Status:      assignment.Program.Status,
			CreatedAt:   assignment.Program.CreatedAt,
			UpdatedAt:   assignment.Program.UpdatedAt,
		},
		CreatedAt: assignment.CreatedAt,
		UpdatedAt: assignment.UpdatedAt,
	}
}

// assignProgramRequest is the request DTO for POST
// /api/v1/trainer/clients/:userID/programs. It only carries the program id; the
// trainer identity never comes from the body and is always resolved from the
// authenticated trainer context.
type assignProgramRequest struct {
	ProgramID string `json:"program_id"`
}

// TrainerClientProgramHandler exposes the authenticated trainer's client
// program assignment operations. It never performs authentication or
// authorization itself: those are enforced by the Authenticate,
// TrainerAuthenticate and RequireTrainerPermission middleware mounted on the
// route. The trainer identity always comes exclusively from the trainer
// context; request parameters, body, headers or any client-supplied identity
// can never influence which assignment is created, listed or removed.
type TrainerClientProgramHandler struct {
	service program_assignments.Service
}

func NewTrainerClientProgramHandler(svc program_assignments.Service) *TrainerClientProgramHandler {
	return &TrainerClientProgramHandler{service: svc}
}

// AssignProgram assigns one of the authenticated trainer's own programs to one
// of the authenticated trainer's active clients. Only the program id is
// accepted from the body; the trainer identity and the client identity are
// resolved from the trainer context and the path respectively and can never be
// supplied by the client.
func (h *TrainerClientProgramHandler) AssignProgram(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	var req assignProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	assignment, err := h.service.AssignProgram(c.Request.Context(), trainerID, c.Param("userID"), req.ProgramID)
	if err != nil {
		h.respondAssignmentError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Program assigned successfully.",
		"data":    newTrainerClientProgramResponse(assignment),
	})
}

// ListClientPrograms returns every active program assignment of one of the
// authenticated trainer's active clients. A client-supplied trainer_id query
// parameter is deliberately ignored: the list is always scoped to the trainer
// identity from the context.
func (h *TrainerClientProgramHandler) ListClientPrograms(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	assignments, err := h.service.ListClientPrograms(c.Request.Context(), trainerID, c.Param("userID"))
	if err != nil {
		h.respondAssignmentError(c, err)
		return
	}

	programs := make([]trainerClientProgramResponse, 0, len(assignments))
	for i := range assignments {
		programs = append(programs, newTrainerClientProgramResponse(&assignments[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client programs retrieved successfully.",
		"data": gin.H{
			"programs": programs,
		},
	})
}

// RemoveAssignment soft-deletes ONLY the program assignment between the
// authenticated trainer and the client identified by the path parameters. The
// program and the trainer-client relationship are never deleted.
func (h *TrainerClientProgramHandler) RemoveAssignment(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	err := h.service.RemoveAssignment(c.Request.Context(), trainerID, c.Param("userID"), c.Param("assignmentID"))
	if err != nil {
		h.respondAssignmentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program assignment removed successfully.",
		"data":    gin.H{},
	})
}

// respondAssignmentError maps program assignment service errors to API
// responses. Internal error details are never exposed to the client.
func (h *TrainerClientProgramHandler) respondAssignmentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, program_assignments.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, program_assignments.ErrClientRelationNotFound):
		RespondError(c, http.StatusNotFound, "CLIENT_RELATION_NOT_FOUND", "Client relationship not found.", nil)
	case errors.Is(err, program_assignments.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	case errors.Is(err, program_assignments.ErrAssignmentNotFound):
		RespondError(c, http.StatusNotFound, "ASSIGNMENT_NOT_FOUND", "Program assignment not found.", nil)
	case errors.Is(err, program_assignments.ErrAssignmentAlreadyActive):
		RespondError(c, http.StatusConflict, "ASSIGNMENT_ALREADY_ACTIVE", "Program already assigned to this client.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
