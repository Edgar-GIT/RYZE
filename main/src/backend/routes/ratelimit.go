package routes

import (
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/config"
	"ryze/backend/middleware"
)

// Rate-limit scopes. Each authentication endpoint limits independently so a
// flood against one endpoint never locks a user out of another.
const (
	scopeAuthRegister       = "auth-register"
	scopeAuthLogin          = "auth-login"
	scopeAdminLogin         = "admin-login"
	scopeAdminVerify        = "admin-verify"
	scopeAuthChangePassword = "auth-change-password"
	scopeAuthDeleteAccount  = "auth-delete-account"
)

// authRateLimitScope is one endpoint's policy. The IP limiter always applies.
// accountLimit > 0 additionally limits by a normalised, hashed identifier read
// from the JSON body field accountField, so credential stuffing against a
// single account is slowed even from many addresses.
//
// The windows are deliberately generous: ordinary users, including someone who
// mistypes a password a few times, never reach a limit. The IP limits are looser
// than the account limits because a shared egress address (an office, a mobile
// carrier) legitimately produces many requests.
type authRateLimitScope struct {
	name          string
	ipLimit       int
	ipWindow      time.Duration
	accountLimit  int
	accountWindow time.Duration
	accountField  string
}

var authRateLimitScopes = []authRateLimitScope{
	{name: scopeAuthRegister, ipLimit: 10, ipWindow: time.Hour, accountLimit: 10, accountWindow: time.Hour, accountField: "email"},
	{name: scopeAuthLogin, ipLimit: 20, ipWindow: 15 * time.Minute, accountLimit: 10, accountWindow: 15 * time.Minute, accountField: "email"},
	{name: scopeAdminLogin, ipLimit: 10, ipWindow: 15 * time.Minute, accountLimit: 10, accountWindow: 15 * time.Minute, accountField: "username"},
	{name: scopeAdminVerify, ipLimit: 10, ipWindow: 15 * time.Minute},
	{name: scopeAuthChangePassword, ipLimit: 20, ipWindow: 15 * time.Minute},
	{name: scopeAuthDeleteAccount, ipLimit: 10, ipWindow: time.Hour},
}

// authRateLimiters owns the limiters for every authentication scope.
type authRateLimiters struct {
	enabled      bool
	ip           map[string]*middleware.Limiter
	account      map[string]*middleware.Limiter
	accountField map[string]string
}

// newAuthRateLimiters builds one IP limiter per scope and, where configured, one
// account limiter per scope. Every limiter shares the same key bound from the
// configuration.
func newAuthRateLimiters(cfg config.RateLimitConfig) *authRateLimiters {
	limiters := &authRateLimiters{
		enabled:      cfg.Enabled,
		ip:           make(map[string]*middleware.Limiter, len(authRateLimitScopes)),
		account:      make(map[string]*middleware.Limiter),
		accountField: make(map[string]string),
	}

	for _, scope := range authRateLimitScopes {
		limiters.ip[scope.name] = middleware.NewLimiter(
			middleware.RateLimitPolicy{Limit: scope.ipLimit, Window: scope.ipWindow},
			cfg.MaxKeys,
		)
		if scope.accountLimit > 0 && scope.accountField != "" {
			limiters.account[scope.name] = middleware.NewLimiter(
				middleware.RateLimitPolicy{Limit: scope.accountLimit, Window: scope.accountWindow},
				cfg.MaxKeys,
			)
			limiters.accountField[scope.name] = scope.accountField
		}
	}

	return limiters
}

// chain returns the rate-limiting middleware for a scope, ordered so the IP
// limiter runs before the account limiter. It is empty when the subsystem is
// disabled, so toggling it never changes the route table.
func (l *authRateLimiters) chain(scope string) []gin.HandlerFunc {
	if !l.enabled {
		return nil
	}

	ipLimiter, ok := l.ip[scope]
	if !ok {
		return nil
	}

	handlers := []gin.HandlerFunc{
		middleware.RateLimit(ipLimiter, middleware.IPRateLimitKey(scope)),
	}
	if limiter, ok := l.account[scope]; ok {
		handlers = append(handlers, middleware.RateLimit(limiter, middleware.AccountRateLimitKey(scope, l.accountField[scope])))
	}
	return handlers
}
