package auth

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/training_reminders"
)

// TrainingRemindersHandler reads and changes the member's training reminder
// preference (v2.131.0).
type TrainingRemindersHandler struct {
	service *training_reminders.Service
}

func NewTrainingRemindersHandler(service *training_reminders.Service) *TrainingRemindersHandler {
	return &TrainingRemindersHandler{service: service}
}

func (h *TrainingRemindersHandler) Get(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	enabled, err := h.service.Enabled(c.Request.Context(), userID)
	if err != nil {
		slog.Error("reminder preference read failed", slog.String("error", err.Error()))
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Training reminders.", "data": gin.H{"enabled": enabled}})
}

func (h *TrainingRemindersHandler) Set(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Enabled == nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"enabled"})
		return
	}
	if err := h.service.SetEnabled(c.Request.Context(), userID, *body.Enabled); err != nil {
		slog.Error("reminder preference save failed", slog.String("error", err.Error()))
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Training reminders saved.", "data": gin.H{"enabled": *body.Enabled}})
}
