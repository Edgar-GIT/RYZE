package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/notifications"
)

// NotificationsHandler exposes the signed-in account's notifications.
type NotificationsHandler struct {
	notifications notifications.Service
}

func NewNotificationsHandler(service notifications.Service) *NotificationsHandler {
	return &NotificationsHandler{notifications: service}
}

// List returns the account's notifications from the last 24 hours.
func (h *NotificationsHandler) List(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	list, err := h.notifications.List(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notifications retrieved.",
		"data":    gin.H{"notifications": nonNil(list.Notifications), "unread": list.Unread},
	})
}

// MarkRead marks one notification read.
func (h *NotificationsHandler) MarkRead(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	id := c.Param("notificationID")
	if _, err := uuid.Parse(id); err != nil {
		RespondError(c, http.StatusNotFound, "NOTIFICATION_NOT_FOUND", "Notification not found.", nil)
		return
	}
	if err := h.notifications.MarkRead(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, notifications.ErrNotFound) {
			RespondError(c, http.StatusNotFound, "NOTIFICATION_NOT_FOUND", "Notification not found.", nil)
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Notification marked as read.", "data": gin.H{}})
}

// MarkAllRead marks every current notification read.
func (h *NotificationsHandler) MarkAllRead(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	if err := h.notifications.MarkAllRead(c.Request.Context(), userID); err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Notifications marked as read.", "data": gin.H{}})
}

// nonNil keeps an empty list as [] in JSON.
func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
