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
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

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

	session := stripe.CheckoutSession{
		ID:          "cs_test_abc123",
		AmountTotal: 4999,
		Currency:    "eur",
		Metadata:    map[string]string{"purchase_id": "purchase-001"},
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
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStripeWebhook_InvalidSignature(t *testing.T) {
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending"},
	}

	session := stripe.CheckoutSession{
		ID:          "cs_test",
		AmountTotal: 100,
		Currency:    "eur",
		Metadata:    map[string]string{"purchase_id": "p1"},
	}
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
		ID:          "cs_test_nopurchase",
		AmountTotal: 100,
		Currency:    "eur",
		Metadata:    map[string]string{},
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

	session := stripe.CheckoutSession{
		ID:          "cs_test_notfound",
		AmountTotal: 100,
		Currency:    "eur",
		Metadata:    map[string]string{"purchase_id": "nonexistent"},
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

	session := stripe.CheckoutSession{
		ID:          "cs_test_amt",
		AmountTotal: 9999,
		Currency:    "eur",
		Metadata:    map[string]string{"purchase_id": "purchase-amt"},
	}
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

	session := stripe.CheckoutSession{
		ID:          "cs_test_cur",
		AmountTotal: 1000,
		Currency:    "usd",
		Metadata:    map[string]string{"purchase_id": "purchase-cur"},
	}
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

	session := stripe.CheckoutSession{
		ID:          "cs_test_done",
		AmountTotal: 5000,
		Currency:    "eur",
		Metadata:    map[string]string{"purchase_id": "purchase-done"},
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

	session := stripe.CheckoutSession{
		ID:          "cs_test_failed",
		AmountTotal: 5000,
		Currency:    "eur",
		Metadata:    map[string]string{"purchase_id": "purchase-failed"},
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

	session := stripe.CheckoutSession{
		ID:          "cs_test_compfail",
		AmountTotal: 2000,
		Currency:    "eur",
		Metadata:    map[string]string{"purchase_id": "purchase-fail"},
	}
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

	session := stripe.CheckoutSession{
		ID:          "cs_test_dup",
		AmountTotal: 3000,
		Currency:    "eur",
		Metadata:    map[string]string{"purchase_id": "purchase-dup"},
	}
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
		Type:       stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded,
		APIVersion: stripe.APIVersion,
		Data:       &stripe.EventData{},
	}
	session := stripe.CheckoutSession{
		ID:          "cs_test_async",
		AmountTotal: 1500,
		Currency:    "eur",
		Metadata:    map[string]string{"purchase_id": "purchase-async"},
	}
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

	session := stripe.CheckoutSession{
		ID:          "cs_test_case",
		AmountTotal: 1000,
		Currency:    "EUR",
		Metadata:    map[string]string{"purchase_id": "purchase-case"},
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
