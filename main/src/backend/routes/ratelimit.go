package routes

import (
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
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
// either from the JSON body field accountField or from the credential cookie
// accountCookie, so credential stuffing against a single account is slowed
// even from many addresses.
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
	accountCookie string
}

var authRateLimitScopes = []authRateLimitScope{
	{name: scopeAuthRegister, ipLimit: 10, ipWindow: time.Hour, accountLimit: 10, accountWindow: time.Hour, accountField: "email"},
	{name: scopeAuthLogin, ipLimit: 20, ipWindow: 15 * time.Minute, accountLimit: 10, accountWindow: 15 * time.Minute, accountField: "email"},
	{name: scopeAdminLogin, ipLimit: 10, ipWindow: 15 * time.Minute, accountLimit: 10, accountWindow: 15 * time.Minute, accountField: "username"},
	// The verify stage has no body identifier; the stage cookie identifies the
	// pending login, so guessing access codes is limited per pending login as
	// well as per address.
	{name: scopeAdminVerify, ipLimit: 10, ipWindow: 15 * time.Minute, accountLimit: 10, accountWindow: 15 * time.Minute, accountCookie: auth.AdminStageTokenCookieName},
	{name: scopeAuthChangePassword, ipLimit: 20, ipWindow: 15 * time.Minute},
	{name: scopeAuthDeleteAccount, ipLimit: 10, ipWindow: time.Hour},
}

// authRateLimiters owns the limiters for every authentication scope.
type authRateLimiters struct {
	enabled      bool
	ip           map[string]*middleware.Limiter
	account      map[string]*middleware.Limiter
	accountField map[string]string
	// accountCookie names the cookie that keys a scope's account limiter when
	// the scope is cookie-keyed instead of body-keyed.
	accountCookie map[string]string
	// accountBodyBytes is the configured request body bound, shared with the
	// body-limit middleware so the identifier read can never be bypassed with
	// a body larger than the default.
	accountBodyBytes int64
}

// newAuthRateLimiters builds one IP limiter per scope and, where configured, one
// account limiter per scope. Every limiter shares the same key bound from the
// configuration.
func newAuthRateLimiters(cfg config.RateLimitConfig, maxBodyBytes int64) *authRateLimiters {
	limiters := &authRateLimiters{
		enabled:          cfg.Enabled,
		ip:               make(map[string]*middleware.Limiter, len(authRateLimitScopes)),
		account:          make(map[string]*middleware.Limiter),
		accountField:     make(map[string]string),
		accountCookie:    make(map[string]string),
		accountBodyBytes: maxBodyBytes,
	}

	for _, scope := range authRateLimitScopes {
		limiters.ip[scope.name] = middleware.NewLimiter(
			middleware.RateLimitPolicy{Limit: scope.ipLimit, Window: scope.ipWindow},
			cfg.MaxKeys,
		)
		if scope.accountLimit > 0 {
			limiters.account[scope.name] = middleware.NewLimiter(
				middleware.RateLimitPolicy{Limit: scope.accountLimit, Window: scope.accountWindow},
				cfg.MaxKeys,
			)
			limiters.accountField[scope.name] = scope.accountField
			limiters.accountCookie[scope.name] = scope.accountCookie
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
		switch {
		case l.accountField[scope] != "":
			handlers = append(handlers, middleware.RateLimit(limiter, middleware.AccountRateLimitKey(scope, l.accountField[scope], l.accountBodyBytes)))
		case l.accountCookie[scope] != "":
			handlers = append(handlers, middleware.RateLimit(limiter, middleware.CookieRateLimitKey(scope, l.accountCookie[scope])))
		}
	}
	return handlers
}
