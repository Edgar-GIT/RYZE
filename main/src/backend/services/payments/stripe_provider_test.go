package payments_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	stripe "github.com/stripe/stripe-go/v86"

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

func TestStripeProvider_CurrencyComesFromSnapshot(t *testing.T) {
	// The session currency must be the one recorded in the purchase snapshot.
	// Hard-coding a currency would let a purchase be paid in a currency it is
	// never verified against, making the payment impossible to complete.
	var capturedBody string

	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(mockStripeSessionResponse("cs_test_ccy", "https://checkout.stripe.com/c/pay/cs_test_ccy")))
	})
	defer cleanup()

	provider := payments.NewStripeProvider("", "")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-ccy",
		AmountMinorUnits: 100,
		Currency:         "GBP",
		ProgramID:        "prog-ccy",
		Method:           payments.PaymentMethodCard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedBody, "currency]=gbp") {
		t.Errorf("expected the snapshot currency in the request, got: %s", capturedBody)
	}
	if strings.Contains(capturedBody, "currency]=eur") {
		t.Errorf("expected no hard-coded currency, got: %s", capturedBody)
	}
}

func TestStripeProvider_UnusableCurrencyIsRejectedBeforeCallingStripe(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("must not reach the Stripe API for an unusable currency")
	})
	defer cleanup()

	provider := payments.NewStripeProvider("", "")

	for name, currency := range map[string]string{
		"too long":  "euro",
		"too short": "eu",
		"not alpha": "e1r",
	} {
		_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
			PurchaseID:       "purchase-ccy-invalid",
			AmountMinorUnits: 100,
			Currency:         currency,
			ProgramID:        "prog-ccy-invalid",
			Method:           payments.PaymentMethodCard,
		})
		if !errors.Is(err, payments.ErrProviderFailure) {
			t.Errorf("%s: expected ErrProviderFailure, got: %v", name, err)
		}
	}
}

func TestStripeProvider_NegativeAmount(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach Stripe API")
	})
	defer cleanup()

	provider := payments.NewStripeProvider("", "")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-neg",
		AmountMinorUnits: -500,
		Currency:         "EUR",
		ProgramID:        "prog-neg",
		Method:           payments.PaymentMethodCard,
	})

	if err == nil {
		t.Fatal("expected error for negative amount")
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

func TestStripeProvider_MBWayMethodIsRejected(t *testing.T) {
	// MB WAY has no provider. The Stripe provider must refuse it instead of
	// sending an unsupported payment method type to Stripe, so an unimplemented
	// method is never presented to a buyer.
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("must not reach the Stripe API for an unsupported payment method")
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

	if err == nil {
		t.Fatal("expected error for the unimplemented mbway method")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("expected ErrProviderFailure, got %v", err)
	}
	if result.PaymentID != "" || result.CheckoutURL != "" {
		t.Errorf("expected no payment result, got %+v", result)
	}
}

func TestStripeProvider_PayPalMethodIsRejected(t *testing.T) {
	// The Stripe provider serves the card method only; the PayPal method is
	// owned by the PayPal provider and must never be routed here.
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("must not reach the Stripe API for a provider-owned method")
	})
	defer cleanup()

	provider := payments.NewStripeProvider("https://example.com/success", "https://example.com/cancel")
	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-paypal",
		AmountMinorUnits: 2500,
		Currency:         "EUR",
		ProgramID:        "prog-paypal",
		Method:           payments.PaymentMethodPayPal,
	})

	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("expected ErrProviderFailure, got %v", err)
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

func TestStripeProvider_SendsNoCardDataToStripe(t *testing.T) {
	sessionID := "cs_test_nocarddata"
	checkoutURL := "https://checkout.stripe.com/c/pay/cs_test_nocarddata"

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
		PurchaseID:       "purchase-nocarddata",
		AmountMinorUnits: 3000,
		Currency:         "EUR",
		ProgramID:        "prog-nocarddata",
		Method:           payments.PaymentMethodCard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Card data is entered exclusively on the Stripe-hosted page, so no PAN,
	// CVC, expiry or card token may ever appear in a RYZE request.
	lowered := strings.ToLower(capturedBody)
	for _, forbidden := range []string{
		"card_number", "card[number]", "number", "cvc", "cvv",
		"exp_month", "exp_year", "card_token", "payment_method_data",
	} {
		if strings.Contains(lowered, forbidden) {
			t.Errorf("expected no card data field %q in the Stripe request, body: %s", forbidden, capturedBody)
		}
	}
}

func TestStripeProvider_CardAmountFromSnapshot(t *testing.T) {
	sessionID := "cs_test_amt_card"
	checkoutURL := "https://checkout.stripe.com/c/pay/cs_test_amt_card"

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
		PurchaseID:       "purchase-amt-card",
		AmountMinorUnits: 7500,
		Currency:         "EUR",
		ProgramID:        "prog-amt-card",
		Method:           payments.PaymentMethodCard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedBody, "unit_amount]=7500") {
		t.Errorf("expected unit_amount=7500 in body, got: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "currency]=eur") {
		t.Errorf("expected currency=eur in body, got: %s", capturedBody)
	}
}

func TestStripeProvider_PurchaseIDInMetadata(t *testing.T) {
	sessionID := "cs_test_pid"
	checkoutURL := "https://checkout.stripe.com/c/pay/cs_test_pid"

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
		PurchaseID:       "purchase-meta-pid",
		AmountMinorUnits: 2000,
		Currency:         "EUR",
		ProgramID:        "prog-meta-pid",
		Method:           payments.PaymentMethodCard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedBody, "metadata[purchase_id]=purchase-meta-pid") {
		t.Errorf("expected purchase_id in metadata, body: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "metadata[program_id]=prog-meta-pid") {
		t.Errorf("expected program_id in metadata, body: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "client_reference_id=purchase-meta-pid") {
		t.Errorf("expected client_reference_id, body: %s", capturedBody)
	}
}

// mockStripeRetrievedSession returns a JSON payload simulating a Stripe
// Checkout Session retrieval response, i.e. the server-side view of a session.
func mockStripeRetrievedSession(sessionID, purchaseID string, amount int64, currency, paymentStatus, mode string) []byte {
	resp := map[string]interface{}{
		"id":                  sessionID,
		"object":              "checkout.session",
		"status":              "complete",
		"url":                 "https://checkout.stripe.com/c/pay/" + sessionID,
		"payment_status":      paymentStatus,
		"mode":                mode,
		"amount_total":        amount,
		"currency":            currency,
		"client_reference_id": purchaseID,
		"metadata":            map[string]string{"purchase_id": purchaseID},
	}
	b, _ := json.Marshal(resp)
	return b
}

// serveStripeSessionRetrieval answers only the session retrieval endpoint and
// records the requested session ID, failing the test on any other call.
func serveStripeSessionRetrieval(t *testing.T, requestedID *string, body []byte) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/checkout/sessions/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		*requestedID = strings.TrimPrefix(r.URL.Path, "/v1/checkout/sessions/")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}
}

func captureRequest(purchaseID, sessionID string) payments.CaptureRequest {
	return payments.CaptureRequest{
		PurchaseID:       purchaseID,
		PaymentID:        sessionID,
		AmountMinorUnits: 4999,
		Currency:         "EUR",
	}
}

func TestStripeProvider_CaptureSuccess(t *testing.T) {
	var requestedID string
	_, cleanup := setupMockStripeServer(t, serveStripeSessionRetrieval(t, &requestedID,
		mockStripeRetrievedSession("cs_test_cap", "purchase-cap", 4999, "eur", "paid", "payment")))
	defer cleanup()

	provider := payments.NewStripeProvider("", "")

	result, err := provider.CapturePayment(context.Background(), captureRequest("purchase-cap", "cs_test_cap"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestedID != "cs_test_cap" {
		t.Errorf("expected the session to be reloaded by ID, requested %q", requestedID)
	}
	if result.PaymentID != "cs_test_cap" {
		t.Errorf("expected PaymentID %q, got %q", "cs_test_cap", result.PaymentID)
	}
	if result.Provider != "stripe" {
		t.Errorf("expected Provider %q, got %q", "stripe", result.Provider)
	}
}

func TestStripeProvider_CaptureIsIdempotent(t *testing.T) {
	var requestedID string
	_, cleanup := setupMockStripeServer(t, serveStripeSessionRetrieval(t, &requestedID,
		mockStripeRetrievedSession("cs_test_idem", "purchase-idem", 4999, "eur", "paid", "payment")))
	defer cleanup()

	provider := payments.NewStripeProvider("", "")

	// The browser return leg and the webhook race: verification is read-only,
	// so repeating it keeps succeeding and never charges twice.
	for i := 0; i < 3; i++ {
		if _, err := provider.CapturePayment(context.Background(), captureRequest("purchase-idem", "cs_test_idem")); err != nil {
			t.Fatalf("attempt %d: unexpected error: %v", i+1, err)
		}
	}
}

func TestStripeProvider_CaptureRejectsUnpaidSession(t *testing.T) {
	for _, status := range []string{"unpaid", "no_payment_required", ""} {
		var requestedID string
		_, cleanup := setupMockStripeServer(t, serveStripeSessionRetrieval(t, &requestedID,
			mockStripeRetrievedSession("cs_test_unpaid", "purchase-cap", 4999, "eur", status, "payment")))

		provider := payments.NewStripeProvider("", "")
		_, err := provider.CapturePayment(context.Background(), captureRequest("purchase-cap", "cs_test_unpaid"))
		if !errors.Is(err, payments.ErrProviderFailure) {
			t.Errorf("payment_status %q: expected ErrProviderFailure, got %v", status, err)
		}
		cleanup()
	}
}

func TestStripeProvider_CaptureRejectsAmountMismatch(t *testing.T) {
	var requestedID string
	_, cleanup := setupMockStripeServer(t, serveStripeSessionRetrieval(t, &requestedID,
		mockStripeRetrievedSession("cs_test_amt", "purchase-cap", 1, "eur", "paid", "payment")))
	defer cleanup()

	provider := payments.NewStripeProvider("", "")

	_, err := provider.CapturePayment(context.Background(), captureRequest("purchase-cap", "cs_test_amt"))
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("expected ErrProviderFailure for an amount mismatch, got %v", err)
	}
}

func TestStripeProvider_CaptureRejectsCurrencyMismatch(t *testing.T) {
	var requestedID string
	_, cleanup := setupMockStripeServer(t, serveStripeSessionRetrieval(t, &requestedID,
		mockStripeRetrievedSession("cs_test_cur", "purchase-cap", 4999, "usd", "paid", "payment")))
	defer cleanup()

	provider := payments.NewStripeProvider("", "")

	_, err := provider.CapturePayment(context.Background(), captureRequest("purchase-cap", "cs_test_cur"))
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("expected ErrProviderFailure for a currency mismatch, got %v", err)
	}
}

func TestStripeProvider_CaptureRejectsAnotherPurchaseSession(t *testing.T) {
	// A session created for a different purchase must never verify, even for a
	// user who owns both purchases.
	var requestedID string
	_, cleanup := setupMockStripeServer(t, serveStripeSessionRetrieval(t, &requestedID,
		mockStripeRetrievedSession("cs_test_other", "purchase-someone-else", 4999, "eur", "paid", "payment")))
	defer cleanup()

	provider := payments.NewStripeProvider("", "")

	_, err := provider.CapturePayment(context.Background(), captureRequest("purchase-cap", "cs_test_other"))
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("expected ErrProviderFailure for a foreign session, got %v", err)
	}
}

func TestStripeProvider_CaptureRejectsNonPaymentMode(t *testing.T) {
	var requestedID string
	_, cleanup := setupMockStripeServer(t, serveStripeSessionRetrieval(t, &requestedID,
		mockStripeRetrievedSession("cs_test_sub", "purchase-cap", 4999, "eur", "paid", "subscription")))
	defer cleanup()

	provider := payments.NewStripeProvider("", "")

	_, err := provider.CapturePayment(context.Background(), captureRequest("purchase-cap", "cs_test_sub"))
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("expected ErrProviderFailure for subscription mode, got %v", err)
	}
}

func TestStripeProvider_CaptureValidatesRequestBeforeCallingStripe(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("must not reach the Stripe API for an invalid capture request")
	})
	defer cleanup()

	provider := payments.NewStripeProvider("", "")

	cases := map[string]payments.CaptureRequest{
		"empty purchase": {PaymentID: "cs_test", AmountMinorUnits: 100, Currency: "EUR"},
		"empty payment":  {PurchaseID: "purchase-cap", AmountMinorUnits: 100, Currency: "EUR"},
		"zero amount":    {PurchaseID: "purchase-cap", PaymentID: "cs_test", Currency: "EUR"},
		"empty currency": {PurchaseID: "purchase-cap", PaymentID: "cs_test", AmountMinorUnits: 100},
	}
	for name, request := range cases {
		if _, err := provider.CapturePayment(context.Background(), request); !errors.Is(err, payments.ErrProviderFailure) {
			t.Errorf("%s: expected ErrProviderFailure, got %v", name, err)
		}
	}
}

func TestStripeProvider_CaptureSurfacesStripeAPIFailure(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"type":"invalid_request_error","message":"No such checkout.session"}}`)
	})
	defer cleanup()

	provider := payments.NewStripeProvider("", "")

	_, err := provider.CapturePayment(context.Background(), captureRequest("purchase-cap", "cs_test_missing"))
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("expected ErrProviderFailure for an unknown session, got %v", err)
	}
}

func TestStripeProvider_CaptureFailsWhenNotConfigured(t *testing.T) {
	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("must not reach the Stripe API when the provider is not configured")
	})
	defer cleanup()

	// The mock harness installs a key; removing it models a missing
	// configuration, which must fail closed rather than call Stripe.
	stripe.Key = ""

	provider := payments.NewStripeProvider("", "")

	if _, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-cap",
		AmountMinorUnits: 100,
		Currency:         "EUR",
		Method:           payments.PaymentMethodCard,
	}); !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("initiation: expected ErrProviderFailure when unconfigured, got %v", err)
	}

	if _, err := provider.CapturePayment(context.Background(), captureRequest("purchase-cap", "cs_test")); !errors.Is(err, payments.ErrProviderFailure) {
		t.Fatalf("capture: expected ErrProviderFailure when unconfigured, got %v", err)
	}
}

func TestStripeProvider_SubstitutesRedirectPlaceholders(t *testing.T) {
	var capturedBody string

	_, cleanup := setupMockStripeServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(mockStripeSessionResponse("cs_test_urls", "https://checkout.stripe.com/c/pay/cs_test_urls")))
	})
	defer cleanup()

	provider := payments.NewStripeProvider(
		"https://ryze.example/en/programs/{program_id}/purchase/success?session_id={CHECKOUT_SESSION_ID}&purchase_id={purchase_id}",
		"https://ryze.example/en/programs/{program_id}/purchase?canceled=1&purchase_id={purchase_id}",
	)

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-urls",
		AmountMinorUnits: 4999,
		Currency:         "EUR",
		ProgramID:        "prog-urls",
		Method:           payments.PaymentMethodCard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The Stripe SDK sends a form-encoded body, so the URLs are percent-encoded.
	decoded, unescapeErr := url.QueryUnescape(capturedBody)
	if unescapeErr != nil {
		t.Fatalf("failed to decode request body: %v", unescapeErr)
	}

	// Per-purchase placeholders are substituted, and Stripe's own
	// {CHECKOUT_SESSION_ID} placeholder is preserved verbatim for Stripe.
	if !strings.Contains(decoded, "success_url=https://ryze.example/en/programs/prog-urls/purchase/success?session_id={CHECKOUT_SESSION_ID}&purchase_id=purchase-urls") {
		t.Errorf("expected substituted success_url, body: %s", capturedBody)
	}
	if !strings.Contains(decoded, "cancel_url=https://ryze.example/en/programs/prog-urls/purchase?canceled=1&purchase_id=purchase-urls") {
		t.Errorf("expected substituted cancel_url, body: %s", capturedBody)
	}
	if strings.Contains(capturedBody, "{program_id}") || strings.Contains(capturedBody, "{purchase_id}") {
		t.Errorf("expected no unsubstituted RYZE placeholders, body: %s", capturedBody)
	}
}
