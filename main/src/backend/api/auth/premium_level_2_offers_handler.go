package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/premium_level_2"
)

// PremiumLevel2OffersHandler is the trainer side of Premium Level 2
// allocation by offer (owner decision, 2026-11-21): the trainer's open
// offers, accept or decline, and whether they take new requests. The trainer
// comes from the trainer context, never from the request.
type PremiumLevel2OffersHandler struct {
	service premium_level_2.OfferService
}

func NewPremiumLevel2OffersHandler(service premium_level_2.OfferService) *PremiumLevel2OffersHandler {
	return &PremiumLevel2OffersHandler{service: service}
}

// List handles GET /trainer/premium-level-2/offers. An offer shows the
// service and the dates only: the intake answers are visible after accepting.
func (h *PremiumLevel2OffersHandler) List(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	offers, accepts, err := h.service.ListOffers(c.Request.Context(), trainerID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]gin.H, 0, len(offers))
	for _, offer := range offers {
		items = append(items, gin.H{
			"id":                 offer.ID,
			"request_id":         offer.RequestID,
			"offer_name":         offer.OfferName,
			"request_created_at": offer.RequestCreatedAt.UTC().Format(time.RFC3339),
			"offered_at":         offer.OfferedAt.UTC().Format(time.RFC3339),
			"expires_at":         offer.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Offers retrieved successfully.", "data": gin.H{
		"offers":           items,
		"accepts_requests": accepts,
	}})
}

// Accept handles POST /trainer/premium-level-2/offers/:offerID/accept.
func (h *PremiumLevel2OffersHandler) Accept(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	request, err := h.service.AcceptOffer(c.Request.Context(), trainerID, userID, c.Param("offerID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Request accepted.", "data": gin.H{"request_id": request.ID}})
}

// Decline handles POST /trainer/premium-level-2/offers/:offerID/decline.
func (h *PremiumLevel2OffersHandler) Decline(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	if err := h.service.DeclineOffer(c.Request.Context(), trainerID, userID, c.Param("offerID")); err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Request declined.", "data": nil})
}

// SetAvailability handles PUT /trainer/premium-level-2/availability.
func (h *PremiumLevel2OffersHandler) SetAvailability(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}
	var body struct {
		Accepts *bool `json:"accepts_requests"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Accepts == nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", []string{"accepts_requests"})
		return
	}
	if err := h.service.SetAvailability(c.Request.Context(), trainerID, *body.Accepts); err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Availability saved.", "data": gin.H{"accepts_requests": *body.Accepts}})
}

func (h *PremiumLevel2OffersHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, premium_level_2.ErrOfferNotFound):
		RespondError(c, http.StatusNotFound, "OFFER_NOT_FOUND", "Offer not found.", nil)
	case errors.Is(err, premium_level_2.ErrOfferExpired):
		RespondError(c, http.StatusConflict, "OFFER_EXPIRED", "The offer expired.", nil)
	case errors.Is(err, premium_level_2.ErrInvalidState), errors.Is(err, premium_level_2.ErrRequestNotFound):
		RespondError(c, http.StatusConflict, "REQUEST_NOT_AVAILABLE", "The request is no longer available.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
