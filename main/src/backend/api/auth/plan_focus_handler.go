package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/plan_focus"
)

type PlanFocusHandler struct {
	service *plan_focus.Service
}

func NewPlanFocusHandler(service *plan_focus.Service) *PlanFocusHandler {
	return &PlanFocusHandler{service: service}
}

// Get handles GET /me/plan-focus: the chosen plan or null.
func (h *PlanFocusHandler) Get(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	focus, err := h.service.Get(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Plan focus retrieved successfully.", "data": focus})
}

type planFocusRequest struct {
	ProgramID string `json:"program_id"`
}

// Set handles PUT /me/plan-focus with { "program_id" }: only a plan the member
// can open; anything else answers 404 PROGRAM_NOT_FOUND.
func (h *PlanFocusHandler) Set(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var request planFocusRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.ProgramID == "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"program_id"})
		return
	}
	focus, err := h.service.Set(c.Request.Context(), userID, request.ProgramID)
	if err != nil {
		if errors.Is(err, plan_focus.ErrProgramNotAccessible) {
			RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Plan focus saved successfully.", "data": focus})
}
