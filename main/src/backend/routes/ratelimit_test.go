package routes

import (
	"testing"

	"ryze/backend/api/auth"
	"ryze/backend/config"
)

func TestNewAuthRateLimitersDisabledYieldsNoMiddleware(t *testing.T) {
	limiters := newAuthRateLimiters(config.RateLimitConfig{Enabled: false, MaxKeys: 100}, 1<<20)

	for _, scope := range authRateLimitScopes {
		if handlers := limiters.chain(scope.name); handlers != nil {
			t.Fatalf("scope %q must have no middleware when rate limiting is disabled", scope.name)
		}
	}
}

func TestAuthRateLimitersGrantIPMiddlewareForEveryScope(t *testing.T) {
	limiters := newAuthRateLimiters(config.RateLimitConfig{Enabled: true, MaxKeys: 100}, 1<<20)

	for _, scope := range authRateLimitScopes {
		if len(limiters.chain(scope.name)) < 1 {
			t.Fatalf("scope %q must always have an IP limiter", scope.name)
		}
	}
}

func TestAuthRateLimitersAddAccountMiddlewareOnlyWhenConfigured(t *testing.T) {
	limiters := newAuthRateLimiters(config.RateLimitConfig{Enabled: true, MaxKeys: 100}, 1<<20)

	if got := len(limiters.chain(scopeAuthLogin)); got != 2 {
		t.Fatalf("login must have an IP and an account limiter, got %d middleware", got)
	}
	if got := len(limiters.chain(scopeAuthRegister)); got != 2 {
		t.Fatalf("register must have an IP and an account limiter, got %d middleware", got)
	}
	if got := len(limiters.chain(scopeAuthChangePassword)); got != 1 {
		t.Fatalf("change-password must have only an IP limiter, got %d middleware", got)
	}
	if got := len(limiters.chain(scopeAdminVerify)); got != 2 {
		t.Fatalf("admin verify must have an IP and a stage-cookie limiter, got %d middleware", got)
	}
}

func TestAdminVerifyAccountLimiterIsKeyedByStageCookie(t *testing.T) {
	limiters := newAuthRateLimiters(config.RateLimitConfig{Enabled: true, MaxKeys: 100}, 1<<20)

	if got := limiters.accountCookie[scopeAdminVerify]; got != auth.AdminStageTokenCookieName {
		t.Fatalf("admin verify must be keyed by the stage cookie, got %q", got)
	}
	if got := limiters.accountField[scopeAuthLogin]; got != "email" {
		t.Fatalf("login must stay body-keyed, got %q", got)
	}
}

func TestAuthRateLimitersIgnoreUnknownScope(t *testing.T) {
	limiters := newAuthRateLimiters(config.RateLimitConfig{Enabled: true, MaxKeys: 100}, 1<<20)

	if handlers := limiters.chain("does-not-exist"); handlers != nil {
		t.Fatalf("an unknown scope must not produce middleware, got %d", len(handlers))
	}
}
