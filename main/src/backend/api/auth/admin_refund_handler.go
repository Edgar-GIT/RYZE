package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/services/refunds"
)

// AdminRefundHandler lets ADMIN_2 refund a completed purchase (v2.131.0).
type AdminRefundHandler struct {
	service *refunds.Service
}

func NewAdminRefundHandler(service *refunds.Service) *AdminRefundHandler {
	return &AdminRefundHandler{service: service}
}

type refundRequest struct {
	Reason          string `json:"reason"`
	Note            string `json:"note"`
	AlreadyRefunded bool   `json:"already_refunded"`
}

func (h *AdminRefundHandler) Refund(c *gin.Context) {
	adminID, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var body refundRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	purchase, err := h.service.Refund(c.Request.Context(), c.Param("id"), adminID, refunds.Input{
		Reason: body.Reason, Note: body.Note, AlreadyRefunded: body.AlreadyRefunded,
	})
	switch {
	case errors.Is(err, refunds.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Choose a refund reason; the note has at most 500 characters.", []string{"reason"})
	case errors.Is(err, refunds.ErrNotFound):
		RespondError(c, http.StatusNotFound, "PURCHASE_NOT_FOUND", "Purchase not found.", nil)
	case errors.Is(err, refunds.ErrNotRefundable):
		RespondError(c, http.StatusConflict, "PURCHASE_NOT_REFUNDABLE", "Only a completed purchase can be refunded.", nil)
	case errors.Is(err, refunds.ErrManualRequired):
		RespondError(c, http.StatusConflict, "REFUND_MANUAL_REQUIRED", "Refund the payment in the provider's dashboard, then record it as already refunded.", nil)
	case errors.Is(err, refunds.ErrProvider):
		slog.Error("provider refund failed", slog.String("error", err.Error()))
		RespondError(c, http.StatusBadGateway, "REFUND_PROVIDER_FAILED", "The payment provider could not refund the payment.", nil)
	case err != nil:
		slog.Error("refund failed", slog.String("error", err.Error()))
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	default:
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Purchase refunded; its access was removed.", "data": gin.H{
			"id": purchase.ID, "status": purchase.Status, "refunded_at": purchase.RefundedAt, "refund_reason": purchase.RefundReason,
		}})
	}
}
