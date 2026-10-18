package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware"
)

func originGuardRouter(t *testing.T, allowedOrigins []string) *gin.Engine {
	t.Helper()

	router := gin.New()
	router.Use(middleware.OriginGuard(allowedOrigins))
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/api/v1/programs", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return router
}

func TestOriginGuardAllowsConfiguredOrigin(t *testing.T) {
	router := originGuardRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", testOrigin)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for an allow-listed origin, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestOriginGuardAllowsSameOriginRequest(t *testing.T) {
	router := originGuardRouter(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Host = "app.ryze.example"
	req.Header.Set("Origin", "https://app.ryze.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a same-origin request, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestOriginGuardRejectsUnknownOrigin(t *testing.T) {
	router := originGuardRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a foreign origin, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "CROSS_ORIGIN_FORBIDDEN") {
		t.Fatalf("expected CROSS_ORIGIN_FORBIDDEN, got %s", rec.Body.String())
	}
}

func TestOriginGuardRejectsSameSiteSiblingHost(t *testing.T) {
	router := originGuardRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Host = "api.ryze.example"
	req.Header.Set("Origin", "https://evil.ryze.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a sibling host outside the allow list, got %d", rec.Code)
	}
}

func TestOriginGuardRejectsOpaqueOrigin(t *testing.T) {
	router := originGuardRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "null")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for an opaque origin, got %d", rec.Code)
	}
}

func TestOriginGuardRejectsMalformedOrigin(t *testing.T) {
	router := originGuardRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "not a url")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a malformed origin, got %d", rec.Code)
	}
}

func TestOriginGuardAllowsRequestsWithoutBrowserOrigin(t *testing.T) {
	router := originGuardRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a request without origin headers, got %d", rec.Code)
	}
}

func TestOriginGuardFallsBackToReferer(t *testing.T) {
	router := originGuardRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Referer", testOrigin+"/login")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a request with an allow-listed referer, got %d", rec.Code)
	}
}

func TestOriginGuardRejectsForeignReferer(t *testing.T) {
	router := originGuardRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Referer", "https://evil.example.com/steal")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a foreign referer, got %d", rec.Code)
	}
}

func TestOriginGuardIgnoresSafeMethods(t *testing.T) {
	router := originGuardRouter(t, []string{testOrigin})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a safe method even with a foreign origin, got %d", rec.Code)
	}
}

func TestOriginGuardNormalizesOriginCasing(t *testing.T) {
	router := originGuardRouter(t, []string{"https://App.Ryze.Example"})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "https://app.ryze.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a case-mismatched allow-listed origin, got %d", rec.Code)
	}
}
