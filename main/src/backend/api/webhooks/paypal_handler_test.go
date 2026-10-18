package webhooks_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	paypal "github.com/plutov/paypal/v4"

	"ryze/backend/api/webhooks"
	"ryze/backend/services/purchases"
)

// --- stubs ---

type stubPayPalVerifier struct {
	response *paypal.VerifyWebhookResponse
	err      error
	order    *paypal.Order
	orderErr error
}

func (s *stubPayPalVerifier) VerifyWebhookSignature(_ context.Context, _ *http.Request, _ string) (*paypal.VerifyWebhookResponse, error) {
	return s.response, s.err
}

func (s *stubPayPalVerifier) GetOrder(_ context.Context, orderID string) (*paypal.Order, error) {
	if s.orderErr != nil {
		return nil, s.orderErr
	}
	// Fail closed when no order is configured: existing APPROVED tests never
	// reach order resolution, so this keeps them honest instead of returning a
	// dangling success.
	if s.order == nil || s.order.ID != orderID {
		return nil, errors.New("order not found")
	}
	return s.order, nil
}

// buildPayPalOrder returns an order carrying the given purchase reference on
// its single purchase unit, mirroring what the server-side order creation
// writes.
func buildPayPalOrder(orderID, referenceID string) *paypal.Order {
	return &paypal.Order{
		ID: orderID,
		PurchaseUnits: []paypal.PurchaseUnit{
			{ReferenceID: referenceID},
		},
	}
}

func buildPayPalOrderApprovedEvent(t *testing.T, referenceID, amountValue, currencyCode string) []byte {
	t.Helper()
	// The order id is derived deterministically from the reference id so the
	// server-side capture path always has a PayPal order to bind to.
	event := map[string]interface{}{
		"id":            "WH-TEST-123",
		"event_type":    "CHECKOUT.ORDER.APPROVED",
		"resource_type": "order",
		"resource": map[string]interface{}{
			"id":           "ORDER-" + referenceID,
			"reference_id": referenceID,
			"amount": map[string]interface{}{
				"currency_code": currencyCode,
				"value":         amountValue,
			},
		},
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal PayPal event: %v", err)
	}
	return payload
}

// buildPayPalOrderApprovedEventWithoutOrderID builds an approved event whose
// resource carries no PayPal order id, used to exercise the missing-order-id
// guard.
func buildPayPalOrderApprovedEventWithoutOrderID(referenceID, amountValue, currencyCode string) []byte {
	event := map[string]interface{}{
		"id":            "WH-NO-ORDERID",
		"event_type":    "CHECKOUT.ORDER.APPROVED",
		"resource_type": "order",
		"resource": map[string]interface{}{
			"reference_id": referenceID,
			"amount": map[string]interface{}{
				"currency_code": currencyCode,
				"value":         amountValue,
			},
		},
	}
	payload, _ := json.Marshal(event)
	return payload
}

func newPayPalTestRouter(handler *webhooks.PayPalWebhookHandler) *gin.Engine {
	router := gin.New()
	router.POST("/api/v1/webhooks/paypal", handler.Handle)
	return router
}

// --- tests ---

func TestPayPalWebhook_ValidApprovedEvent(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-pp-001",
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-pp-001", "49.99", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_InvalidVerification(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-pp-001",
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "FAILURE"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-pp-001", "49.99", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_VerificationError(t *testing.T) {
	svc := &stubPurchaseService{}
	verifier := &stubPayPalVerifier{
		err: errors.New("paypal API unavailable"),
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-err", "10.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_MalformedPayload(t *testing.T) {
	svc := &stubPurchaseService{}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader([]byte(`not json`)))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_UnsupportedEventType(t *testing.T) {
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending", PaymentMethod: "paypal"},
	}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	event := map[string]interface{}{
		"id":            "WH-UNSUPPORTED",
		"event_type":    "PAYMENT.CAPTURE.REFUNDED",
		"resource_type": "capture",
		"resource":      map[string]interface{}{},
	}
	payload, _ := json.Marshal(event)

	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unsupported event, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_NoReferenceID(t *testing.T) {
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending", PaymentMethod: "paypal"},
	}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	event := map[string]interface{}{
		"id":            "WH-NOREF",
		"event_type":    "CHECKOUT.ORDER.APPROVED",
		"resource_type": "order",
		"resource": map[string]interface{}{
			"reference_id": "",
			"amount": map[string]interface{}{
				"currency_code": "EUR",
				"value":         "1.00",
			},
		},
	}
	payload, _ := json.Marshal(event)

	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for missing reference_id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_MissingOrderID(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-noorder",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	// An APPROVED event without an order id cannot be captured server-side, so
	// the handler must return 500 to trigger provider retry instead of
	// completing without provider-side verification.
	payload := buildPayPalOrderApprovedEventWithoutOrderID("purchase-noorder", "10.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for missing order id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_PurchaseNotFound(t *testing.T) {
	svc := &stubPurchaseService{
		err: purchases.ErrPurchaseNotFound,
	}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "nonexistent", "10.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unknown purchase, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_AmountMismatch(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-amt-pp",
		PriceMinorUnits: 4999,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-amt-pp", "99.99", "EUR")
	records := &recordingOutcomeRecorder{}
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc, webhooks.WithPayPalOutcomeRecorder(records))
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for amount mismatch, got %d: %s", w.Code, w.Body.String())
	}
	if got := records.outcomes()["WH-TEST-123"]; got != "amount_mismatch" {
		t.Fatalf("expected amount_mismatch outcome for the delivered event, got %q", got)
	}
}

func TestPayPalWebhook_CurrencyMismatch(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-cur-pp",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-cur-pp", "10.00", "USD")
	records := &recordingOutcomeRecorder{}
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc, webhooks.WithPayPalOutcomeRecorder(records))
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for currency mismatch, got %d: %s", w.Code, w.Body.String())
	}
	if got := records.outcomes()["WH-TEST-123"]; got != "currency_mismatch" {
		t.Fatalf("expected currency_mismatch outcome for the delivered event, got %q", got)
	}
}

func TestPayPalWebhook_AlreadyCompleted(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-done-pp",
		PriceMinorUnits: 5000,
		Currency:        "EUR",
		Status:          "completed",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-done-pp", "50.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for already completed, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_NotPending(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-fail-pp",
		PriceMinorUnits: 5000,
		Currency:        "EUR",
		Status:          "failed",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-fail-pp", "50.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for not pending, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_CompletePurchaseFailure(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-cpfail-pp",
		PriceMinorUnits: 2000,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{
		purchase: purchase,
		err:      errors.New("database transaction failed"),
	}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-cpfail-pp", "20.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for completion failure, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_DuplicateDeliveryIdempotent(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-dup-pp",
		PriceMinorUnits: 3000,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	completeCount := 0
	svc := &completionCountingService{
		purchase:      purchase,
		completeCount: &completeCount,
	}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-dup-pp", "30.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("attempt %d: expected 200, got %d: %s", i+1, w.Code, w.Body.String())
		}
	}

	if completeCount != 3 {
		t.Errorf("expected CompletePurchase called 3 times, got %d", completeCount)
	}
}

func TestPayPalWebhook_NoWebhookIDConfigured(t *testing.T) {
	svc := &stubPurchaseService{}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-001", "10.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for no webhook ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_UppercaseCurrencyMatch(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-upper-pp",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-upper-pp", "10.00", "eur")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for case-insensitive currency match, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_AmountConversionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		minor    int64
		expected string
	}{
		{"whole euros", 100, "1.00"},
		{"fractional", 4999, "49.99"},
		{"single cent", 1, "0.01"},
		{"ten cents", 10, "0.10"},
		{"large amount", 100000, "1000.00"},
		{"50 cents", 50, "0.50"},
		{"99 cents", 99, "0.99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			purchase := &purchases.Purchase{
				ID:              "purchase-amt-edge",
				PriceMinorUnits: tt.minor,
				Currency:        "EUR",
				Status:          "pending",
				PaymentMethod:   "paypal",
			}
			svc := &stubPurchaseService{purchase: purchase}
			verifier := &stubPayPalVerifier{
				response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
			}

			payload := buildPayPalOrderApprovedEvent(t, "purchase-amt-edge", tt.expected, "EUR")
			handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
			router := newPayPalTestRouter(handler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 for amount %s, got %d: %s", tt.expected, w.Code, w.Body.String())
			}
		})
	}
}

// TestPayPalWebhook_PaymentMethodMismatchDoesNotComplete proves a verified
// PayPal order can never complete a purchase bound to a different payment
// method, mirroring the equivalent guarantee on the Stripe webhook. A PayPal
// order is a real charge, so completing a card or MB WAY purchase from one
// would grant an entitlement that was never paid for by that method.
func TestPayPalWebhook_PaymentMethodMismatchDoesNotComplete(t *testing.T) {
	for _, recorded := range []string{"card", "mbway", ""} {
		t.Run("recorded="+recorded, func(t *testing.T) {
			purchase := &purchases.Purchase{
				ID:              "p1",
				PriceMinorUnits: 100,
				Currency:        "EUR",
				Status:          "pending",
				PaymentMethod:   recorded,
			}
			completeCount := 0
			svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}
			verifier := &stubPayPalVerifier{
				response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
			}

			payload := buildPayPalOrderApprovedEvent(t, "p1", "1.00", "EUR")
			handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
			router := newPayPalTestRouter(handler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
			if w.Body.String() == "completed" {
				t.Error("a method mismatch must never complete a purchase")
			}
			if completeCount != 0 {
				t.Errorf("expected no completion attempt, got %d", completeCount)
			}
		})
	}
}

// TestPayPalWebhook_PayPalMethodCompletes proves the method check does not block
// the legitimate path: an order for a purchase recorded as PayPal still
// completes exactly once.
func TestPayPalWebhook_PayPalMethodCompletes(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p1",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "p1", "1.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "completed" {
		t.Errorf("expected the purchase to be completed, got %q", w.Body.String())
	}
	if completeCount != 1 {
		t.Errorf("expected exactly one completion, got %d", completeCount)
	}
}

// buildPayPalCaptureCompletedEvent builds a verified PAYMENT.CAPTURE.COMPLETED
// settlement referencing the given related order id.
func buildPayPalCaptureCompletedEvent(orderID, amountValue, currencyCode, captureStatus string) []byte {
	resource := map[string]interface{}{
		"id":     "CAP-" + orderID,
		"status": captureStatus,
		"amount": map[string]interface{}{
			"currency_code": currencyCode,
			"value":         amountValue,
		},
	}
	if orderID != "" {
		resource["supplementary_data"] = map[string]interface{}{
			"related_ids": map[string]interface{}{
				"order_id": orderID,
			},
		}
	}
	event := map[string]interface{}{
		"id":         "WH-CAPTURE-" + orderID,
		"event_type": "PAYMENT.CAPTURE.COMPLETED",
		"resource":   resource,
	}
	payload, _ := json.Marshal(event)
	return payload
}

// --- PAYMENT.CAPTURE.COMPLETED ---

// TestPayPalWebhook_CaptureCompletedPendingPurchaseCompletes proves a confirmed
// capture settlement on a still-pending purchase completes it through the pure
// completion path: the funds already moved at PayPal, so no re-capture happens.
func TestPayPalWebhook_CaptureCompletedPendingPurchaseCompletes(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p1",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
		order:    buildPayPalOrder("ORDER-CAPTURE-1", "p1"),
	}

	payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-1", "1.00", "EUR", "COMPLETED")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "completed" {
		t.Fatalf("expected the purchase to be completed, got %q", w.Body.String())
	}
	if completeCount != 1 {
		t.Fatalf("expected exactly one completion, got %d", completeCount)
	}
}

// TestPayPalWebhook_CaptureCompletedFailedPurchaseReconciles proves the
// primary late-settlement path for PayPal: a confirmed capture that settled a
// purchase already superseded (failed) is reconciled to completed.
func TestPayPalWebhook_CaptureCompletedFailedPurchaseReconciles(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p-late-capture",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "failed",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
		order:    buildPayPalOrder("ORDER-CAPTURE-LATE", "p-late-capture"),
	}

	payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-LATE", "1.00", "EUR", "COMPLETED")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "reconciled" {
		t.Fatalf("expected the capture to reconcile, got %q", w.Body.String())
	}
	if svc.reconcileCount != 1 {
		t.Fatalf("expected exactly one reconciliation, got %d", svc.reconcileCount)
	}
}

// TestPayPalWebhook_CaptureCompletedFailedPurchaseConflictAcknowledged proves a
// capture that cannot define a second sale is acknowledged so the provider
// stops redelivering.
func TestPayPalWebhook_CaptureCompletedFailedPurchaseConflictAcknowledged(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p-capture-conflict",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "failed",
		PaymentMethod:   "paypal",
	}
	svc := &stubPurchaseService{purchase: purchase, reconcileErr: purchases.ErrReconciliationConflict}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
		order:    buildPayPalOrder("ORDER-CAPTURE-CONFLICT", "p-capture-conflict"),
	}

	payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-CONFLICT", "1.00", "EUR", "COMPLETED")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.reconcileCount != 1 {
		t.Fatalf("expected one reconciliation attempt, got %d", svc.reconcileCount)
	}
}

// TestPayPalWebhook_CaptureCompletedAlreadyCompletedIsIdempotent proves a
// repeated capture delivery for a completed purchase is acknowledged without
// re-completing or re-reconciling.
func TestPayPalWebhook_CaptureCompletedAlreadyCompletedIsIdempotent(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p-capture-done",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "completed",
		PaymentMethod:   "paypal",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
		order:    buildPayPalOrder("ORDER-CAPTURE-DONE", "p-capture-done"),
	}

	payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-DONE", "1.00", "EUR", "COMPLETED")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "already completed" {
		t.Fatalf("expected an idempotent acknowledgement, got %q", w.Body.String())
	}
	if completeCount != 0 {
		t.Fatalf("expected no completion, got %d", completeCount)
	}
}

// TestPayPalWebhook_CaptureEventStatusIsEnforced proves only a COMPLETED
// capture resource is a settlement; any other status proves nothing and is
// acknowledged without touching the purchase.
func TestPayPalWebhook_CaptureEventStatusIsEnforced(t *testing.T) {
	for _, status := range []string{"", "PENDING", "FAILED", "REFUNDED"} {
		t.Run("status="+status, func(t *testing.T) {
			purchase := &purchases.Purchase{
				ID:              "p-capture-status",
				PriceMinorUnits: 100,
				Currency:        "EUR",
				Status:          "pending",
				PaymentMethod:   "paypal",
			}
			completeCount := 0
			svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}
			verifier := &stubPayPalVerifier{
				response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
				order:    buildPayPalOrder("ORDER-CAPTURE-STATUS", "p-capture-status"),
			}

			payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-STATUS", "1.00", "EUR", status)
			handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
			router := newPayPalTestRouter(handler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
			if w.Body.String() == "completed" || w.Body.String() == "reconciled" {
				t.Fatalf("a %q capture must never settle a purchase, got %q", status, w.Body.String())
			}
			if completeCount != 0 {
				t.Fatalf("expected no completion, got %d", completeCount)
			}
		})
	}
}

// TestPayPalWebhook_CaptureEventWithoutRelatedOrderIDIsIgnored proves a capture
// settlement with no traceable order cannot be verified and is ignored, not
// answered with a retry the provider could never satisfy.
func TestPayPalWebhook_CaptureEventWithoutRelatedOrderIDIsIgnored(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p-capture-noorder",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
		order:    buildPayPalOrder("ORDER-UNUSED", "p-capture-noorder"),
	}

	payload := buildPayPalCaptureCompletedEvent("", "1.00", "EUR", "COMPLETED")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if completeCount != 0 {
		t.Fatalf("expected no completion, got %d", completeCount)
	}
}

// TestPayPalWebhook_CaptureOrderResolutionFailureRetries proves an order-load
// failure (API unavailable) is answered with 500 so the provider redelivers;
// the settlement must never be guessed from the event payload alone.
func TestPayPalWebhook_CaptureOrderResolutionFailureRetries(t *testing.T) {
	for name, verifier := range map[string]*stubPayPalVerifier{
		"api-unavailable": {
			response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
			orderErr: errors.New("paypal API unavailable"),
		},
		"order-missing": {
			response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			purchase := &purchases.Purchase{
				ID:              "p-capture-orderfail",
				PriceMinorUnits: 100,
				Currency:        "EUR",
				Status:          "pending",
				PaymentMethod:   "paypal",
			}
			completeCount := 0
			svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}

			payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-ORDERFAIL", "1.00", "EUR", "COMPLETED")
			handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
			router := newPayPalTestRouter(handler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
			router.ServeHTTP(w, req)

			if w.Code != http.StatusInternalServerError {
				t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
			}
			if completeCount != 0 {
				t.Fatalf("expected no completion without order resolution, got %d", completeCount)
			}
		})
	}
}

// TestPayPalWebhook_CaptureAmountMismatchRetries proves the captured amount is
// verified against the immutable snapshot before any completion; a mismatch is
// a provider-integration signal, so it is answered with 500.
func TestPayPalWebhook_CaptureAmountMismatchRetries(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p-capture-amount",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
		order:    buildPayPalOrder("ORDER-CAPTURE-AMOUNT", "p-capture-amount"),
	}

	payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-AMOUNT", "9.99", "EUR", "COMPLETED")
	records := &recordingOutcomeRecorder{}
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc, webhooks.WithPayPalOutcomeRecorder(records))
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for amount mismatch, got %d: %s", w.Code, w.Body.String())
	}
	if got := records.outcomes()["WH-CAPTURE-ORDER-CAPTURE-AMOUNT"]; got != "amount_mismatch" {
		t.Fatalf("expected amount_mismatch outcome for the delivered event, got %q", got)
	}
	if completeCount != 0 {
		t.Fatalf("expected no completion on an amount mismatch, got %d", completeCount)
	}
}

// TestPayPalWebhook_CaptureMethodMismatchDoesNotComplete proves a verified
// capture can never settle a purchase bound to a different method, mirroring
// the APPROVED path.
func TestPayPalWebhook_CaptureMethodMismatchDoesNotComplete(t *testing.T) {
	for _, recorded := range []string{"card", "mbway", ""} {
		t.Run("recorded="+recorded, func(t *testing.T) {
			purchase := &purchases.Purchase{
				ID:              "p-capture-method",
				PriceMinorUnits: 100,
				Currency:        "EUR",
				Status:          "pending",
				PaymentMethod:   recorded,
			}
			completeCount := 0
			svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}
			verifier := &stubPayPalVerifier{
				response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
				order:    buildPayPalOrder("ORDER-CAPTURE-METHOD", "p-capture-method"),
			}

			payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-METHOD", "1.00", "EUR", "COMPLETED")
			handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
			router := newPayPalTestRouter(handler)

			w := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
			if w.Body.String() == "completed" || w.Body.String() == "reconciled" {
				t.Fatal("a method mismatch must never settle a purchase")
			}
			if completeCount != 0 {
				t.Fatalf("expected no completion, got %d", completeCount)
			}
		})
	}
}

// TestPayPalWebhook_CaptureUnknownPurchaseIsIgnored proves a capture for an
// order the backend cannot find is acknowledged: it cannot be granted, and
// a retry would never change that. This is the "no record" settlement.
func TestPayPalWebhook_CaptureUnknownPurchaseIsIgnored(t *testing.T) {
	svc := &stubPurchaseService{err: purchases.ErrPurchaseNotFound}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
		order:    buildPayPalOrder("ORDER-CAPTURE-UNKNOWN", "p-does-not-exist"),
	}

	payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-UNKNOWN", "1.00", "EUR", "COMPLETED")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- late approvals ---

// TestPayPalWebhook_LateApprovalOnFailedPurchaseIsNotCaptured proves a late
// APPROVAL is consent, not a payment: the order is deliberately left
// uncaptured, so no funds move, and the acknowledgement is purely diagnostic.
func TestPayPalWebhook_LateApprovalOnFailedPurchaseIsNotCaptured(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p-late-approval",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "failed",
		PaymentMethod:   "paypal",
	}
	completeCount := 0
	svc := &completionCountingService{purchase: purchase, completeCount: &completeCount}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "p-late-approval", "1.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "late approval acknowledged" {
		t.Fatalf("expected a late-approval acknowledgement, got %q", w.Body.String())
	}
	if completeCount != 0 {
		t.Fatalf("a late approval must never capture or complete, got %d attempts", completeCount)
	}
}

// --- resolved pairs ---

// TestPayPalWebhook_ApprovedDuplicateEntitlementIsAcknowledged proves an
// approved order whose pair already resolved another way (a sibling purchase
// completed first) is acknowledged with 200 and recorded as a resolved
// conflict, instead of being retried forever for a duplicate sale that can
// never succeed.
func TestPayPalWebhook_ApprovedDuplicateEntitlementIsAcknowledged(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p-dup-approved",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	records := &recordingOutcomeRecorder{}
	svc := &stubPurchaseService{purchase: purchase, completeErr: purchases.ErrDuplicateEntitlement}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "p-dup-approved", "1.00", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc, webhooks.WithPayPalOutcomeRecorder(records))
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for a resolved pair, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "pair already resolved" {
		t.Fatalf("expected a resolved-pair acknowledgement, got %q", w.Body.String())
	}
	if got := records.outcomes()["WH-TEST-123"]; got != "conflict_pair_resolved" {
		t.Fatalf("expected a conflict_pair_resolved outcome, got %q", got)
	}
}

// TestPayPalWebhook_CaptureCompletedDuplicateEntitlementIsAcknowledged proves
// the captured-payment completion path treats a resolved pair the same way:
// the funds settled, but the pair already owns access, so the delivery is
// final and recorded instead of retried.
func TestPayPalWebhook_CaptureCompletedDuplicateEntitlementIsAcknowledged(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "p-dup-capture",
		PriceMinorUnits: 100,
		Currency:        "EUR",
		Status:          "pending",
		PaymentMethod:   "paypal",
	}
	records := &recordingOutcomeRecorder{}
	svc := &stubPurchaseService{purchase: purchase, completeErr: purchases.ErrDuplicateEntitlement}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
		order:    buildPayPalOrder("ORDER-CAPTURE-DUP", "p-dup-capture"),
	}

	payload := buildPayPalCaptureCompletedEvent("ORDER-CAPTURE-DUP", "1.00", "EUR", "COMPLETED")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc, webhooks.WithPayPalOutcomeRecorder(records))
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for a resolved pair, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "pair already resolved" {
		t.Fatalf("expected a resolved-pair acknowledgement, got %q", w.Body.String())
	}
	if got := records.outcomes()["WH-CAPTURE-ORDER-CAPTURE-DUP"]; got != "conflict_pair_resolved" {
		t.Fatalf("expected a conflict_pair_resolved outcome, got %q", got)
	}
}
