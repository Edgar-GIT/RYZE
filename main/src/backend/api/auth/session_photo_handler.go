package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/session_photos"
)

// SessionPhotoHandler serves a member's photo of their own workout.
type SessionPhotoHandler struct {
	service *session_photos.Service
}

func NewSessionPhotoHandler(service *session_photos.Service) *SessionPhotoHandler {
	return &SessionPhotoHandler{service: service}
}

func (h *SessionPhotoHandler) respond(c *gin.Context, err error) {
	switch {
	case errors.Is(err, session_photos.ErrNotFound), errors.Is(err, session_photos.ErrNoPhoto):
		RespondError(c, http.StatusNotFound, "NOT_FOUND", "Not found.", nil)
	case errors.Is(err, session_photos.ErrNotAllowed):
		RespondError(c, http.StatusConflict, "WORKOUT_PHOTO_NOT_ALLOWED", "This workout cannot take a photo.", nil)
	case errors.Is(err, session_photos.ErrImageTooLarge):
		RespondError(c, http.StatusRequestEntityTooLarge, "IMAGE_TOO_LARGE", "The photo is too large.", nil)
	case errors.Is(err, session_photos.ErrInvalidImage):
		RespondError(c, http.StatusBadRequest, "INVALID_IMAGE", "Use a JPEG or PNG photo.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}

// Upload handles POST /me/workout-sessions/:sessionID/photo (multipart "file").
func (h *SessionPhotoHandler) Upload(c *gin.Context) {
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
	if err := h.service.Set(c.Request.Context(), userID, c.Param("sessionID"), file); err != nil {
		h.respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Workout photo saved.", "data": gin.H{"has_photo": true}})
}

// Remove handles DELETE /me/workout-sessions/:sessionID/photo.
func (h *SessionPhotoHandler) Remove(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	if err := h.service.Remove(c.Request.Context(), userID, c.Param("sessionID")); err != nil {
		h.respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Workout photo removed.", "data": gin.H{"has_photo": false}})
}

// Serve handles GET /me/workout-sessions/:sessionID/photo: the owner only,
// never cached by shared caches.
func (h *SessionPhotoHandler) Serve(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	path, err := h.service.Path(c.Request.Context(), userID, c.Param("sessionID"))
	if err != nil {
		h.respond(c, err)
		return
	}
	c.Header("Content-Type", "image/jpeg")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Header("Cache-Control", "private, max-age=3600")
	c.File(path)
}
