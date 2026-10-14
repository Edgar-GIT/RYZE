package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/version"
)

// fakePinger is a controllable DatabasePinger for the health endpoint.
type fakePinger struct {
	err error
}

func (f fakePinger) PingContext(context.Context) error { return f.err }

func newHealthRouter(db auth.DatabasePinger) *gin.Engine {
	handler := auth.NewHealthHandler(db)

	router := gin.New()
	router.GET("/api/v1/health", handler.Status)
	router.GET("/api/v1/version", handler.Version)
	return router
}

func TestHealthStatusHealthy(t *testing.T) {
	router := newHealthRouter(fakePinger{})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected status ok, got %s", rec.Body.String())
	}
}

func TestHealthStatusWithoutDatabase(t *testing.T) {
	router := newHealthRouter(nil)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 without a database handle, got %d", rec.Code)
	}
}

func TestHealthStatusDatabaseUnreachable(t *testing.T) {
	router := newHealthRouter(fakePinger{err: errors.New("connection refused")})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"SERVICE_UNAVAILABLE"`) {
		t.Fatalf("expected SERVICE_UNAVAILABLE, got %s", rec.Body.String())
	}
}

func TestHealthVersionReportsBuildVersion(t *testing.T) {
	original := version.Version
	version.Version = "9.9.9-test"
	t.Cleanup(func() { version.Version = original })

	router := newHealthRouter(nil)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/version", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode version response: %v", err)
	}
	if !payload.Success || payload.Data.Version != "9.9.9-test" {
		t.Fatalf("expected version 9.9.9-test, got %s", rec.Body.String())
	}
}
