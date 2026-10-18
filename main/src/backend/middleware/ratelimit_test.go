package middleware_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware"
)

// fixedClock returns a clock function backed by the given pointer, so a test
// can move time forward without sleeping.
func fixedClock(now *time.Time) func() time.Time {
	return func() time.Time { return *now }
}

func newTestLimiter(limit int, window time.Duration, now *time.Time, maxKeys int) *middleware.Limiter {
	return middleware.NewLimiter(
		middleware.RateLimitPolicy{Limit: limit, Window: window},
		maxKeys,
		middleware.WithLimiterClock(fixedClock(now)),
	)
}

func TestLimiterAllowsWithinLimitAndThenBlocks(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(3, time.Minute, &now, 100)

	for i := 0; i < 3; i++ {
		if allowed, _ := limiter.Allow("client"); !allowed {
			t.Fatalf("request %d within the limit must be allowed", i+1)
		}
	}

	allowed, retryAfter := limiter.Allow("client")
	if allowed {
		t.Fatal("the request past the limit must be blocked")
	}
	if retryAfter <= 0 || retryAfter > time.Minute {
		t.Fatalf("retry after must be within the window, got %s", retryAfter)
	}
}

func TestLimiterResetsAfterWindow(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(1, time.Minute, &now, 100)

	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("the first request must be allowed")
	}
	if allowed, _ := limiter.Allow("client"); allowed {
		t.Fatal("the second request within the window must be blocked")
	}

	now = now.Add(time.Minute)
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("a request after the window must be allowed again")
	}
}

func TestLimiterTracksKeysIndependently(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(1, time.Minute, &now, 100)

	if allowed, _ := limiter.Allow("a"); !allowed {
		t.Fatal("first key must be allowed")
	}
	if allowed, _ := limiter.Allow("b"); !allowed {
		t.Fatal("a different key must not be affected by another key's state")
	}
}

func TestLimiterWithNonPositiveLimitAllowsEverything(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(0, time.Minute, &now, 100)

	for i := 0; i < 100; i++ {
		if allowed, retryAfter := limiter.Allow("client"); !allowed || retryAfter != 0 {
			t.Fatalf("a disabled limiter must allow everything, got allowed=%v retry=%s", allowed, retryAfter)
		}
	}
}

func TestLimiterBoundsTrackedKeysWithLRU(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(1, time.Hour, &now, 2)

	if allowed, _ := limiter.Allow("a"); !allowed {
		t.Fatal("a must be allowed")
	}
	if allowed, _ := limiter.Allow("b"); !allowed {
		t.Fatal("b must be allowed")
	}
	if allowed, _ := limiter.Allow("c"); !allowed {
		t.Fatal("c must be allowed and evict the least recently used key")
	}

	// "a" was the least recently used and was evicted, so it starts fresh; if it
	// had been retained it would still be inside its window and be blocked.
	if allowed, _ := limiter.Allow("a"); !allowed {
		t.Fatal("an evicted key must be forgotten and allowed again")
	}
}

// rateLimitRouter builds a router whose endpoint echoes the body it received,
// so tests can assert the account key never consumes or alters the body.
func rateLimitRouter(limiter *middleware.Limiter, keyFn func(*gin.Context) string) *gin.Engine {
	router := gin.New()
	router.POST("/login", middleware.RateLimit(limiter, keyFn), func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "read failed: %v", err)
			return
		}
		c.String(http.StatusOK, string(body))
	})
	return router
}

func TestRateLimitReturns429WithRetryAfter(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(1, time.Minute, &now, 100)
	router := rateLimitRouter(limiter, middleware.IPRateLimitKey("test"))

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/login", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request must succeed, got %d", first.Code)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/login", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("a 429 must carry a Retry-After header")
	}
	seconds, err := strconv.Atoi(second.Header().Get("Retry-After"))
	if err != nil || seconds < 1 || seconds > 60 {
		t.Fatalf("Retry-After must be a whole number of seconds, got %q", second.Header().Get("Retry-After"))
	}
}

func TestAccountRateLimitSharesCounterAcrossNormalisedValues(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(1, time.Minute, &now, 100)
	router := rateLimitRouter(limiter, middleware.AccountRateLimitKey("test", "email", 1<<20))

	first := httptest.NewRecorder()
	router.ServeHTTP(first, newJSONRequest(t, `{"email":"Ada@Example.com","password":"x"}`))
	if first.Code != http.StatusOK {
		t.Fatalf("first request must succeed, got %d", first.Code)
	}

	// Same address, different case and surrounding whitespace, must hit the
	// same counter.
	second := httptest.NewRecorder()
	router.ServeHTTP(second, newJSONRequest(t, `{"email":"  ada@example.com  ","password":"x"}`))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("a normalised duplicate must be limited, got %d", second.Code)
	}
}

func TestAccountRateLimitRestoresBodyForHandler(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(5, time.Minute, &now, 100)
	router := rateLimitRouter(limiter, middleware.AccountRateLimitKey("test", "email", 1<<20))

	payload := `{"email":"ada@example.com","password":"secret"}`
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, newJSONRequest(t, payload))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != payload {
		t.Fatalf("the handler must observe the unchanged body, got %q", rec.Body.String())
	}
}

func TestAccountRateLimitPassesThroughWithoutIdentifier(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(1, time.Minute, &now, 100)
	router := rateLimitRouter(limiter, middleware.AccountRateLimitKey("test", "email", 1<<20))

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, newJSONRequest(t, `{"password":"x"}`))
		if rec.Code != http.StatusOK {
			t.Fatalf("a request without the identifier must fall back to the IP limiter, got %d", rec.Code)
		}
	}
}

func TestAccountRateLimitReadsIdentifierWithinConfiguredBound(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(1, time.Minute, &now, 100)
	router := rateLimitRouter(limiter, middleware.AccountRateLimitKey("test", "email", 64))

	first := httptest.NewRecorder()
	router.ServeHTTP(first, newJSONRequest(t, `{"email":"ada@example.com"}`))
	if first.Code != http.StatusOK {
		t.Fatalf("first request must succeed, got %d", first.Code)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, newJSONRequest(t, `{"email":"ada@example.com"}`))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("the identifier must be read within the configured bound, got %d", second.Code)
	}
}

func TestAccountRateLimitSkipsBodiesLargerThanConfiguredBound(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(1, time.Minute, &now, 100)
	router := rateLimitRouter(limiter, middleware.AccountRateLimitKey("test", "email", 8))

	// A body larger than the configured bound yields no identifier, and the
	// handler must still observe every byte of the original body.
	payload := `{"email":"ada@example.com"}`
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, newJSONRequest(t, payload))
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d must pass through, got %d", i+1, rec.Code)
		}
		if rec.Body.String() != payload {
			t.Fatalf("the handler must observe the full body, got %q", rec.Body.String())
		}
	}
}

func TestCookieRateLimitKeysByCookieValue(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(1, time.Minute, &now, 100)
	router := rateLimitRouter(limiter, middleware.CookieRateLimitKey("test", "stage"))

	request := func(cookie string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "stage", Value: cookie})
		}
		return req
	}

	first := httptest.NewRecorder()
	router.ServeHTTP(first, request("stage-token-a"))
	if first.Code != http.StatusOK {
		t.Fatalf("first request must succeed, got %d", first.Code)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, request("stage-token-a"))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("the same cookie value must share a counter, got %d", second.Code)
	}

	third := httptest.NewRecorder()
	router.ServeHTTP(third, request("stage-token-b"))
	if third.Code != http.StatusOK {
		t.Fatalf("a different cookie value must have its own counter, got %d", third.Code)
	}

	fourth := httptest.NewRecorder()
	router.ServeHTTP(fourth, request(""))
	if fourth.Code != http.StatusOK {
		t.Fatalf("a request without the cookie must pass through, got %d", fourth.Code)
	}
}

func newJSONRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	return req
}
