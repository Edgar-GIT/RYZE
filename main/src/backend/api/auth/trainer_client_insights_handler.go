package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/trainer_client_insights"
)

type TrainerClientInsightsHandler struct {
	service *trainer_client_insights.Service
}

func NewTrainerClientInsightsHandler(service *trainer_client_insights.Service) *TrainerClientInsightsHandler {
	return &TrainerClientInsightsHandler{service: service}
}

func (h *TrainerClientInsightsHandler) GetOverview(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	overview, err := h.service.GetOverview(c.Request.Context(), trainerID, c.Param("userID"))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client overview retrieved successfully.",
		"data":    overview,
	})
}

func (h *TrainerClientInsightsHandler) GetStatistics(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	stats, err := h.service.GetStatistics(c.Request.Context(), trainerID, c.Param("userID"))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client statistics retrieved successfully.",
		"data":    stats,
	})
}

func (h *TrainerClientInsightsHandler) ListWorkoutSessions(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	limit, err := queryInt(c, "limit", trainer_client_insights.DefaultHistoryLimit)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	result, err := h.service.ListWorkoutSessions(c.Request.Context(), trainerID, c.Param("userID"), page, limit)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client workout sessions retrieved successfully.",
		"data": gin.H{
			"sessions": result.Sessions,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": result.TotalPages,
			},
		},
	})
}

func (h *TrainerClientInsightsHandler) GetWorkoutSession(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	session, err := h.service.GetWorkoutSession(
		c.Request.Context(),
		trainerID,
		c.Param("userID"),
		c.Param("sessionID"),
	)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client workout session retrieved successfully.",
		"data":    newWorkoutSessionResponse(session),
	})
}

func (h *TrainerClientInsightsHandler) GetExerciseHistory(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	limit, err := queryInt(c, "limit", trainer_client_insights.DefaultHistoryLimit)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	history, err := h.service.GetExerciseHistory(
		c.Request.Context(),
		trainerID,
		c.Param("userID"),
		c.Param("exerciseID"),
		limit,
	)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client exercise history retrieved successfully.",
		"data":    history,
	})
}

func (h *TrainerClientInsightsHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, trainer_client_insights.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, trainer_client_insights.ErrClientNotFound):
		RespondError(c, http.StatusNotFound, "CLIENT_NOT_FOUND", "Client not found.", nil)
	case errors.Is(err, trainer_client_insights.ErrSessionNotFound):
		RespondError(c, http.StatusNotFound, "WORKOUT_SESSION_NOT_FOUND", "Workout session not found.", nil)
	case errors.Is(err, trainer_client_insights.ErrStatisticsInternal),
		errors.Is(err, trainer_client_insights.ErrSessionInternal),
		errors.Is(err, trainer_client_insights.ErrInsightUnavailable):
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.", nil)
	}
}
