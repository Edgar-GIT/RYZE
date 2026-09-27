package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestPaymentMethodsEndpointAllConfigured(t *testing.T) {
	router := newMethodsTestRouter(payments.NewMethodProviderMap(payments.NewFakeProvider(), payments.NewFakeProvider()))
	rec := performMethodsRequest(router)

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

	want := []string{"card", "mbway", "paypal"}
	if len(body.Data) != len(want) {
		t.Fatalf("expected %d methods, got %v", len(want), body.Data)
	}
	for i := range want {
		if body.Data[i]["method"] != want[i] {
			t.Fatalf("expected method %q at position %d, got %v", want[i], i, body.Data[i])
		}
		if body.Data[i]["label"] == "" {
			t.Fatalf("method %q must carry a display label", want[i])
		}
	}
}

func TestPaymentMethodsEndpointOnlyPayPal(t *testing.T) {
	router := newMethodsTestRouter(payments.NewMethodProviderMap(nil, payments.NewFakeProvider()))
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
	router := newMethodsTestRouter(payments.NewMethodProviderMap(nil, nil))
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
