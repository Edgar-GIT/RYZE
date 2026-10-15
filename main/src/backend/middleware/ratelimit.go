package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
)

// RateLimitPolicy describes one fixed window: at most Limit requests per key
// per Window. A Limit that is not positive disables the limiter.
type RateLimitPolicy struct {
	Limit  int
	Window time.Duration
}

// limiterEntry is a single tracked key inside a window.
type limiterEntry struct {
	key     string
	count   int
	resetAt time.Time
}

// Limiter enforces one policy per key using a fixed window. Counters are held
// in memory: each application instance enforces the limits independently, which
// is documented in security.MD. Entries are kept in least-recently-used order
// so a flood of distinct keys cannot grow the map without bound; past maxKeys
// the least recently used key is dropped.
//
// The zero usable state (Limit <= 0) allows everything, so a misconfigured
// policy can never lock users out.
type Limiter struct {
	policy  RateLimitPolicy
	maxKeys int
	now     func() time.Time

	mu      sync.Mutex
	entries map[string]*limiterEntry
	order   []*limiterEntry // most recently used first
}

// LimiterOption customises limiter construction.
type LimiterOption func(*Limiter)

// WithLimiterClock replaces the clock the limiter reads time from, so tests can
// advance time deterministically instead of sleeping.
func WithLimiterClock(now func() time.Time) LimiterOption {
	return func(l *Limiter) { l.now = now }
}

// NewLimiter builds a limiter for one policy. maxKeys bounds how many distinct
// keys are tracked at once; a value that is not positive keeps the bound
// unbounded, which only controlled environments should use.
func NewLimiter(policy RateLimitPolicy, maxKeys int, opts ...LimiterOption) *Limiter {
	l := &Limiter{
		policy:  policy,
		maxKeys: maxKeys,
		now:     time.Now,
		entries: make(map[string]*limiterEntry),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Allow reports whether the key may proceed. When it may not, the returned
// duration is the time left until the current window resets, which the caller
// reports as Retry-After. Counting continues while the window is open, and a
// window that has already elapsed starts a fresh one.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	if l.policy.Limit <= 0 {
		return true, 0
	}

	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if entry, ok := l.entries[key]; ok {
		if now.Before(entry.resetAt) {
			entry.count++
			l.promote(entry)
			if entry.count > l.policy.Limit {
				return false, entry.resetAt.Sub(now)
			}
			return true, 0
		}
		l.remove(entry)
	}

	if l.maxKeys > 0 && len(l.entries) >= l.maxKeys {
		l.evictOldest()
	}

	l.entries[key] = &limiterEntry{key: key, count: 1, resetAt: now.Add(l.policy.Window)}
	// Most recently used first, so eviction can always take the last element.
	l.order = append([]*limiterEntry{l.entries[key]}, l.order...)
	return true, 0
}

func (l *Limiter) promote(entry *limiterEntry) {
	for i, candidate := range l.order {
		if candidate == entry {
			copy(l.order[1:i+1], l.order[:i])
			l.order[0] = entry
			return
		}
	}
}

func (l *Limiter) remove(entry *limiterEntry) {
	delete(l.entries, entry.key)
	for i, candidate := range l.order {
		if candidate == entry {
			l.order = append(l.order[:i], l.order[i+1:]...)
			return
		}
	}
}

func (l *Limiter) evictOldest() {
	last := len(l.order) - 1
	if last < 0 {
		return
	}
	oldest := l.order[last]
	l.order = l.order[:last]
	delete(l.entries, oldest.key)
}

// identifierKey is the process-local, random key used to derive the account
// part of a rate-limit key. Identifiers are only ever used through this keyed
// digest, so a leaked counter table cannot be turned into a dictionary of
// registered accounts.
var identifierKey = sync.OnceValue(func() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		// crypto/rand failing means the platform is broken; fall back to a
		// fixed key rather than panicking inside a request path.
		return []byte("ryze-rate-limit-fallback-key--do-not-use")
	}
	return key
})

// rateLimitFieldMaxBodyBytes bounds how much of a body the account limiter will
// read to locate the identifier. Bodies already bounded by BodyLimit are read
// in full in practice; a larger body simply yields no identifier and the
// account limiter is skipped for that request.
const rateLimitFieldMaxBodyBytes = 1 << 20

// rateLimitKey builds the composite key for a scope, a key kind and a value.
func rateLimitKey(scope, kind, value string) string {
	return scope + "|" + kind + "|" + value
}

// rateLimitDigest hashes an arbitrary value into a stable, compact key part.
// Identifiers are normalised before hashing so "Ada@Example.com" and
// "ada@example.com" share one counter.
func rateLimitDigest(value string) string {
	mac := hmac.New(sha256.New, identifierKey())
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// IPRateLimitKey keys a request by client address within a scope.
func IPRateLimitKey(scope string) func(*gin.Context) string {
	return func(c *gin.Context) string {
		return rateLimitKey(scope, "ip", c.ClientIP())
	}
}

// AccountRateLimitKey keys a request by the normalised value of a string field
// in the JSON body (for example "email" on login). When the body carries no
// readable value for that field, the key is empty and the caller skips the
// account limiter, leaving the IP limiter in place. The body is restored after
// reading, so the handler observes the original content unchanged.
func AccountRateLimitKey(scope, field string) func(*gin.Context) string {
	return func(c *gin.Context) string {
		value := stringField(c, field)
		if value == "" {
			return ""
		}
		return rateLimitKey(scope, "account", rateLimitDigest(value))
	}
}

// stringField reads a top-level string field from the request body and puts the
// body back exactly as it was found.
func stringField(c *gin.Context, field string) string {
	if c.Request == nil || c.Request.Body == nil {
		return ""
	}

	data, err := io.ReadAll(io.LimitReader(c.Request.Body, rateLimitFieldMaxBodyBytes+1))
	// Re-expose everything read plus whatever remains, so a body larger than the
	// bound is never truncated for the handler.
	restored := io.NopCloser(io.MultiReader(bytes.NewReader(data), c.Request.Body))
	c.Request.Body = restored
	if err != nil || len(data) > rateLimitFieldMaxBodyBytes {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	raw, ok := payload[field]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value))
}

// RateLimit returns middleware that rejects a request that has exhausted the
// limiter for its key with 429 RATE_LIMITED and a Retry-After header naming how
// many seconds remain in the window. When the key is empty the request passes
// through, so a body without the expected identifier falls back to the IP
// limiter alone.
func RateLimit(limiter *Limiter, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFn(c)
		if key == "" {
			c.Next()
			return
		}

		allowed, retryAfter := limiter.Allow(key)
		if allowed {
			c.Next()
			return
		}

		respondRateLimited(c, retryAfter)
	}
}

// respondRateLimited writes the shared 429 response. Retry-After is expressed
// in whole seconds, rounded up and never below one, because a zero value would
// tell the client to retry immediately.
func respondRateLimited(c *gin.Context, retryAfter time.Duration) {
	seconds := int(retryAfter / time.Second)
	if retryAfter%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		seconds = 1
	}

	c.Header("Retry-After", strconv.Itoa(seconds))
	auth.RespondError(c, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests. Please try again later.", nil)
	c.Abort()
}
