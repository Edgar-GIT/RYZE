package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/trainercontext"
	"ryze/backend/services/trainer_earnings"
)

type TrainerEarningsHandler struct {
	service *trainer_earnings.Service
}

func NewTrainerEarningsHandler(service *trainer_earnings.Service) *TrainerEarningsHandler {
	return &TrainerEarningsHandler{service: service}
}

// Get handles GET /trainer/earnings for the authenticated trainer only.
func (h *TrainerEarningsHandler) Get(c *gin.Context) {
	identity, err := trainercontext.IdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	report, err := h.service.Report(c.Request.Context(), identity.TrainerID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Earnings retrieved successfully.", "data": report})
}
