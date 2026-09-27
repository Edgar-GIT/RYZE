package webhooks_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/webhook"

	"ryze/backend/api/webhooks"
	"ryze/backend/services/purchases"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- stubs ---

type stubPurchaseService struct {
	purchase *purchases.Purchase
	err      error
}

func (s *stubPurchaseService) CreatePurchaseIntent(_ context.Context, _, _ string) (*purchases.Purchase, error) {
	return nil, errors.New("not implemented")
}

func (s *stubPurchaseService) InitiatePayment(_ context.Context, _, _, _ string) (*purchases.PaymentResult, error) {
	return nil, errors.New("not implemented")
}

func (s *stubPurchaseService) CompletePurchase(_ context.Context, _ string) (*purchases.Purchase, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.purchase, nil
}

func (s *stubPurchaseService) GetPurchaseByID(_ context.Context, _ string) (*purchases.Purchase, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.purchase, nil
}

func (s *stubPurchaseService) ListPurchases(_ context.Context, _ string) ([]purchases.Purchase, error) {
	return nil, nil
}

func (s *stubPurchaseService) CapturePayment(_ context.Context, _, _, _ string) (*purchases.Purchase, error) {
	return s.purchase, s.err
}

func (s *stubPurchaseService) CompletePurchaseWithCapture(_ context.Context, _, _ string) (*purchases.Purchase, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.purchase, nil
}

func (s *stubPurchaseService) CompleteTestPurchase(_ context.Context, _, _ string) (*purchases.Purchase, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.purchase, nil
}

// --- helpers ---

const testWebhookSecret = "whsec_test_secret_key_1234567890"

func signPayload(t *testing.T, payload []byte) string {
	t.Helper()
	sp := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   payload,
		Secret:    testWebhookSecret,
		Timestamp: time.Now(),
	})
	return sp.Header
}

func buildCheckoutSessionEvent(t *testing.T, session stripe.CheckoutSession) []byte {
	t.Helper()
	event := stripe.Event{
		ID:         "evt_test_123",
		Object:     "event",
		Type:       stripe.EventTypeCheckoutSessionCompleted,
		APIVersion: stripe.APIVersion,
		Data:       &stripe.EventData{},
	}
	raw, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("failed to marshal session: %v", err)
	}
	event.Data.Raw = raw
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}
	return payload
}

// newPaidCheckoutSession builds a Checkout Session fixture representing a fully
// paid one-time payment of the given RYZE purchase, mirroring a real Stripe
// webhook payload: paid status, payment mode, total, currency and the client
// reference ID that binds the session to the purchase.
func newPaidCheckoutSession(sessionID, purchaseID string, amount int64, currency string) stripe.CheckoutSession {
	return stripe.CheckoutSession{
		ID:                sessionID,
		Object:            "checkout.session",
		Mode:              stripe.CheckoutSessionModePayment,
		PaymentStatus:     stripe.CheckoutSessionPaymentStatusPaid,
		AmountTotal:       amount,
		Currency:          stripe.Currency(currency),
		ClientReferenceID: purchaseID,
		Metadata:          map[string]string{"purchase_id": purchaseID},
	}
}

// buildEventOfType builds a signed-ready event payload of the given type around
// an arbitrary JSON data object.
func buildEventOfType(eventID string, eventType stripe.EventType, data any) []byte {
	raw, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	event := stripe.Event{
		ID:         eventID,
		Object:     "event",
		Type:       eventType,
		APIVersion: stripe.APIVersion,
		Data:       &stripe.EventData{Raw: raw},
	}
	payload, err := json.Marshal(event)
	if err != nil {
		panic(err)
	}
	return payload
}

func newTestRouter(handler *webhooks.StripeWebhookHandler) *gin.Engine {
	router := gin.New()
	router.POST("/api/v1/webhooks/stripe", handler.Handle)
	return router
}

// --- tests ---

func TestStripeWebhook_ValidCompletedEvent(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-001",
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          "pending",
	}
	svc := &stubPurchaseService{purchase: purchase}

	session := newPaidCheckoutSession("cs_test_abc123", "purchase-001", 4999, "eur")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_InvalidSignature(t *testing.T) {
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending"},
	}

	session := newPaidCheckoutSession("cs_test", "p1", 100, "eur")
	payload := buildCheckoutSessionEvent(t, session)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", "t=1234567890,v1=invalid_signature_value")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_MissingSignatureHeader(t *testing.T) {
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending"},
	}
	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader([]byte(`{}`)))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_MalformedPayload(t *testing.T) {
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending"},
	}
	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Stripe-Signature", "t=1234567890,v1=fakesig")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_UnsupportedEventType(t *testing.T) {
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending"},
	}

	event := stripe.Event{
		ID:         "evt_test_unsupported",
		Object:     "event",
		Type:       "invoice.created",
		APIVersion: stripe.APIVersion,
		Data: &stripe.EventData{
			Raw: json.RawMessage(`{}`),
		},
	}
	payload, _ := json.Marshal(event)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unsupported event, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_NoPurchaseIDInMetadata(t *testing.T) {
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending"},
	}

	session := stripe.CheckoutSession{
		ID:            "cs_test_nopurchase",
		Object:        "checkout.session",
		Mode:          stripe.CheckoutSessionModePayment,
		PaymentStatus: stripe.CheckoutSessionPaymentStatusPaid,
		AmountTotal:   100,
		Currency:      "eur",
		Metadata:      map[string]string{},
	}
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for missing purchase_id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_PurchaseNotFound(t *testing.T) {
	svc := &stubPurchaseService{
		err: purchases.ErrPurchaseNotFound,
	}

	session := newPaidCheckoutSession("cs_test_notfound", "nonexistent", 100, "eur")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unknown purchase, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_AmountMismatch(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-amt",
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          "pending",
	}
	svc := &stubPurchaseService{purchase: purchase}

	session := newPaidCheckoutSession("cs_test_amt", "purchase-amt", 9999, "eur")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for amount mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_CurrencyMismatch(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-cur",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          "pending",
	}
	svc := &stubPurchaseService{purchase: purchase}

	session := newPaidCheckoutSession("cs_test_cur", "purchase-cur", 1000, "usd")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for currency mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_AlreadyCompleted(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-done",
		PriceMinorUnits: 5000,
		Currency:        "EUR",
		Status:          "completed",
	}
	svc := &stubPurchaseService{purchase: purchase}

	session := newPaidCheckoutSession("cs_test_done", "purchase-done", 5000, "eur")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for already completed, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_NotPending(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-failed",
		PriceMinorUnits: 5000,
		Currency:        "EUR",
		Status:          "failed",
	}
	svc := &stubPurchaseService{purchase: purchase}

	session := newPaidCheckoutSession("cs_test_failed", "purchase-failed", 5000, "eur")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for not pending, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_CompletePurchaseFailure(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-fail",
		PriceMinorUnits: 2000,
		Currency:        "EUR",
		Status:          "pending",
	}
	svc := &stubPurchaseService{
		purchase: purchase,
		err:      errors.New("database connection lost"),
	}

	session := newPaidCheckoutSession("cs_test_compfail", "purchase-fail", 2000, "eur")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for completion failure, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_DuplicateDeliveryIdempotent(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-dup",
		PriceMinorUnits: 3000,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{
		purchase:      purchase,
		completeCount: &completeCount,
	}

	session := newPaidCheckoutSession("cs_test_dup", "purchase-dup", 3000, "eur")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
		req.Header.Set("Stripe-Signature", sigHeader)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("attempt %d: expected 200, got %d: %s", i+1, w.Code, w.Body.String())
		}
	}

	if completeCount != 3 {
		t.Errorf("expected CompletePurchase called 3 times, got %d", completeCount)
	}
}

func TestStripeWebhook_AsyncPaymentSucceeded(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-async",
		PriceMinorUnits: 1500,
		Currency:        "EUR",
		Status:          "pending",
	}
	svc := &stubPurchaseService{purchase: purchase}

	event := stripe.Event{
		ID:         "evt_test_async",
		Object:     "event",
		Type:       stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded,
		APIVersion: stripe.APIVersion,
		Data:       &stripe.EventData{},
	}
	session := newPaidCheckoutSession("cs_test_async", "purchase-async", 1500, "eur")
	raw, _ := json.Marshal(session)
	event.Data.Raw = raw
	payload, _ := json.Marshal(event)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for async payment succeeded, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_CaseInsensitiveCurrencyMatch(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-case",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          "pending",
	}
	svc := &stubPurchaseService{purchase: purchase}

	session := newPaidCheckoutSession("cs_test_case", "purchase-case", 1000, "EUR")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for case-insensitive currency match, got %d: %s", w.Code, w.Body.String())
	}
}

// completionCountingService tracks how many times CompletePurchase is called.
type completionCountingService struct {
	purchase      *purchases.Purchase
	err           error
	completeCount *int
}

func (s *completionCountingService) CreatePurchaseIntent(_ context.Context, _, _ string) (*purchases.Purchase, error) {
	return nil, errors.New("not implemented")
}
func (s *completionCountingService) InitiatePayment(_ context.Context, _, _, _ string) (*purchases.PaymentResult, error) {
	return nil, errors.New("not implemented")
}
func (s *completionCountingService) CompletePurchase(_ context.Context, _ string) (*purchases.Purchase, error) {
	*s.completeCount++
	if s.err != nil {
		return nil, s.err
	}
	return s.purchase, nil
}
func (s *completionCountingService) GetPurchaseByID(_ context.Context, _ string) (*purchases.Purchase, error) {
	return s.purchase, nil
}
func (s *completionCountingService) ListPurchases(_ context.Context, _ string) ([]purchases.Purchase, error) {
	return nil, nil
}
func (s *completionCountingService) CapturePayment(_ context.Context, _, _, _ string) (*purchases.Purchase, error) {
	return s.purchase, nil
}
func (s *completionCountingService) CompletePurchaseWithCapture(_ context.Context, _, _ string) (*purchases.Purchase, error) {
	*s.completeCount++
	if s.err != nil {
		return nil, s.err
	}
	return s.purchase, nil
}

func (s *completionCountingService) CompleteTestPurchase(_ context.Context, _, _ string) (*purchases.Purchase, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.purchase, nil
}

func TestStripeWebhook_UnpaidSessionIsIgnored(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-unpaid",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	// A delayed payment method completes the Checkout Session before the money
	// is settled, so payment_status is still unpaid.
	session := newPaidCheckoutSession("cs_test_unpaid", "purchase-unpaid", 4500, "eur")
	session.PaymentStatus = stripe.CheckoutSessionPaymentStatusUnpaid

	payload := buildCheckoutSessionEvent(t, session)
	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signPayload(t, payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unpaid session, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Errorf("unpaid session must never complete a purchase, CompletePurchase called %d times", completeCount)
	}
}

func TestStripeWebhook_NoPaymentStatusIsIgnored(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-nopaystatus",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	// An absent payment_status is treated as not paid: verification fails closed.
	session := newPaidCheckoutSession("cs_test_nopaystatus", "purchase-nopaystatus", 4500, "eur")
	session.PaymentStatus = ""

	payload := buildCheckoutSessionEvent(t, session)
	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signPayload(t, payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for missing payment_status, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Errorf("missing payment_status must never complete a purchase, CompletePurchase called %d times", completeCount)
	}
}

func TestStripeWebhook_SubscriptionModeIsIgnored(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-sub",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	// RYZE offers no subscriptions, so a session in subscription mode can
	// never complete a purchase.
	session := newPaidCheckoutSession("cs_test_sub", "purchase-sub", 4500, "eur")
	session.Mode = stripe.CheckoutSessionModeSubscription

	payload := buildCheckoutSessionEvent(t, session)
	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signPayload(t, payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for subscription mode, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Errorf("subscription mode must never complete a purchase, CompletePurchase called %d times", completeCount)
	}
}

func TestStripeWebhook_ClientReferenceMismatchRejected(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-ref",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	// Metadata points at one purchase while the client reference ID — the
	// binding set at session creation — points at another. The session is not
	// the one created for this purchase, so the event is rejected.
	session := newPaidCheckoutSession("cs_test_ref", "purchase-other", 4500, "eur")
	session.Metadata = map[string]string{"purchase_id": "purchase-ref"}

	payload := buildCheckoutSessionEvent(t, session)
	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signPayload(t, payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for purchase reference mismatch, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Errorf("reference mismatch must never complete a purchase, CompletePurchase called %d times", completeCount)
	}
}

func TestStripeWebhook_MissingClientReferenceIDRejected(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-noref",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	session := newPaidCheckoutSession("cs_test_noref", "purchase-noref", 4500, "eur")
	session.ClientReferenceID = ""

	payload := buildCheckoutSessionEvent(t, session)
	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signPayload(t, payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing client reference ID, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Errorf("missing client reference ID must never complete a purchase, CompletePurchase called %d times", completeCount)
	}
}

func TestStripeWebhook_RefundEventIsIgnored(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-refund",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	// RYZE purchases are final: there is no refund workflow, so refund and
	// dispute events are acknowledged and ignored.
	for _, eventType := range []stripe.EventType{
		"charge.refunded",
		"refund.created",
		"charge.dispute.created",
	} {
		completeCount = 0
		payload := buildEventOfType("evt_test_refund", eventType, newPaidCheckoutSession("cs_test_refund", "purchase-refund", 4500, "eur"))
		handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
		router := newTestRouter(handler)

		w := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
		req.Header.Set("Stripe-Signature", signPayload(t, payload))
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", eventType, w.Code, w.Body.String())
		}
		if completeCount != 0 {
			t.Errorf("%s: must never complete a purchase, CompletePurchase called %d times", eventType, completeCount)
		}
	}
}

func TestStripeWebhook_ExpiredSessionIsIgnored(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-expired",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	session := newPaidCheckoutSession("cs_test_expired", "purchase-expired", 4500, "eur")
	session.PaymentStatus = stripe.CheckoutSessionPaymentStatusUnpaid
	session.Status = stripe.CheckoutSessionStatusExpired

	payload := buildEventOfType("evt_test_expired", stripe.EventTypeCheckoutSessionExpired, session)
	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signPayload(t, payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for expired session, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Errorf("expired session must never complete a purchase, CompletePurchase called %d times", completeCount)
	}
}

func TestStripeWebhook_EmptySigningSecretCompletesNothing(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-nosecret",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	// Stripe is never enabled without a signing secret, so a handler built with
	// an empty secret must reject every delivery.
	session := newPaidCheckoutSession("cs_test_nosecret", "purchase-nosecret", 4500, "eur")
	payload := buildCheckoutSessionEvent(t, session)

	handler := webhooks.NewStripeWebhookHandler("", svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signPayload(t, payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without a signing secret, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Errorf("unverifiable delivery must never complete a purchase, CompletePurchase called %d times", completeCount)
	}
}

func TestStripeWebhook_SignatureSignedWithAnotherSecretRejected(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-othersecret",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	session := newPaidCheckoutSession("cs_test_othersecret", "purchase-othersecret", 4500, "eur")
	payload := buildCheckoutSessionEvent(t, session)

	forged := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   payload,
		Secret:    "whsec_a_completely_different_secret",
		Timestamp: time.Now(),
	})

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", forged.Header)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a signature from another secret, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Errorf("forged signature must never complete a purchase, CompletePurchase called %d times", completeCount)
	}
}

func TestStripeWebhook_TamperedPayloadRejected(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-tampered",
		PriceMinorUnits: 4500,
		Currency:        "EUR",
		Status:          "pending",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

	session := newPaidCheckoutSession("cs_test_tampered", "purchase-tampered", 1, "eur")
	payload := buildCheckoutSessionEvent(t, session)
	sigHeader := signPayload(t, payload)

	// The body is swapped after signing: a valid amount is smuggled in with a
	// signature that covers the original body.
	swapped := buildCheckoutSessionEvent(t, newPaidCheckoutSession("cs_test_tampered", "purchase-tampered", 4500, "eur"))

	handler := webhooks.NewStripeWebhookHandler(testWebhookSecret, svc)
	router := newTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(swapped))
	req.Header.Set("Stripe-Signature", sigHeader)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for a tampered payload, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Errorf("tampered payload must never complete a purchase, CompletePurchase called %d times", completeCount)
	}
}
