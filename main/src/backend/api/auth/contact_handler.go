package auth

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/contact"
)

const contactFormMemoryBytes = 2 << 20

// ContactHandler accepts the public contact form. The recipient is configured
// server-side and the handler only reports success after the notifier accepts
// the message.
type ContactHandler struct {
	service contact.Service
}

func NewContactHandler(service contact.Service) *ContactHandler {
	return &ContactHandler{service: service}
}

// Submit validates the multipart form and delivers the message.
func (h *ContactHandler) Submit(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(contactFormMemoryBytes); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	if c.PostForm("website") != "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	message := contact.Message{
		Name:    c.PostForm("name"),
		Email:   c.PostForm("email"),
		Subject: c.PostForm("subject"),
		Body:    c.PostForm("message"),
	}

	file, header, err := c.Request.FormFile("attachment")
	switch {
	case err == nil:
		defer file.Close()
		data, readErr := io.ReadAll(io.LimitReader(file, contact.MaxAttachmentBytes+1))
		if readErr != nil || len(data) > contact.MaxAttachmentBytes {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
			return
		}
		message.Attachment = &contact.Attachment{
			Filename: header.Filename,
			Data:     data,
		}
	case errors.Is(err, http.ErrMissingFile):
		// The attachment is optional.
	default:
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	if err := h.service.Submit(c.Request.Context(), message); err != nil {
		switch {
		case errors.Is(err, contact.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, contact.ErrUnavailable):
			RespondError(c, http.StatusServiceUnavailable, "CONTACT_UNAVAILABLE", "The contact form is temporarily unavailable. Please email us directly.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Message sent successfully.",
		"data":    gin.H{"status": "sent"},
	})
}
