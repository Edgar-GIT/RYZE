package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/program_access"
)

// ProgramAccessHandler exposes the authenticated client's entitlement-backed
// program access read. It never performs authentication or authorization
// itself: those are enforced by the Authenticate middleware mounted on the
// route and by the program_access service. The user identity always comes
// exclusively from the authentication context; query parameters, body, headers
// or any client-supplied identity can never influence which program is
// returned.
type ProgramAccessHandler struct {
	service program_access.Service
}

func NewProgramAccessHandler(svc program_access.Service) *ProgramAccessHandler {
	return &ProgramAccessHandler{service: svc}
}

// GetProgramAccess returns the client-safe structure of one program the
// authenticated user is entitled to access. A missing entitlement is
// indistinguishable from an unavailable program: both return a generic not
// found so access is never revealed.
func (h *ProgramAccessHandler) GetProgramAccess(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	detail, err := h.service.GetProgramAccess(c.Request.Context(), userID, c.Param("programID"))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program access retrieved successfully.",
		"data":    newPublicProgramDetailResponse(detail),
	})
}

func (h *ProgramAccessHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, program_access.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, program_access.ErrProgramNotAccessible):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
