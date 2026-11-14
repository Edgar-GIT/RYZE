package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/plan_preferences"
)

// PlanPreferencesHandler saves the member's sex and plan audience preference.
type PlanPreferencesHandler struct {
	service plan_preferences.Service
}

func NewPlanPreferencesHandler(service plan_preferences.Service) *PlanPreferencesHandler {
	return &PlanPreferencesHandler{service: service}
}

type planPreferencesRequest struct {
	Sex         *string `json:"sex"`
	FilterBySex *bool   `json:"filter_by_sex"`
}

// Save handles PATCH /api/v1/me/plan-preferences.
func (h *PlanPreferencesHandler) Save(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var req planPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	saved, err := h.service.Save(c.Request.Context(), userID, plan_preferences.Input{Sex: req.Sex, FilterBySex: req.FilterBySex})
	if err != nil {
		switch {
		case errors.Is(err, plan_preferences.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, plan_preferences.ErrUserNotFound):
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Plan preferences saved.",
		"data":    gin.H{"sex": saved.Sex, "plan_audience_filter": saved.FilterBySex},
	})
}
