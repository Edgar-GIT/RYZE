package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/profile"
)

// ProfileDetailsHandler serves the owner's public "about" (biography, date of
// birth, training start year, weight, social networks).
type ProfileDetailsHandler struct {
	service *profile.DetailsService
}

func NewProfileDetailsHandler(service *profile.DetailsService) *ProfileDetailsHandler {
	return &ProfileDetailsHandler{service: service}
}

// Get handles GET /me/profile/details.
func (h *ProfileDetailsHandler) Get(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	details, err := h.service.Get(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Profile details retrieved.", "data": details})
}

// Save handles PUT /me/profile/details: the whole "about" is replaced.
func (h *ProfileDetailsHandler) Save(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var input profile.DetailsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	details, err := h.service.Save(c.Request.Context(), userID, input)
	if fields, ok := profile.IsInvalidDetails(err); ok {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", fields)
		return
	}
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Profile details saved.", "data": details})
}
