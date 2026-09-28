package webhooks

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"

	"ryze/backend/services/payments"
	"ryze/backend/services/purchases"
)

// StripeWebhookHandler handles Stripe webhook events. It verifies the webhook
// signature, extracts the RYZE purchase identifier from trusted Stripe metadata,
// validates the payment amount, currency and method against the immutable
// purchase snapshot, and calls CompletePurchase as the only completion mechanism.
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
//  6. Verify the session is a fully paid one-time payment (payment_status,
//     mode, client_reference_id) before anything else.
//  7. Extract the RYZE purchase identifier from trusted Stripe metadata.
//  8. Verify payment amount, currency and payment method against the immutable
//     purchase snapshot.
//  9. Call CompletePurchase().
//
// Response semantics:
//   - 400: invalid signature, missing header, malformed payload, unexpected event data
//   - 200: unsupported event type (safely ignored), not a successful payment,
//     unknown purchase, already completed, not pending
//   - 500: internal errors where provider retry is desirable (completion failure,
//     amount/currency/payment method mismatch)
//
// Supported event types:
//   - checkout.session.completed: the primary payment success event
//   - checkout.session.async_payment_succeeded: async payment methods (e.g. bank
//     transfers)
//
// MB WAY is served by the same Checkout Session as card, so it is completed by
// the same events. The buyer approves the payment in the MB WAY app, and the
// session only reports paid once Stripe confirms it.
//
// All other event types — including refunds and disputes — are safely
// acknowledged with 200 and ignored. RYZE purchases are final: no refund
// workflow exists and no event type can revert a completed purchase.
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
// (or async_payment_succeeded) event. It first proves from the event itself that
// the session is a fully paid one-time payment belonging to the RYZE purchase,
// then validates it against the immutable purchase snapshot, and finally calls
// CompletePurchase.
func (h *StripeWebhookHandler) handleCheckoutSessionCompleted(c *gin.Context, event stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		log.Printf("[STRIPE-WEBHOOK] failed to unmarshal checkout session: %v", err)
		c.String(http.StatusBadRequest, "invalid event data")
		return
	}

	// A session is only a successful payment when Stripe reports it as fully
	// paid. Delayed payment methods emit checkout.session.completed while
	// still unpaid and settle later through async_payment_succeeded, so an
	// unpaid session is acknowledged and ignored instead of completing
	// anything.
	if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		log.Printf("[STRIPE-WEBHOOK] session %s is not a successful payment (payment_status=%s)", session.ID, session.PaymentStatus)
		c.String(http.StatusOK, "payment not completed")
		return
	}

	// Only RYZE one-time payments can complete a purchase. A session from any
	// other mode was never created by the RYZE purchase flow.
	if session.Mode != stripe.CheckoutSessionModePayment {
		log.Printf("[STRIPE-WEBHOOK] session %s has unsupported mode %q", session.ID, session.Mode)
		c.String(http.StatusOK, "unsupported checkout mode")
		return
	}

	purchaseID := session.Metadata["purchase_id"]
	if purchaseID == "" {
		log.Printf("[STRIPE-WEBHOOK] session %s has no purchase_id in metadata", session.ID)
		c.String(http.StatusOK, "no purchase_id in metadata")
		return
	}

	// The client reference ID is set at session creation and is the strongest
	// binding between the Stripe session and the RYZE purchase. A mismatch
	// means this session is not the one created for this purchase, so the event
	// is rejected and never completes anything.
	if session.ClientReferenceID != purchaseID {
		log.Printf("[STRIPE-WEBHOOK] session %s reference mismatch: client_reference_id=%q metadata purchase_id=%q", session.ID, session.ClientReferenceID, purchaseID)
		c.String(http.StatusBadRequest, "purchase reference mismatch")
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

	// The session must have been paid with the method bound to the purchase at
	// initiation. A session settled with a different method than the one the
	// purchase recorded can never complete it.
	if purchase.PaymentMethod == "" {
		log.Printf("[STRIPE-WEBHOOK] purchase %s has no recorded payment method", purchaseID)
		c.String(http.StatusOK, "purchase has no recorded payment method")
		return
	}
	if !sessionUsesPaymentMethod(session, payments.PaymentMethod(purchase.PaymentMethod)) {
		log.Printf("[STRIPE-WEBHOOK] payment method mismatch for purchase %s: session=%v recorded=%s", purchaseID, session.PaymentMethodTypes, purchase.PaymentMethod)
		c.String(http.StatusInternalServerError, "payment method mismatch")
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

// sessionUsesPaymentMethod reports whether the session was created for the given
// RYZE payment method. A session without payment method types proves nothing and
// is therefore never accepted as a match.
func sessionUsesPaymentMethod(session stripe.CheckoutSession, method payments.PaymentMethod) bool {
	if len(session.PaymentMethodTypes) == 0 {
		return false
	}
	for _, sessionMethod := range session.PaymentMethodTypes {
		if payments.IsStripePaymentMethodType(method, string(sessionMethod)) {
			return true
		}
	}
	return false
}
