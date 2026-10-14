package middleware_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware"
)

const bodyLimit = int64(16)

// bodyLimitRouter builds a router whose single endpoint echoes the request body
// it received, so a test can prove the middleware re-exposes the exact bytes.
func bodyLimitRouter(t *testing.T, maxBytes int64) *gin.Engine {
	t.Helper()

	router := gin.New()
	router.Use(middleware.BodyLimit(maxBytes))
	router.POST("/echo", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "read failed: %v", err)
			return
		}
		c.String(http.StatusOK, string(body))
	})

	return router
}

func TestBodyLimitAllowsBodyWithinLimit(t *testing.T) {
	router := bodyLimitRouter(t, bodyLimit)

	payload := []byte("small body")
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body %q)", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != string(payload) {
		t.Fatalf("handler must receive the unchanged body, got %q", rec.Body.String())
	}
}

func TestBodyLimitAllowsBodyAtLimit(t *testing.T) {
	router := bodyLimitRouter(t, bodyLimit)

	payload := bytes.Repeat([]byte("a"), int(bodyLimit))
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected a body exactly at the limit to be allowed, got %d", rec.Code)
	}
}

func TestBodyLimitRejectsDeclaredOversizedBody(t *testing.T) {
	router := bodyLimitRouter(t, bodyLimit)

	payload := bytes.Repeat([]byte("a"), int(bodyLimit)+1)
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertEntityTooLarge(t, rec)
}

func TestBodyLimitRejectsUndeclaredOversizedBody(t *testing.T) {
	router := bodyLimitRouter(t, bodyLimit)

	payload := bytes.Repeat([]byte("a"), int(bodyLimit)+1)
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(payload))
	// Simulate a chunked request whose length is unknown up front: the body is
	// still bounded, and the rejection is still a deterministic 413.
	req.ContentLength = -1
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertEntityTooLarge(t, rec)
}

func TestBodyLimitDisabledWhenNotPositive(t *testing.T) {
	router := bodyLimitRouter(t, 0)

	payload := bytes.Repeat([]byte("a"), 1024)
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected the limit to be disabled for a non-positive value, got %d", rec.Code)
	}
}

func assertEntityTooLarge(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d (body %q)", rec.Code, rec.Body.String())
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error body must be valid JSON: %v (body %q)", err, rec.Body.String())
	}
	if envelope.Success {
		t.Fatal("expected success=false in the error envelope")
	}
	if envelope.Error.Code != "REQUEST_ENTITY_TOO_LARGE" {
		t.Fatalf("expected REQUEST_ENTITY_TOO_LARGE, got %q", envelope.Error.Code)
	}
}
