package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/services/payments"
)

func newMethodsTestRouter(methodMap *payments.MethodProviderMap) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/payments/methods", auth.NewPaymentMethodsHandler(methodMap).List)
	return router
}

func performMethodsRequest(router *gin.Engine) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/methods", nil)
	router.ServeHTTP(rec, req)
	return rec
}

type methodsResponseBody struct {
	Success bool                `json:"success"`
	Data    []map[string]string `json:"data"`
}

func fetchAdvertisedMethods(t *testing.T, methodMap *payments.MethodProviderMap) []map[string]string {
	t.Helper()
	rec := performMethodsRequest(newMethodsTestRouter(methodMap))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body methodsResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success {
		t.Fatal("expected a success response")
	}
	return body.Data
}

func advertisedMethodNames(methods []map[string]string) []string {
	names := make([]string, 0, len(methods))
	for _, method := range methods {
		names = append(names, method["method"])
	}
	return names
}

func TestPaymentMethodsEndpointAllConfigured(t *testing.T) {
	// Stripe settles both card and MB WAY, so the real wiring registers the
	// same provider instance in both slots.
	stripe := payments.NewFakeProvider()
	methodMap := payments.NewMethodProviderMap(stripe, stripe, payments.NewFakeProvider())

	data := fetchAdvertisedMethods(t, methodMap)

	want := []string{"card", "mbway", "paypal"}
	got := advertisedMethodNames(data)
	if len(got) != len(want) {
		t.Fatalf("expected %d methods, got %v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected method %q at position %d, got %v", want[i], i, got)
		}
	}
	for _, method := range data {
		if method["label"] == "" {
			t.Fatalf("method %q must carry a display label", method["method"])
		}
	}
}

func TestPaymentMethodsEndpointMBWayLabel(t *testing.T) {
	stripe := payments.NewFakeProvider()
	data := fetchAdvertisedMethods(t, payments.NewMethodProviderMap(stripe, stripe, nil))

	for _, method := range data {
		if method["method"] == "mbway" && method["label"] != "MB WAY" {
			t.Fatalf("expected the MB WAY label, got %q", method["label"])
		}
	}
}

func TestPaymentMethodsEndpointAdvertisesEachMethodOnlyWithItsProvider(t *testing.T) {
	// Every method is advertised exactly when its own slot holds a provider, so
	// the buyer only ever sees a method RYZE can actually settle.
	cases := map[string]struct {
		methodMap *payments.MethodProviderMap
		want      []string
	}{
		"nothing": {
			methodMap: payments.NewMethodProviderMap(nil, nil, nil),
		},
		"card only": {
			methodMap: payments.NewMethodProviderMap(payments.NewFakeProvider(), nil, nil),
			want:      []string{"card"},
		},
		"mbway only": {
			methodMap: payments.NewMethodProviderMap(nil, payments.NewFakeProvider(), nil),
			want:      []string{"mbway"},
		},
		"paypal only": {
			methodMap: payments.NewMethodProviderMap(nil, nil, payments.NewFakeProvider()),
			want:      []string{"paypal"},
		},
		"card and paypal": {
			methodMap: payments.NewMethodProviderMap(payments.NewFakeProvider(), nil, payments.NewFakeProvider()),
			want:      []string{"card", "paypal"},
		},
		"mbway and paypal": {
			methodMap: payments.NewMethodProviderMap(nil, payments.NewFakeProvider(), payments.NewFakeProvider()),
			want:      []string{"mbway", "paypal"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := advertisedMethodNames(fetchAdvertisedMethods(t, tc.methodMap))
			if len(got) != len(tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("expected method %q at position %d, got %v", tc.want[i], i, got)
				}
			}
		})
	}
}

// TestPaymentMethodsEndpointMBWayFollowsCard proves MB WAY shares the Stripe
// provider: the real wiring registers one instance in both slots, so MB WAY
// appears and disappears together with card.
func TestPaymentMethodsEndpointMBWayFollowsCard(t *testing.T) {
	stripe := payments.NewFakeProvider()
	data := fetchAdvertisedMethods(t, payments.NewMethodProviderMap(stripe, stripe, nil))

	got := advertisedMethodNames(data)
	if len(got) != 2 || got[0] != "card" || got[1] != "mbway" {
		t.Fatalf("expected card followed by mbway, got %v", got)
	}
}

// TestPaymentMethodsEndpointNeverExposesProviderInternals verifies the public
// contract: the endpoint returns only the method identifier and its label, so no
// provider name, credential or endpoint can leak. The method identifiers and
// labels themselves are part of the public contract and are not leaks.
func TestPaymentMethodsEndpointNeverExposesProviderInternals(t *testing.T) {
	stripe := payments.NewFakeProvider()
	data := fetchAdvertisedMethods(t, payments.NewMethodProviderMap(stripe, stripe, payments.NewFakeProvider()))

	forbidden := []string{"stripe", "sk_", "whsec_", "api.stripe.com", "checkout.stripe.com", "secret", "credential"}
	for _, method := range data {
		if len(method) != 2 {
			t.Errorf("expected only method and label, got %v", method)
			continue
		}
		if method["method"] == "" || method["label"] == "" {
			t.Errorf("expected a method identifier and a label, got %v", method)
		}
		for key, value := range method {
			if containsFold(value, forbidden...) {
				t.Errorf("field %q leaks provider internals: %q", key, value)
			}
		}
	}
}

func containsFold(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(strings.ToLower(value), strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func TestPaymentMethodsEndpointOnlyPayPal(t *testing.T) {
	router := newMethodsTestRouter(payments.NewMethodProviderMap(nil, nil, payments.NewFakeProvider()))
	rec := performMethodsRequest(router)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body methodsResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0]["method"] != "paypal" {
		t.Fatalf("expected only paypal, got %v", body.Data)
	}
}

func TestPaymentMethodsEndpointNoneConfigured(t *testing.T) {
	router := newMethodsTestRouter(payments.NewMethodProviderMap(nil, nil, nil))
	rec := performMethodsRequest(router)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body methodsResponseBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 0 {
		t.Fatalf("expected no advertised methods, got %v", body.Data)
	}
}
