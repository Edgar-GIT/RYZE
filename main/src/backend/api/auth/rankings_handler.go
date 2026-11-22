package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/rankings"
)

type RankingsHandler struct {
	service *rankings.Service
}

func NewRankingsHandler(service *rankings.Service) *RankingsHandler {
	return &RankingsHandler{service: service}
}

// List handles GET /me/achievements/rankings.
func (h *RankingsHandler) List(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	list, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Rankings retrieved successfully.", "data": list})
}
