package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/health_consent"
)

// HealthConsentHandler exposes the person's explicit consent for health data
// (GDPR art. 9(2)(a), v2.131.0): read it, give it, and withdraw it (which
// erases the body measurements and the questionnaire answers).
type HealthConsentHandler struct {
	service *health_consent.Service
}

func NewHealthConsentHandler(service *health_consent.Service) *HealthConsentHandler {
	return &HealthConsentHandler{service: service}
}

func (h *HealthConsentHandler) Get(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	at, err := h.service.ConsentedAt(c.Request.Context(), userID)
	if err != nil {
		respondHealthConsentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Health data consent.", "data": gin.H{"consented_at": at}})
}

func (h *HealthConsentHandler) Give(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	at, err := h.service.Give(c.Request.Context(), userID)
	if err != nil {
		respondHealthConsentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Health data consent given.", "data": gin.H{"consented_at": at}})
}

func (h *HealthConsentHandler) Withdraw(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	if err := h.service.Withdraw(c.Request.Context(), userID); err != nil {
		respondHealthConsentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Health data consent withdrawn; the health data was erased.", "data": gin.H{"consented_at": nil}})
}

func respondHealthConsentError(c *gin.Context, err error) {
	if errors.Is(err, health_consent.ErrRequired) {
		RespondError(c, http.StatusBadRequest, "HEALTH_CONSENT_REQUIRED", "Explicit consent for health data is required.", []string{"health_consent"})
		return
	}
	slog.Error("health consent request failed", slog.String("error", err.Error()))
	RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
}
