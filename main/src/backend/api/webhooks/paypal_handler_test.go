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
}

func (s *stubPayPalVerifier) VerifyWebhookSignature(_ context.Context, _ *http.Request, _ string) (*paypal.VerifyWebhookResponse, error) {
	return s.response, s.err
}

func buildPayPalOrderApprovedEvent(t *testing.T, referenceID, amountValue, currencyCode string) []byte {
	t.Helper()
	event := map[string]interface{}{
		"id":            "WH-TEST-123",
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
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal PayPal event: %v", err)
	}
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
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending"},
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
		purchase: &purchases.Purchase{ID: "p1", PriceMinorUnits: 100, Currency: "EUR", Status: "pending"},
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
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-amt-pp", "99.99", "EUR")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for amount mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_CurrencyMismatch(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-cur-pp",
		PriceMinorUnits: 1000,
		Currency:        "EUR",
		Status:          "pending",
	}
	svc := &stubPurchaseService{purchase: purchase}
	verifier := &stubPayPalVerifier{
		response: &paypal.VerifyWebhookResponse{VerificationStatus: "SUCCESS"},
	}

	payload := buildPayPalOrderApprovedEvent(t, "purchase-cur-pp", "10.00", "USD")
	handler := webhooks.NewPayPalWebhookHandler(verifier, "WH-123", svc)
	router := newPayPalTestRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/webhooks/paypal", bytes.NewReader(payload))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for currency mismatch, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPayPalWebhook_AlreadyCompleted(t *testing.T) {
	purchase := &purchases.Purchase{
		ID:              "purchase-done-pp",
		PriceMinorUnits: 5000,
		Currency:        "EUR",
		Status:          "completed",
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
