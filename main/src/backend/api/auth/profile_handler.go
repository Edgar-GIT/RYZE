package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/profile"
)

// ProfileHandler serves the account's public identity: display name,
// @username and profile photo. The account is always the signed-in one.
type ProfileHandler struct {
	service profile.Service
}

func NewProfileHandler(service profile.Service) *ProfileHandler {
	return &ProfileHandler{service: service}
}

type updateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
}

// Update sets the display name (empty clears it) and, when given, a new
// @username.
func (h *ProfileHandler) Update(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	user, err := h.service.UpdateProfile(c.Request.Context(), userID, req.DisplayName, req.Username)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Profile updated.", "data": newAccountResponse(user)})
}

// SetAvatar stores a new profile photo (multipart field "file").
func (h *ProfileHandler) SetAvatar(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	header, err := c.FormFile("file")
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "A photo is required.", nil)
		return
	}
	file, err := header.Open()
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "A photo is required.", nil)
		return
	}
	defer file.Close()
	user, err := h.service.SetAvatar(c.Request.Context(), userID, file)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Profile photo updated.", "data": newAccountResponse(user)})
}

// RemoveAvatar deletes the profile photo.
func (h *ProfileHandler) RemoveAvatar(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	user, err := h.service.RemoveAvatar(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Profile photo removed.", "data": newAccountResponse(user)})
}

func (h *ProfileHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, profile.ErrNotFound):
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
	case errors.Is(err, profile.ErrInvalidDisplayName):
		RespondError(c, http.StatusBadRequest, "INVALID_DISPLAY_NAME", "Display names have 1 to 40 characters.", nil)
	case errors.Is(err, profile.ErrInvalidUsername):
		RespondError(c, http.StatusBadRequest, "INVALID_USERNAME", "Usernames have 3 to 30 lowercase letters, digits, dots or underscores.", nil)
	case errors.Is(err, profile.ErrUsernameTaken):
		RespondError(c, http.StatusConflict, "USERNAME_TAKEN", "This username is already in use.", nil)
	case errors.Is(err, profile.ErrInvalidImage):
		RespondError(c, http.StatusBadRequest, "INVALID_IMAGE", "The photo must be a JPEG or PNG image.", nil)
	case errors.Is(err, profile.ErrImageTooLarge):
		RespondError(c, http.StatusRequestEntityTooLarge, "IMAGE_TOO_LARGE", "The photo is too large.", nil)
	case errors.Is(err, profile.ErrStorageUnavailable):
		RespondError(c, http.StatusServiceUnavailable, "UPLOADS_UNAVAILABLE", "Photo uploads are not available right now.", nil)
	default:
		slog.Error("profile request failed", slog.String("error", err.Error()))
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
