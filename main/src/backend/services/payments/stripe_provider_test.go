package payments_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	stripe "github.com/stripe/stripe-go/v82"

	"ryze/backend/services/payments"
)

// mockStripeSessionResponse returns a JSON payload simulating a Stripe Checkout
// Session creation response for the given session ID and checkout URL.
func mockStripeSessionResponse(sessionID, checkoutURL string) []byte {
	resp := map[string]interface{}{
		"id":                  sessionID,
		"object":              "checkout.session",
		"status":              "open",
		"url":                 checkoutURL,
		"payment_status":      "unpaid",
		"mode":                "payment",
		"client_reference_id": "test-purchase-123",
	}
	b, _ := json.Marshal(resp)
	return b
}

// setupMockStripeServer creates a test HTTP server that mimics the Stripe API
// and configures the global Stripe SDK backend to point to it. It returns the
// server (which must be closed by the caller) and a function to restore the
// original Stripe backend.
func setupMockStripeServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, func()) {
	t.Helper()

	server := httptest.NewServer(handler)

	originalBackend := stripe.GetBackend(stripe.APIBackend)
	stripe.Key = "sk_test_fake_key"

	testBackend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		URL:        stripe.String(server.URL + "/"),
		HTTPClient: server.Client(),
	})
	stripe.SetBackend(stripe.APIBackend, testBackend)

	cleanup := func() {
		server.Close()
		stripe.SetBackend(stripe.APIBackend, originalBackend)
		stripe.Key = ""
	}

	return server, cleanup
}

func TestStripeProvider_Success(t *testing.T) {
	sessionID := "cs_test_abc123"
	checkoutURL := "https://checkout.stripe.com/c/pay/cs_test_abc123"

	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/checkout/sessions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("expected Bearer token, got %s", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(mockStripeSessionResponse(sessionID, checkoutURL)))
	})
	defer cleanup()

	provider := payments.NewStripeProvider("https://example.com/success", "https://example.com/cancel")

	result, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "test-purchase-123",
		AmountMinorUnits: 4999,
		Currency:         "EUR",
		ProgramID:        "prog-abc",
		Method:           payments.PaymentMethodCard,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentID != sessionID {
		t.Errorf("expected PaymentID %q, got %q", sessionID, result.PaymentID)
	}
	if result.CheckoutURL != checkoutURL {
		t.Errorf("expected CheckoutURL %q, got %q", checkoutURL, result.CheckoutURL)
	}
	if result.Provider != "stripe" {
		t.Errorf("expected Provider %q, got %q", "stripe", result.Provider)
	}
	if result.PurchaseID != "test-purchase-123" {
		t.Errorf("expected PurchaseID %q, got %q", "test-purchase-123", result.PurchaseID)
	}
	if result.Status != payments.PaymentStatusRequiresAction {
		t.Errorf("expected Status %q, got %q", payments.PaymentStatusRequiresAction, result.Status)
	}
}

func TestStripeProvider_EmptyPurchaseID(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach Stripe API")
	})
	defer cleanup()

	provider := payments.NewStripeProvider("", "")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		AmountMinorUnits: 100,
		Currency:         "EUR",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodCard,
	})

	if err == nil {
		t.Fatal("expected error for empty purchase ID")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

func TestStripeProvider_ZeroAmount(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach Stripe API")
	})
	defer cleanup()

	provider := payments.NewStripeProvider("", "")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-1",
		AmountMinorUnits: 0,
		Currency:         "EUR",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodCard,
	})

	if err == nil {
		t.Fatal("expected error for zero amount")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

func TestStripeProvider_EmptyCurrency(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach Stripe API")
	})
	defer cleanup()

	provider := payments.NewStripeProvider("", "")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-1",
		AmountMinorUnits: 100,
		Currency:         "",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodCard,
	})

	if err == nil {
		t.Fatal("expected error for empty currency")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

func TestStripeProvider_StripeAPIError(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"type":"invalid_request_error","message":"Invalid currency"}}`)
	})
	defer cleanup()

	provider := payments.NewStripeProvider("https://example.com/success", "https://example.com/cancel")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-1",
		AmountMinorUnits: 100,
		Currency:         "EUR",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodCard,
	})

	if err == nil {
		t.Fatal("expected error for Stripe API failure")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

func TestStripeProvider_EmptyCheckoutURL(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":     "cs_test_nourl",
			"object": "checkout.session",
			"status": "open",
		}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(b))
	})
	defer cleanup()

	provider := payments.NewStripeProvider("https://example.com/success", "https://example.com/cancel")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-1",
		AmountMinorUnits: 100,
		Currency:         "EUR",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodCard,
	})

	if err == nil {
		t.Fatal("expected error for missing checkout URL")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

func TestStripeProvider_IdempotencyKey(t *testing.T) {
	sessionID := "cs_test_idempotent"
	checkoutURL := "https://checkout.stripe.com/c/pay/cs_test_idempotent"
	purchaseID := "purchase-idem-42"

	var capturedIDempotencyKey string

	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedIDempotencyKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(mockStripeSessionResponse(sessionID, checkoutURL)))
	})
	defer cleanup()

	provider := payments.NewStripeProvider("https://example.com/success", "https://example.com/cancel")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       purchaseID,
		AmountMinorUnits: 2999,
		Currency:         "EUR",
		ProgramID:        "prog-idem",
		Method:           payments.PaymentMethodCard,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedKey := fmt.Sprintf("ryze-purchase-%s", purchaseID)
	if capturedIDempotencyKey != expectedKey {
		t.Errorf("expected idempotency key %q, got %q", expectedKey, capturedIDempotencyKey)
	}
}

func TestStripeProvider_NoURLs(t *testing.T) {
	sessionID := "cs_test_nourls"
	checkoutURL := "https://checkout.stripe.com/c/pay/cs_test_nourls"

	var capturedBody string

	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			capturedBody = string(body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(mockStripeSessionResponse(sessionID, checkoutURL)))
	})
	defer cleanup()

	provider := payments.NewStripeProvider("", "")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-nourls",
		AmountMinorUnits: 500,
		Currency:         "usd",
		ProgramID:        "prog-nourls",
		Method:           payments.PaymentMethodCard,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(capturedBody, "success_url") {
		t.Errorf("expected no success_url when empty, body contains: %s", capturedBody)
	}
	if strings.Contains(capturedBody, "cancel_url") {
		t.Errorf("expected no cancel_url when empty, body contains: %s", capturedBody)
	}
}

func TestStripeProvider_MBWayMethod(t *testing.T) {
	sessionID := "cs_test_mbway123"
	checkoutURL := "https://checkout.stripe.com/c/pay/cs_test_mbway123"

	var capturedBody string

	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(mockStripeSessionResponse(sessionID, checkoutURL)))
	})
	defer cleanup()

	provider := payments.NewStripeProvider("https://example.com/success", "https://example.com/cancel")
	result, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-mbway",
		AmountMinorUnits: 2500,
		Currency:         "EUR",
		ProgramID:        "prog-mbway",
		Method:           payments.PaymentMethodMBWay,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentID != sessionID {
		t.Errorf("expected PaymentID %q, got %q", sessionID, result.PaymentID)
	}
	if !strings.Contains(capturedBody, "mb_way") {
		t.Errorf("expected mb_way in payment_method_types, body: %s", capturedBody)
	}
	if strings.Contains(capturedBody, "payment_method_types") && strings.Contains(capturedBody, "payment_method_types%5B0%5D=card") {
		t.Errorf("expected card not present when mbway selected, body: %s", capturedBody)
	}
}

func TestStripeProvider_PaymentMethodTypesCard(t *testing.T) {
	sessionID := "cs_test_card"
	checkoutURL := "https://checkout.stripe.com/c/pay/cs_test_card"

	var capturedBody string

	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(mockStripeSessionResponse(sessionID, checkoutURL)))
	})
	defer cleanup()

	provider := payments.NewStripeProvider("https://example.com/success", "https://example.com/cancel")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-card",
		AmountMinorUnits: 5000,
		Currency:         "EUR",
		ProgramID:        "prog-card",
		Method:           payments.PaymentMethodCard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedBody, "payment_method_types[0]=card") {
		t.Errorf("expected card in payment_method_types, body: %s", capturedBody)
	}
}

func TestStripeProvider_MetadataIncludesMethod(t *testing.T) {
	sessionID := "cs_test_meta"
	checkoutURL := "https://checkout.stripe.com/c/pay/cs_test_meta"

	var capturedBody string

	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(mockStripeSessionResponse(sessionID, checkoutURL)))
	})
	defer cleanup()

	provider := payments.NewStripeProvider("https://example.com/success", "https://example.com/cancel")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-meta",
		AmountMinorUnits: 3000,
		Currency:         "EUR",
		ProgramID:        "prog-meta",
		Method:           payments.PaymentMethodMBWay,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedBody, "metadata[method]=mbway") {
		t.Errorf("expected method=mbway in metadata, body: %s", capturedBody)
	}
}
