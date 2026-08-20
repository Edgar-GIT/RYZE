package webhooks

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

	"ryze/backend/services/purchases"
)

// StripeWebhookHandler handles Stripe webhook events. It verifies the webhook
// signature, extracts the RYZE purchase identifier from trusted Stripe metadata,
// validates the payment amount and currency against the immutable purchase
// snapshot, and calls CompletePurchase as the only completion mechanism.
//
// Checkout initiation does not complete a purchase. Browser redirects do not
// complete a purchase. Only verified provider events can trigger CompletePurchase.
//
// StripeWebhookHandler is safe for concurrent use by multiple goroutines.
type StripeWebhookHandler struct {
	webhookSecret   string
	purchaseService purchases.Service
}

// NewStripeWebhookHandler returns a handler configured with the Stripe webhook
// signing secret and the purchase service. The secret is used to verify
// incoming Stripe-Signature headers; without a valid secret no webhook can
// trigger purchase completion.
func NewStripeWebhookHandler(webhookSecret string, purchaseService purchases.Service) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		webhookSecret:   webhookSecret,
		purchaseService: purchaseService,
	}
}

// Handle processes an incoming Stripe webhook. The flow is:
//
//  1. Read the raw request body.
//  2. Read the Stripe-Signature header.
//  3. Verify the signature using the configured webhook secret.
//  4. Parse the verified event.
//  5. Handle only checkout.session.completed and async_payment_succeeded events.
//  6. Extract the RYZE purchase identifier from trusted Stripe metadata.
//  7. Verify payment amount and currency against the immutable purchase snapshot.
//  8. Call CompletePurchase().
//
// Response semantics:
//   - 400: invalid signature, missing header, malformed payload
//   - 200: unsupported event type (safely ignored), unknown purchase, already completed, not pending
//   - 500: internal errors where provider retry is desirable (completion failure, amount/currency mismatch)
//
// Supported event types:
//   - checkout.session.completed: the primary payment success event
//   - checkout.session.async_payment_succeeded: async payment methods (e.g. bank transfers)
//
// All other event types are safely acknowledged with 200 to prevent infinite retries.
func (h *StripeWebhookHandler) Handle(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[STRIPE-WEBHOOK] failed to read request body: %v", err)
		c.String(http.StatusBadRequest, "unable to read body")
		return
	}

	sigHeader := c.GetHeader("Stripe-Signature")
	if sigHeader == "" {
		c.String(http.StatusBadRequest, "missing Stripe-Signature header")
		return
	}

	event, err := webhook.ConstructEvent(payload, sigHeader, h.webhookSecret)
	if err != nil {
		log.Printf("[STRIPE-WEBHOOK] signature verification failed: %v", err)
		c.String(http.StatusBadRequest, "invalid signature")
		return
	}

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted, stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded:
		h.handleCheckoutSessionCompleted(c, event)
	default:
		log.Printf("[STRIPE-WEBHOOK] ignoring event type: %s", event.Type)
		c.String(http.StatusOK, "event type not handled")
		return
	}
}

// handleCheckoutSessionCompleted processes a verified checkout.session.completed
// (or async_payment_succeeded) event. It extracts purchase details from trusted
// Stripe metadata, validates them against the immutable purchase snapshot, and
// calls CompletePurchase.
func (h *StripeWebhookHandler) handleCheckoutSessionCompleted(c *gin.Context, event stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Printf("[STRIPE-WEBHOOK] failed to unmarshal checkout session: %v", err)
		c.String(http.StatusBadRequest, "invalid event data")
		return
	}

	purchaseID := session.Metadata["purchase_id"]
	if purchaseID == "" {
		log.Printf("[STRIPE-WEBHOOK] session %s has no purchase_id in metadata", session.ID)
		c.String(http.StatusOK, "no purchase_id in metadata")
		return
	}

	purchase, err := h.purchaseService.GetPurchaseByID(c.Request.Context(), purchaseID)
	if err != nil {
		if errors.Is(err, purchases.ErrPurchaseNotFound) {
			log.Printf("[STRIPE-WEBHOOK] purchase %s not found", purchaseID)
			c.String(http.StatusOK, "purchase not found")
			return
		}
		log.Printf("[STRIPE-WEBHOOK] failed to load purchase %s: %v", purchaseID, err)
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	if session.AmountTotal != purchase.PriceMinorUnits {
		log.Printf("[STRIPE-WEBHOOK] amount mismatch for purchase %s: provider=%d snapshot=%d", purchaseID, session.AmountTotal, purchase.PriceMinorUnits)
		c.String(http.StatusInternalServerError, "amount mismatch")
		return
	}

	if strings.ToLower(string(session.Currency)) != strings.ToLower(purchase.Currency) {
		log.Printf("[STRIPE-WEBHOOK] currency mismatch for purchase %s: provider=%s snapshot=%s", purchaseID, session.Currency, purchase.Currency)
		c.String(http.StatusInternalServerError, "currency mismatch")
		return
	}

	if purchase.Status == "completed" {
		log.Printf("[STRIPE-WEBHOOK] purchase %s already completed", purchaseID)
		c.String(http.StatusOK, "already completed")
		return
	}

	if purchase.Status != "pending" {
		log.Printf("[STRIPE-WEBHOOK] purchase %s is not pending (status=%s)", purchaseID, purchase.Status)
		c.String(http.StatusOK, "purchase not pending")
		return
	}

	result, err := h.purchaseService.CompletePurchase(c.Request.Context(), purchaseID)
	if err != nil {
		log.Printf("[STRIPE-WEBHOOK] CompletePurchase failed for %s: %v", purchaseID, err)
		c.String(http.StatusInternalServerError, "completion failed")
		return
	}

	log.Printf("[STRIPE-WEBHOOK] purchase %s completed successfully via Stripe event %s", result.ID, event.ID)
	c.String(http.StatusOK, "completed")
}
