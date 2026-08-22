package payments_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ryze/backend/services/payments"
)

func TestPayPalProvider_Success(t *testing.T) {
	orderID := "5O190127TN364715T"
	approvalURL := "https://www.sandbox.paypal.com/checkoutnow?token=5O190127TN364715T"

	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v2/checkout/orders" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("expected Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"id":     orderID,
			"status": "CREATED",
			"links": []map[string]interface{}{
				{"href": approvalURL, "rel": "approve", "method": "GET"},
				{"href": "https://api-m.sandbox.paypal.com/v2/checkout/orders/" + orderID, "rel": "self", "method": "GET"},
			},
		}
		b, _ := json.Marshal(resp)
		fmt.Fprint(w, string(b))
	})
	defer server.Close()

	result, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-123",
		AmountMinorUnits: 4999,
		Currency:         "EUR",
		ProgramID:        "prog-abc",
		Method:           payments.PaymentMethodPayPal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentID != orderID {
		t.Errorf("expected PaymentID %q, got %q", orderID, result.PaymentID)
	}
	if result.CheckoutURL != approvalURL {
		t.Errorf("expected CheckoutURL %q, got %q", approvalURL, result.CheckoutURL)
	}
	if result.Provider != "paypal" {
		t.Errorf("expected Provider %q, got %q", "paypal", result.Provider)
	}
	if result.PurchaseID != "purchase-123" {
		t.Errorf("expected PurchaseID %q, got %q", "purchase-123", result.PurchaseID)
	}
	if result.Status != payments.PaymentStatusRequiresAction {
		t.Errorf("expected Status %q, got %q", payments.PaymentStatusRequiresAction, result.Status)
	}
}

func TestPayPalProvider_EmptyPurchaseID(t *testing.T) {
	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach PayPal API")
	})
	defer server.Close()

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		AmountMinorUnits: 100,
		Currency:         "EUR",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodPayPal,
	})
	if err == nil {
		t.Fatal("expected error for empty purchase ID")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

func TestPayPalProvider_ZeroAmount(t *testing.T) {
	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach PayPal API")
	})
	defer server.Close()

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-1",
		AmountMinorUnits: 0,
		Currency:         "EUR",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodPayPal,
	})
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

func TestPayPalProvider_EmptyCurrency(t *testing.T) {
	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach PayPal API")
	})
	defer server.Close()

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-1",
		AmountMinorUnits: 100,
		Currency:         "",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodPayPal,
	})
	if err == nil {
		t.Fatal("expected error for empty currency")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

func TestPayPalProvider_IdempotencyKey(t *testing.T) {
	orderID := "5O190127TN364715T"
	approvalURL := "https://www.sandbox.paypal.com/checkoutnow?token=5O190127TN364715T"
	purchaseID := "purchase-idem-42"

	var capturedRequestID string

	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Header.Get("PayPal-Request-Id")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"id":     orderID,
			"status": "CREATED",
			"links": []map[string]interface{}{
				{"href": approvalURL, "rel": "approve", "method": "GET"},
			},
		}
		b, _ := json.Marshal(resp)
		fmt.Fprint(w, string(b))
	})
	defer server.Close()

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       purchaseID,
		AmountMinorUnits: 2999,
		Currency:         "EUR",
		ProgramID:        "prog-idem",
		Method:           payments.PaymentMethodPayPal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedKey := fmt.Sprintf("ryze-purchase-%s", purchaseID)
	if capturedRequestID != expectedKey {
		t.Errorf("expected idempotency key %q, got %q", expectedKey, capturedRequestID)
	}
}

func TestPayPalProvider_AmountConversion(t *testing.T) {
	tests := []struct {
		minorUnits int64
		expected   string
	}{
		{100, "1.00"},
		{4999, "49.99"},
		{10, "0.10"},
		{1, "0.01"},
		{10000, "100.00"},
		{10050, "100.50"},
		{10005, "100.05"},
	}

	for _, tt := range tests {
		var capturedValue string

		server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			units := body["purchase_units"].([]interface{})
			amount := units[0].(map[string]interface{})["amount"].(map[string]interface{})
			capturedValue = amount["value"].(string)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			resp := map[string]interface{}{
				"id":     "test-order",
				"status": "CREATED",
				"links":  []map[string]interface{}{},
			}
			b, _ := json.Marshal(resp)
			fmt.Fprint(w, string(b))
		})

		_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
			PurchaseID:       "test-purchase",
			AmountMinorUnits: tt.minorUnits,
			Currency:         "EUR",
			ProgramID:        "test-program",
			Method:           payments.PaymentMethodPayPal,
		})
		if err != nil {
			t.Fatalf("unexpected error for amount %d: %v", tt.minorUnits, err)
		}
		if capturedValue != tt.expected {
			t.Errorf("expected amount value %q, got %q", tt.expected, capturedValue)
		}

		server.Close()
	}
}

func TestPayPalProvider_APIError(t *testing.T) {
	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		resp := map[string]interface{}{
			"name":    "UNPROCESSABLE_ENTITY",
			"message": "The requested action could not be performed.",
		}
		b, _ := json.Marshal(resp)
		fmt.Fprint(w, string(b))
	})
	defer server.Close()

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-1",
		AmountMinorUnits: 100,
		Currency:         "EUR",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodPayPal,
	})
	if err == nil {
		t.Fatal("expected error for PayPal API failure")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

func TestPayPalProvider_NoApprovalURL(t *testing.T) {
	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"id":     "test-order-nolink",
			"status": "CREATED",
			"links":  []map[string]interface{}{},
		}
		b, _ := json.Marshal(resp)
		fmt.Fprint(w, string(b))
	})
	defer server.Close()

	result, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-1",
		AmountMinorUnits: 100,
		Currency:         "EUR",
		ProgramID:        "prog-1",
		Method:           payments.PaymentMethodPayPal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CheckoutURL != "" {
		t.Errorf("expected empty CheckoutURL when no approve link, got %q", result.CheckoutURL)
	}
}

func TestPayPalProvider_CurrencyUppercased(t *testing.T) {
	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		units := body["purchase_units"].([]interface{})
		amount := units[0].(map[string]interface{})["amount"].(map[string]interface{})
		currency := amount["currency_code"].(string)
		if currency != "EUR" {
			t.Errorf("expected uppercase currency EUR, got %s", currency)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"id":     "test-order",
			"status": "CREATED",
			"links":  []map[string]interface{}{},
		}
		b, _ := json.Marshal(resp)
		fmt.Fprint(w, string(b))
	})
	defer server.Close()

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "test-purchase",
		AmountMinorUnits: 100,
		Currency:         "eur",
		ProgramID:        "test-program",
		Method:           payments.PaymentMethodPayPal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPayPalProvider_VerifyRequestPayload(t *testing.T) {
	var capturedBody map[string]interface{}

	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"id":     "test-order",
			"status": "CREATED",
			"links":  []map[string]interface{}{},
		}
		b, _ := json.Marshal(resp)
		fmt.Fprint(w, string(b))
	})
	defer server.Close()

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-verify",
		AmountMinorUnits: 9999,
		Currency:         "EUR",
		ProgramID:        "prog-verify",
		Method:           payments.PaymentMethodPayPal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBody == nil {
		t.Fatal("expected request body")
	}
	if intent, _ := capturedBody["intent"].(string); intent != "CAPTURE" {
		t.Errorf("expected intent CAPTURE, got %s", intent)
	}
	units := capturedBody["purchase_units"].([]interface{})
	if len(units) != 1 {
		t.Fatalf("expected 1 purchase unit, got %d", len(units))
	}
	unit := units[0].(map[string]interface{})
	if ref, _ := unit["reference_id"].(string); ref != "purchase-verify" {
		t.Errorf("expected reference_id purchase-verify, got %s", ref)
	}
	amount := unit["amount"].(map[string]interface{})
	if val, _ := amount["value"].(string); val != "99.99" {
		t.Errorf("expected amount value 99.99, got %s", val)
	}
	if cur, _ := amount["currency_code"].(string); cur != "EUR" {
		t.Errorf("expected currency EUR, got %s", cur)
	}
}

// --- NewPayPalProvider configuration tests ---

func TestNewPayPalProvider_EmptyClientID(t *testing.T) {
	_, err := payments.NewPayPalProvider("", "secret", "sandbox")
	if err == nil {
		t.Fatal("expected error for empty client ID")
	}
}

func TestNewPayPalProvider_EmptySecret(t *testing.T) {
	_, err := payments.NewPayPalProvider("client-id", "", "sandbox")
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestNewPayPalProvider_InvalidMode(t *testing.T) {
	_, err := payments.NewPayPalProvider("client-id", "secret", "production")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestNewPayPalProvider_LiveMode(t *testing.T) {
	provider, err := payments.NewPayPalProvider("client-id", "secret", "live")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestPayPalProvider_NegativeAmount(t *testing.T) {
	server, provider := setupPayPalTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach PayPal API")
	})
	defer server.Close()

	_, err := provider.InitiatePayment(context.Background(), payments.PaymentRequest{
		PurchaseID:       "purchase-neg",
		AmountMinorUnits: -100,
		Currency:         "EUR",
		ProgramID:        "prog-neg",
		Method:           payments.PaymentMethodPayPal,
	})
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
	if !errors.Is(err, payments.ErrProviderFailure) {
		t.Errorf("expected ErrProviderFailure, got: %v", err)
	}
}

// --- helper ---

func setupPayPalTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *payments.PayPalProvider) {
	t.Helper()

	// Wrap the handler to also handle the OAuth2 token endpoint that the
	// PayPal SDK calls before every authenticated request.
	wrappedHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth2/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := map[string]interface{}{
				"scope":        "",
				"access_token": "test-access-token",
				"token_type":   "Bearer",
				"app_id":       "APP-80312839XS434310T",
				"expires_in":   32400,
				"nonce":        "test-nonce",
			}
			b, _ := json.Marshal(resp)
			fmt.Fprint(w, string(b))
			return
		}
		handler(w, r)
	}

	server := httptest.NewServer(http.HandlerFunc(wrappedHandler))

	provider, err := payments.NewPayPalProvider("test-client-id", "test-secret", "sandbox")
	if err != nil {
		t.Fatalf("failed to create PayPal provider: %v", err)
	}

	// Override the PayPal SDK's HTTP client to route all requests to our
	// test server. We create a transport that rewrites every request URL
	// to point at the test server.
	transport := &roundTripperFunc{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		},
	}
	provider.SetHTTPClient(&http.Client{Transport: transport})

	return server, provider
}

// roundTripperFunc adapts a function to the http.RoundTripper interface.
type roundTripperFunc struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (f *roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f.roundTrip(req)
}

// Ensure imports used
var _ = strings.TrimSpace
