package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	router := gin.New()
	router.Use(middleware.SecurityHeaders())
	router.GET("/api/v1/me", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	router.GET("/api/v1/programs", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

	private := httptest.NewRecorder()
	router.ServeHTTP(private, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	for name, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
		"Cache-Control":          "no-store",
	} {
		if got := private.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	public := httptest.NewRecorder()
	router.ServeHTTP(public, httptest.NewRequest(http.MethodGet, "/api/v1/programs", nil))
	if public.Header().Get("Cache-Control") != "" {
		t.Error("public catalogue responses must stay cacheable")
	}
	if public.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("every response must disable content sniffing")
	}
}
