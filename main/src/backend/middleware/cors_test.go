package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware"
)

const testOrigin = "http://localhost:5173"

func corsRouter(t *testing.T, allowedOrigins []string) *gin.Engine {
	t.Helper()

	router := gin.New()
	router.Use(middleware.CORS(allowedOrigins))
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return router
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	router := corsRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", testOrigin)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != testOrigin {
		t.Fatalf("expected Access-Control-Allow-Origin %q, got %q", testOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials true, got %q", rec.Header().Get("Access-Control-Allow-Credentials"))
	}
	if rec.Header().Get("Vary") != "Origin" {
		t.Fatalf("expected Vary Origin, got %q", rec.Header().Get("Vary"))
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	router := corsRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSHandlesPreflight(t *testing.T) {
	router := corsRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", testOrigin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != testOrigin {
		t.Fatalf("expected Access-Control-Allow-Origin %q, got %q", testOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected Access-Control-Allow-Methods to be set")
	}
	if rec.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Fatal("expected Access-Control-Allow-Headers to be set")
	}
}

func TestCORSPreflightRejectsUnknownOrigin(t *testing.T) {
	router := corsRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSRequestWithoutOriginIgnores(t *testing.T) {
	router := corsRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}
