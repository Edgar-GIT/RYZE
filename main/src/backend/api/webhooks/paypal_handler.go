package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	paypal "github.com/plutov/paypal/v4"

	"ryze/backend/services/purchases"
)

// PayPalSignatureVerifier verifies the authenticity of incoming PayPal webhook
// requests. The PayPal SDK client satisfies this interface.
type PayPalSignatureVerifier interface {
	VerifyWebhookSignature(ctx context.Context, httpReq *http.Request, webhookID string) (*paypal.VerifyWebhookResponse, error)
}

// paypalWebhookEvent represents the top-level structure of a PayPal webhook
// event. Only the fields needed for RYZE webhook processing are included; the
// full PayPal event payload is not trusted beyond these verified fields.
type paypalWebhookEvent struct {
	ID        string `json:"id"`
	EventType string `json:"event_type"`
	Resource  struct {
		ReferenceID string `json:"reference_id"`
		Amount      struct {
			CurrencyCode string `json:"currency_code"`
			Value        string `json:"value"`
		} `json:"amount"`
	} `json:"resource"`
}

// PayPalWebhookHandler handles PayPal webhook events. It verifies the webhook
// authenticity using the PayPal SDK's VerifyWebhookSignature, extracts the
// RYZE purchase identifier from the verified order's purchase unit reference
// ID, validates the payment amount and currency against the immutable purchase
// snapshot, and calls CompletePurchase as the only completion mechanism.
//
// Checkout initiation does not complete a purchase. Browser redirects do not
// complete a purchase. Only verified provider events can trigger CompletePurchase.
//
// PayPalWebhookHandler is safe for concurrent use by multiple goroutines.
type PayPalWebhookHandler struct {
	verifier        PayPalSignatureVerifier
	webhookID       string
	purchaseService purchases.Service
}

// NewPayPalWebhookHandler returns a handler configured with the PayPal webhook
// verifier, webhook ID and purchase service. The webhook ID is used by PayPal's
// server-side verification API to confirm the event's authenticity.
func NewPayPalWebhookHandler(verifier PayPalSignatureVerifier, webhookID string, purchaseService purchases.Service) *PayPalWebhookHandler {
	return &PayPalWebhookHandler{
		verifier:        verifier,
		webhookID:       webhookID,
		purchaseService: purchaseService,
	}
}

// Handle processes an incoming PayPal webhook. The flow is:
//
//  1. Read the raw request body (needed for both verification and parsing).
//  2. Verify the webhook authenticity using PayPal's server-side verification.
//  3. Parse the verified event.
//  4. Handle only CHECKOUT.ORDER.APPROVED events.
//  5. Extract the RYZE purchase identifier from the purchase unit reference ID.
//  6. Verify payment amount and currency against the immutable purchase snapshot.
//  7. Call CompletePurchase().
//
// Response semantics:
//   - 400: invalid verification, malformed payload
//   - 200: unsupported event type (safely ignored), unknown purchase, already completed, not pending
//   - 500: internal errors where provider retry is desirable (completion failure, amount/currency mismatch)
//
// Supported event types:
//   - CHECKOUT.ORDER.APPROVED: the buyer has approved the order
//
// All other event types are safely acknowledged with 200 to prevent infinite retries.
//
// PayPal webhook authenticity is verified using PayPal's official
// VerifyWebhookSignature API, which validates the cryptographic signature of
// the incoming request against the configured webhook ID.
func (h *PayPalWebhookHandler) Handle(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[PAYPAL-WEBHOOK] failed to read request body: %v", err)
		c.String(http.StatusBadRequest, "unable to read body")
		return
	}

	if h.webhookID == "" {
		log.Printf("[PAYPAL-WEBHOOK] no webhook ID configured")
		c.String(http.StatusBadRequest, "webhook not configured")
		return
	}

	// Re-create a request with the body bytes for verification.
	verifyReq := c.Request.Clone(c.Request.Context())
	verifyReq.Body = io.NopCloser(strings.NewReader(string(rawBody)))

	verifyResp, err := h.verifier.VerifyWebhookSignature(c.Request.Context(), verifyReq, h.webhookID)
	if err != nil {
		log.Printf("[PAYPAL-WEBHOOK] signature verification failed: %v", err)
		c.String(http.StatusBadRequest, "verification failed")
		return
	}

	if verifyResp.VerificationStatus != "SUCCESS" {
		log.Printf("[PAYPAL-WEBHOOK] invalid verification status: %s", verifyResp.VerificationStatus)
		c.String(http.StatusBadRequest, "invalid signature")
		return
	}

	var event paypalWebhookEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		log.Printf("[PAYPAL-WEBHOOK] failed to parse event payload: %v", err)
		c.String(http.StatusBadRequest, "invalid payload")
		return
	}

	switch event.EventType {
	case "CHECKOUT.ORDER.APPROVED":
		h.handleOrderApproved(c, event)
	default:
		log.Printf("[PAYPAL-WEBHOOK] ignoring event type: %s", event.EventType)
		c.String(http.StatusOK, "event type not handled")
		return
	}
}

// handleOrderApproved processes a verified CHECKOUT.ORDER.APPROVED event. It
// extracts purchase details from the verified order's purchase unit, validates
// them against the immutable purchase snapshot, and calls CompletePurchase.
func (h *PayPalWebhookHandler) handleOrderApproved(c *gin.Context, event paypalWebhookEvent) {
	purchaseID := event.Resource.ReferenceID
	if purchaseID == "" {
		log.Printf("[PAYPAL-WEBHOOK] event %s has no reference_id in resource", event.ID)
		c.String(http.StatusOK, "no reference_id in resource")
		return
	}

	purchase, err := h.purchaseService.GetPurchaseByID(c.Request.Context(), purchaseID)
	if err != nil {
		if errors.Is(err, purchases.ErrPurchaseNotFound) {
			log.Printf("[PAYPAL-WEBHOOK] purchase %s not found", purchaseID)
			c.String(http.StatusOK, "purchase not found")
			return
		}
		log.Printf("[PAYPAL-WEBHOOK] failed to load purchase %s: %v", purchaseID, err)
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	expectedAmount := minorUnitsToDecimalString(purchase.PriceMinorUnits)
	if event.Resource.Amount.Value != expectedAmount {
		log.Printf("[PAYPAL-WEBHOOK] amount mismatch for purchase %s: provider=%s snapshot=%d (%s)", purchaseID, event.Resource.Amount.Value, purchase.PriceMinorUnits, expectedAmount)
		c.String(http.StatusInternalServerError, "amount mismatch")
		return
	}

	if strings.ToUpper(event.Resource.Amount.CurrencyCode) != strings.ToUpper(purchase.Currency) {
		log.Printf("[PAYPAL-WEBHOOK] currency mismatch for purchase %s: provider=%s snapshot=%s", purchaseID, event.Resource.Amount.CurrencyCode, purchase.Currency)
		c.String(http.StatusInternalServerError, "currency mismatch")
		return
	}

	if purchase.Status == "completed" {
		log.Printf("[PAYPAL-WEBHOOK] purchase %s already completed", purchaseID)
		c.String(http.StatusOK, "already completed")
		return
	}

	if purchase.Status != "pending" {
		log.Printf("[PAYPAL-WEBHOOK] purchase %s is not pending (status=%s)", purchaseID, purchase.Status)
		c.String(http.StatusOK, "purchase not pending")
		return
	}

	result, err := h.purchaseService.CompletePurchase(c.Request.Context(), purchaseID)
	if err != nil {
		log.Printf("[PAYPAL-WEBHOOK] CompletePurchase failed for %s: %v", purchaseID, err)
		c.String(http.StatusInternalServerError, "completion failed")
		return
	}

	log.Printf("[PAYPAL-WEBHOOK] purchase %s completed successfully via PayPal event %s", result.ID, event.ID)
	c.String(http.StatusOK, "completed")
}

// minorUnitsToDecimalString converts a minor currency units amount (e.g. 4999
// cents) to a decimal string (e.g. "49.99") as used by the PayPal API.
// This duplicates the function in paypal_provider.go to avoid a cross-package
// dependency; both implementations are identical.
func minorUnitsToDecimalString(minorUnits int64) string {
	negative := minorUnits < 0
	if negative {
		minorUnits = -minorUnits
	}

	whole := minorUnits / 100
	fraction := minorUnits % 100

	if fraction == 0 {
		if negative {
			return fmt.Sprintf("-%d.00", whole)
		}
		return fmt.Sprintf("%d.00", whole)
	}

	if negative {
		return fmt.Sprintf("-%d.%02d", whole, fraction)
	}
	return fmt.Sprintf("%d.%02d", whole, fraction)
}
