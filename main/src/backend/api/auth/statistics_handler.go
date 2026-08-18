package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/statistics"
)

// StatisticsHandler exposes the authenticated client's workout statistics. It
// never performs authentication or authorization itself: those are enforced by
// the Authenticate middleware mounted on the route. The user identity always
// comes exclusively from the authentication context; query parameters, body,
// headers or any client-supplied identity can never influence which statistics
// are returned.
type StatisticsHandler struct {
	service statistics.Service
}

func NewStatisticsHandler(svc statistics.Service) *StatisticsHandler {
	return &StatisticsHandler{service: svc}
}

// GetStatistics returns the computed workout statistics for the authenticated
// client. A client-supplied user_id is deliberately ignored: the statistics are
// always scoped to the identity from the authentication context.
func (h *StatisticsHandler) GetStatistics(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	stats, err := h.service.GetClientStatistics(c.Request.Context(), userID)
	if err != nil {
		h.respondStatisticsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Statistics retrieved successfully.",
		"data":    stats,
	})
}

// respondStatisticsError maps statistics service errors to API responses.
// Internal error details are never exposed to the client.
func (h *StatisticsHandler) respondStatisticsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, statistics.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
