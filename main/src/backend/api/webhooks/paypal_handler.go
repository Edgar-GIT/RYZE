package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	paypal "github.com/plutov/paypal/v4"

	"ryze/backend/models"
	"ryze/backend/services/payments"
	"ryze/backend/services/purchases"
)

// PayPalSignatureVerifier verifies the authenticity of incoming PayPal webhook
// requests. The PayPal SDK client satisfies this interface.
type PayPalSignatureVerifier interface {
	VerifyWebhookSignature(ctx context.Context, httpReq *http.Request, webhookID string) (*paypal.VerifyWebhookResponse, error)
}

// PayPalOrderReader resolves a PayPal Order from the PayPal API. Confirmed
// capture settlements reference an order id, and the RYZE purchase id is only
// recovered by loading that order and reading its purchase-unit reference. The
// PayPal SDK client satisfies this interface.
type PayPalOrderReader interface {
	GetOrder(ctx context.Context, orderID string) (*paypal.Order, error)
}

// PayPalWebhookVerifier is the combined surface a PayPal webhook handler needs:
// request signature verification plus server-side order resolution. The same
// SDK client satisfies both.
type PayPalWebhookVerifier interface {
	PayPalSignatureVerifier
	PayPalOrderReader
}

// paypalWebhookEvent represents the top-level structure of a PayPal webhook
// event. Only the fields needed for RYZE webhook processing are included; the
// full PayPal event payload is not trusted beyond these verified fields.
type paypalWebhookEvent struct {
	ID        string `json:"id"`
	EventType string `json:"event_type"`
	Resource  struct {
		// ID is the PayPal Order identifier (CHECKOUT.ORDER.APPROVED) or the
		// Capture identifier (PAYMENT.CAPTURE.COMPLETED), depending on the
		// event type.
		ID     string `json:"id"`
		Status string `json:"status"`
		// ReferenceID is the RYZE purchase id carried on the order's single
		// purchase unit (CHECKOUT.ORDER.APPROVED only).
		ReferenceID string `json:"reference_id"`
		Amount      struct {
			CurrencyCode string `json:"currency_code"`
			Value        string `json:"value"`
		} `json:"amount"`
		// SupplementaryData carries the related order id on a capture object,
		// which is how a PAYMENT.CAPTURE.COMPLETED settlement is traced back to
		// the RYZE purchase.
		SupplementaryData struct {
			RelatedIDs struct {
				OrderID string `json:"order_id"`
			} `json:"related_ids"`
		} `json:"supplementary_data"`
	} `json:"resource"`
}

// PayPalWebhookHandler handles PayPal webhook events. It verifies the webhook
// authenticity using the PayPal SDK's VerifyWebhookSignature, extracts the
// RYZE purchase identifier from the verified order's purchase unit reference
// ID and the PayPal Order ID, validates the payment amount and currency
// against the immutable purchase snapshot, captures the approved order
// server-side through the purchase service, and completes the purchase.
//
// Checkout initiation does not complete a purchase. Browser redirects do not
// complete a purchase. Only verified provider events or a server-side capture
// bound to the purchase (reference id, amount and currency verification) can
// trigger purchase completion.
//
// Every verified delivery that resolves to a RYZE purchase also leaves a
// durable reconciliation event describing its verdict, so a settlement that
// cannot be resolved from the purchase row stays queryable through the
// restricted administrative reconciliation surface. Verdicts reached before
// the purchase is recovered (malformed or unresolved deliveries) have nothing
// safe to attach the record to and remain in the webhook logs only. Recording
// is always best-effort and never changes the HTTP verdict.
//
// PayPalWebhookHandler is safe for concurrent use by multiple goroutines.
type PayPalWebhookHandler struct {
	verifier        PayPalWebhookVerifier
	webhookID       string
	purchaseService purchases.Service
	outcomes        ReconciliationOutcomeRecorder
}

// PayPalWebhookHandlerOption configures a PayPalWebhookHandler.
type PayPalWebhookHandlerOption func(*PayPalWebhookHandler)

// WithPayPalOutcomeRecorder wires the durable reconciliation outcome recorder
// into the handler. It is optional for backward compatibility with existing
// constructor call sites; without it, verdicts are only logged and the HTTP
// behavior is unchanged.
func WithPayPalOutcomeRecorder(outcomes ReconciliationOutcomeRecorder) PayPalWebhookHandlerOption {
	return func(h *PayPalWebhookHandler) {
		h.outcomes = outcomes
	}
}

// NewPayPalWebhookHandler returns a handler configured with the PayPal webhook
// verifier, webhook ID and purchase service. The webhook ID is used by PayPal's
// server-side verification API to confirm the event's authenticity.
func NewPayPalWebhookHandler(verifier PayPalWebhookVerifier, webhookID string, purchaseService purchases.Service, opts ...PayPalWebhookHandlerOption) *PayPalWebhookHandler {
	h := &PayPalWebhookHandler{
		verifier:        verifier,
		webhookID:       webhookID,
		purchaseService: purchaseService,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	return h
}

// Handle processes an incoming PayPal webhook. The flow is:
//
//  1. Read the raw request body (needed for both verification and parsing).
//  2. Verify the webhook authenticity using PayPal's server-side verification.
//  3. Parse the verified event.
//  4. Handle only CHECKOUT.ORDER.APPROVED and PAYMENT.CAPTURE.COMPLETED events.
//  5. Extract the RYZE purchase identifier (purchase-unit reference for an
//     approved order, related order id for a confirmed capture).
//  6. Verify payment amount and currency against the immutable purchase snapshot.
//  7. Capture approved orders server-side and complete; reconcile confirmed
//     captures when the purchase was superseded in the meantime.
//
//   - 400: invalid verification, malformed payload
//   - 200: unsupported event type (safely ignored), unknown purchase, already
//     completed, not pending, late settlements (reconciled or diagnosed),
//     reference/amount/currency/payment-method mismatch, late approval, Test
//     Mode purchase carrying a real payment event. The mismatches are
//     acknowledged (not retried) because a duplicate delivery cannot change the
//     immutable purchase snapshot: the durable reconciliation event is the
//     operator-facing diagnosis instead.
//   - 500: internal errors where provider retry is desirable (completion or
//     reconciliation failure, transient load failure).
//
// Supported event types:
//   - CHECKOUT.ORDER.APPROVED: the buyer has approved the order (consent, not a
//     captured payment — the order is captured server-side by this handler).
//   - PAYMENT.CAPTURE.COMPLETED: an order was captured and the funds moved.
//     This is the settlement signal that reconciles a purchase superseded
//     between the capture on the browser return leg and completion.
//
// All other event types are safely acknowledged with 200 to prevent infinite
// retries.
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
	case "PAYMENT.CAPTURE.COMPLETED":
		h.handleCaptureCompleted(c, event)
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
	ctx := c.Request.Context()
	purchaseID := event.Resource.ReferenceID
	if purchaseID == "" {
		log.Printf("[PAYPAL-WEBHOOK] event %s has no reference_id in resource", event.ID)
		c.String(http.StatusOK, "no reference_id in resource")
		return
	}

	purchase, err := h.purchaseService.GetPurchaseByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, purchases.ErrPurchaseNotFound) {
			log.Printf("[PAYPAL-WEBHOOK] purchase %s not found", purchaseID)
			c.String(http.StatusOK, "purchase not found")
			return
		}
		log.Printf("[PAYPAL-WEBHOOK] failed to load purchase %s: %v", purchaseID, err)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeTransientFailure)
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	// A verified real-payment order must never be captured against a Test Mode
	// purchase. Test Mode IDs never reach a live provider, so an approval
	// carrying one is an anomaly: it is acknowledged, recorded for
	// investigation, and the order is deliberately left uncaptured.
	if purchase.Test {
		log.Printf("[PAYPAL-WEBHOOK] verified order event %s resolved to Test Mode purchase %s; not capturing", event.ID, purchaseID)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeTestPurchaseEvent)
		c.String(http.StatusOK, "test purchase")
		return
	}

	// The amount conversion is the single shared implementation used to both
	// create and verify the order, so the two can never drift apart. A mismatch
	// is permanent and never captured; it is acknowledged and recorded.
	expectedAmount := payments.MinorUnitsToDecimalString(purchase.PriceMinorUnits)
	if event.Resource.Amount.Value != expectedAmount {
		log.Printf("[PAYPAL-WEBHOOK] amount mismatch for purchase %s: provider=%s snapshot=%d (%s)", purchaseID, event.Resource.Amount.Value, purchase.PriceMinorUnits, expectedAmount)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeAmountMismatch)
		c.String(http.StatusOK, "amount mismatch")
		return
	}

	if strings.ToUpper(event.Resource.Amount.CurrencyCode) != strings.ToUpper(purchase.Currency) {
		log.Printf("[PAYPAL-WEBHOOK] currency mismatch for purchase %s: provider=%s snapshot=%s", purchaseID, event.Resource.Amount.CurrencyCode, purchase.Currency)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeCurrencyMismatch)
		c.String(http.StatusOK, "currency mismatch")
		return
	}

	// The purchase must be bound to the PayPal method. A PayPal order can only
	// ever complete a purchase recorded as PayPal, mirroring the equivalent
	// check the Stripe webhook performs on the session's payment method types.
	// A purchase with no recorded method proves nothing and is ignored.
	if purchase.PaymentMethod != string(payments.PaymentMethodPayPal) {
		log.Printf("[PAYPAL-WEBHOOK] payment method mismatch for purchase %s: recorded=%q", purchaseID, purchase.PaymentMethod)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeMethodMismatch)
		c.String(http.StatusOK, "purchase is not bound to the paypal method")
		return
	}

	if purchase.Status == "completed" {
		log.Printf("[PAYPAL-WEBHOOK] purchase %s already completed", purchaseID)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeAlreadyCompleted)
		c.String(http.StatusOK, "already completed")
		return
	}

	if purchase.Status != "pending" {
		if purchase.Status == "failed" {
			// A late APPROVAL is consent, not a captured payment. Capturing the
			// order now would move money at PayPal with no guarantee that the
			// purchase can still complete (a completed purchase or an active
			// entitlement for the pair may already exist), so the order is
			// deliberately left uncaptured — it lapses at PayPal and no funds
			// move. If the order was captured on the browser return leg, PayPal
			// delivers PAYMENT.CAPTURE.COMPLETED and this handler reconciles it;
			// this branch only ever records the diagnostic.
			log.Printf("[PAYPAL-WEBHOOK] late order approval for failed purchase %s (order %s): not captured, no funds moved; a PAYMENT.CAPTURE.COMPLETED event would be reconciled", purchaseID, event.Resource.ID)
			recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeLateApproval)
			c.String(http.StatusOK, "late approval acknowledged")
			return
		}
		log.Printf("[PAYPAL-WEBHOOK] purchase %s is not pending (status=%s)", purchaseID, purchase.Status)
		c.String(http.StatusOK, "purchase not pending")
		return
	}

	orderID := event.Resource.ID
	if orderID == "" {
		log.Printf("[PAYPAL-WEBHOOK] event %s has no order id in resource", event.ID)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeTransientFailure)
		c.String(http.StatusInternalServerError, "missing order id")
		return
	}

	// Capture the approved order server-side and complete the purchase. The
	// provider binds the order to the purchase (reference id, amount and
	// currency) before any capture happens; a failed capture never completes
	// the purchase.
	result, err := h.purchaseService.CompletePurchaseWithCapture(ctx, purchaseID, orderID)
	if err != nil {
		switch {
		case errors.Is(err, purchases.ErrPurchaseNotFound), errors.Is(err, purchases.ErrPurchaseNotCompleted), errors.Is(err, purchases.ErrPurchaseNotPending):
			// The purchase moved on concurrently; the durable state is the
			// source of truth and the order stays uncaptured.
			log.Printf("[PAYPAL-WEBHOOK] capture/complete for %s observed the purchase moving on: %v", purchaseID, err)
			recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomePurchaseMovedOn)
			c.String(http.StatusOK, "purchase moved on")
		case errors.Is(err, purchases.ErrEntitlementIntegrity):
			log.Printf("[PAYPAL-WEBHOOK] capture/complete for %s hit an entitlement integrity violation: %v", purchaseID, err)
			recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeEntitlementIntegrity)
			c.String(http.StatusInternalServerError, "entitlement integrity")
		default:
			// A genuine capture or completion failure is transient: the
			// provider retries and the provisional record is upgraded to the
			// retry's verdict.
			log.Printf("[PAYPAL-WEBHOOK] capture/complete failed for %s: %v", purchaseID, err)
			recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeTransientFailure)
			c.String(http.StatusInternalServerError, "completion failed")
		}
		return
	}

	log.Printf("[PAYPAL-WEBHOOK] purchase %s completed successfully via PayPal event %s (order %s)", result.ID, event.ID, orderID)
	recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeCompleted)
	c.String(http.StatusOK, "completed")
}

// handleCaptureCompleted processes a verified PAYMENT.CAPTURE.COMPLETED event.
// Unlike CHECKOUT.ORDER.APPROVED, which is buyer consent before any money
// moves, this event proves the order was captured and the funds settled. The
// RYZE purchase is recovered from the order's single purchase-unit reference
// (the order id comes from the capture's related_ids), the captured amount and
// currency are verified against the immutable snapshot, and the purchase is
// routed to completion — the normal pending path, the late-settlement
// reconciliation path for a superseded purchase, or the idempotent
// already-completed acknowledgement.
//
// This closes the window where the browser return leg captured the order while
// a supersede was landing on the purchase: previously the verified capture was
// discarded with a 200 acknowledgement and the buyer kept no access. Now the
// confirmed settlement reconciles the failed purchase atomically (or keeps the
// failed record as the diagnostic recovery state when the pair already
// resolved).
func (h *PayPalWebhookHandler) handleCaptureCompleted(c *gin.Context, event paypalWebhookEvent) {
	ctx := c.Request.Context()

	// The event type itself is the settlement signal; a resource without a
	// COMPLETED status is malformed and must never complete anything, but it
	// also cannot be fixed by retrying, so it is acknowledged and ignored. The
	// purchase cannot be recovered from a malformed capture, so there is no
	// durable record to attach; it stays in the webhook logs.
	if event.Resource.Status != "COMPLETED" {
		log.Printf("[PAYPAL-WEBHOOK] capture event %s has non-completed status %q", event.ID, event.Resource.Status)
		c.String(http.StatusOK, "capture is not completed")
		return
	}

	orderID := event.Resource.SupplementaryData.RelatedIDs.OrderID
	if orderID == "" {
		log.Printf("[PAYPAL-WEBHOOK] capture event %s has no related order id", event.ID)
		c.String(http.StatusOK, "no related order id")
		return
	}

	// The order is loaded from the PayPal API, never trusted from the event
	// payload, and its single purchase-unit reference identifies the RYZE
	// purchase this capture settled.
	order, err := h.verifier.GetOrder(ctx, orderID)
	if err != nil {
		log.Printf("[PAYPAL-WEBHOOK] failed to load order %s for capture event %s: %v", orderID, event.ID, err)
		c.String(http.StatusInternalServerError, "failed to load order")
		return
	}
	if order == nil {
		log.Printf("[PAYPAL-WEBHOOK] order %s not found for capture event %s", orderID, event.ID)
		c.String(http.StatusInternalServerError, "order not found")
		return
	}

	purchaseID := payPalOrderPurchaseID(order)
	if purchaseID == "" {
		log.Printf("[PAYPAL-WEBHOOK] order %s does not reference exactly one RYZE purchase", orderID)
		c.String(http.StatusOK, "order has no purchase reference")
		return
	}

	purchase, err := h.purchaseService.GetPurchaseByID(ctx, purchaseID)
	if err != nil {
		if errors.Is(err, purchases.ErrPurchaseNotFound) {
			log.Printf("[PAYPAL-WEBHOOK] purchase %s not found for capture event %s", purchaseID, event.ID)
			c.String(http.StatusOK, "purchase not found")
			return
		}
		log.Printf("[PAYPAL-WEBHOOK] failed to load purchase %s: %v", purchaseID, err)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeTransientFailure)
		c.String(http.StatusInternalServerError, "internal error")
		return
	}

	// A verified settlement event must never touch a Test Mode purchase.
	// Test Mode IDs never reach a live provider, so a capture carrying one is
	// an anomaly: it is acknowledged (never completing anything), recorded for
	// investigation, and never retried.
	if purchase.Test {
		log.Printf("[PAYPAL-WEBHOOK] verified capture event %s resolved to Test Mode purchase %s; not completing", event.ID, purchaseID)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeTestPurchaseEvent)
		c.String(http.StatusOK, "test purchase")
		return
	}

	expectedAmount := payments.MinorUnitsToDecimalString(purchase.PriceMinorUnits)
	if event.Resource.Amount.Value != expectedAmount {
		log.Printf("[PAYPAL-WEBHOOK] captured amount mismatch for purchase %s: provider=%s snapshot=%d (%s)", purchaseID, event.Resource.Amount.Value, purchase.PriceMinorUnits, expectedAmount)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeAmountMismatch)
		c.String(http.StatusOK, "amount mismatch")
		return
	}

	if strings.ToUpper(event.Resource.Amount.CurrencyCode) != strings.ToUpper(purchase.Currency) {
		log.Printf("[PAYPAL-WEBHOOK] captured currency mismatch for purchase %s: provider=%s snapshot=%s", purchaseID, event.Resource.Amount.CurrencyCode, purchase.Currency)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeCurrencyMismatch)
		c.String(http.StatusOK, "currency mismatch")
		return
	}

	// A PayPal capture can only ever settle a purchase recorded as PayPal.
	if purchase.PaymentMethod != string(payments.PaymentMethodPayPal) {
		log.Printf("[PAYPAL-WEBHOOK] payment method mismatch for purchase %s: recorded=%q", purchaseID, purchase.PaymentMethod)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeMethodMismatch)
		c.String(http.StatusOK, "purchase is not bound to the paypal method")
		return
	}

	switch purchase.Status {
	case "pending":
		// The funds are already captured at PayPal; only the local completion
		// remains, so the pure completion path applies and no re-capture
		// happens.
		result, err := h.purchaseService.CompletePurchase(ctx, purchaseID)
		if err != nil {
			switch {
			case errors.Is(err, purchases.ErrPurchaseNotFound), errors.Is(err, purchases.ErrPurchaseNotCompleted):
				log.Printf("[PAYPAL-WEBHOOK] CompletePurchase for captured purchase %s observed the purchase moving on: %v", purchaseID, err)
				recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomePurchaseMovedOn)
				c.String(http.StatusOK, "purchase moved on")
			case errors.Is(err, purchases.ErrEntitlementIntegrity):
				log.Printf("[PAYPAL-WEBHOOK] CompletePurchase for captured purchase %s hit an entitlement integrity violation: %v", purchaseID, err)
				recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeEntitlementIntegrity)
				c.String(http.StatusInternalServerError, "entitlement integrity")
			default:
				log.Printf("[PAYPAL-WEBHOOK] CompletePurchase failed for captured purchase %s: %v", purchaseID, err)
				recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeTransientFailure)
				c.String(http.StatusInternalServerError, "completion failed")
			}
			return
		}
		log.Printf("[PAYPAL-WEBHOOK] purchase %s completed via captured-payment event %s (order %s)", result.ID, event.ID, orderID)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeCompleted)
		c.String(http.StatusOK, "completed")
	case "failed":
		reconcileLateSettlement(c, "PAYPAL-WEBHOOK", "paypal", event.ID, purchaseID, h.purchaseService, h.outcomes)
	case "completed":
		log.Printf("[PAYPAL-WEBHOOK] purchase %s already completed", purchaseID)
		recordOutcome(ctx, h.outcomes, "paypal", purchaseID, event.ID, models.ReconciliationOutcomeAlreadyCompleted)
		c.String(http.StatusOK, "already completed")
	default:
		log.Printf("[PAYPAL-WEBHOOK] purchase %s is not pending (status=%s)", purchaseID, purchase.Status)
		c.String(http.StatusOK, "purchase not pending")
	}
}

// payPalOrderPurchaseID returns the single RYZE purchase reference embedded in
// a PayPal order's purchase units. RYZE orders always carry exactly one
// purchase unit whose reference id is the purchase id; anything else — no
// reference, or more than one distinct reference — proves nothing and fails
// closed with an empty result.
func payPalOrderPurchaseID(order *paypal.Order) string {
	if order == nil {
		return ""
	}
	var references []string
	seen := make(map[string]bool)
	for _, unit := range order.PurchaseUnits {
		id := strings.TrimSpace(unit.ReferenceID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		references = append(references, id)
	}
	if len(references) != 1 {
		return ""
	}
	return references[0]
}
