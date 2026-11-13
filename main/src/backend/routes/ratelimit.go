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
	scopeContact            = "contact"
	scopeFeedback           = "feedback"
	scopePasswordForgot     = "password-forgot"
	scopePasswordReset      = "password-reset"
	scopeEmailVerify        = "email-verify"
	scopeEmailResend        = "email-resend"
	scopeGoogleStart        = "google-start"
	scopeGoogleCallback     = "google-callback"
	scopeSupportOpen        = "support-open"
	scopeSupportMessage     = "support-message"
	scopeSupportStream      = "support-stream"
	scopePurchase           = "purchase"
	scopeNutritionGenerate  = "nutrition-generate"
	scopeClientLookup       = "trainer-client-lookup"
	scopeDocumentUpload     = "document-upload"
	scopeProgramView        = "program-view"
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
	{name: scopeContact, ipLimit: 10, ipWindow: time.Hour},
	{name: scopeFeedback, ipLimit: 10, ipWindow: time.Hour},
	// Recovery requests are limited per address as well as per IP, so the
	// endpoint cannot flood one inbox; the service adds its own per-account
	// issue throttle on top.
	{name: scopePasswordForgot, ipLimit: 10, ipWindow: time.Hour, accountLimit: 5, accountWindow: time.Hour, accountField: "email"},
	{name: scopePasswordReset, ipLimit: 20, ipWindow: 15 * time.Minute},
	{name: scopeEmailVerify, ipLimit: 20, ipWindow: 15 * time.Minute},
	{name: scopeEmailResend, ipLimit: 10, ipWindow: time.Hour},
	// Google Sign-In: each start and callback costs a round trip to Google.
	{name: scopeGoogleStart, ipLimit: 30, ipWindow: 15 * time.Minute},
	{name: scopeGoogleCallback, ipLimit: 30, ipWindow: 15 * time.Minute},
	// Support chat: opening conversations is rare, messages are bounded so a
	// script cannot flood the support queue, and stream reconnections are
	// bounded so a broken client cannot hold the server in a reconnect loop.
	{name: scopeSupportOpen, ipLimit: 10, ipWindow: time.Hour},
	{name: scopeSupportMessage, ipLimit: 60, ipWindow: 10 * time.Minute},
	{name: scopeSupportStream, ipLimit: 120, ipWindow: 10 * time.Minute},
	// Purchases: each creation or payment start can open a provider checkout
	// session, so a script cannot spray the payment providers.
	{name: scopePurchase, ipLimit: 30, ipWindow: 10 * time.Minute},
	// Nutrition generation runs the plan builder on the server.
	{name: scopeNutritionGenerate, ipLimit: 20, ipWindow: time.Hour},
	// A trainer looks clients up by exact email; the bound keeps the lookup
	// from being used to test many addresses for an account.
	{name: scopeClientLookup, ipLimit: 60, ipWindow: 15 * time.Minute},
	// Trainer application documents: each upload is stored on disk.
	{name: scopeDocumentUpload, ipLimit: 30, ipWindow: time.Hour},
	// Program views: a visitor browsing many programs stays well inside the
	// bound; a script cannot inflate the featured ranking.
	{name: scopeProgramView, ipLimit: 120, ipWindow: 10 * time.Minute},
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
