package auth

import (
	"net/http"
	"time"
)

// AccessTokenCookieName is the HttpOnly cookie carrying the access token.
const AccessTokenCookieName = "ryze_access_token"

// AdminAccessTokenCookieName is the HttpOnly cookie carrying the admin access
// token. It intentionally differs from the client cookie so an admin session
// can never be confused with a regular user session in the same browser.
const AdminAccessTokenCookieName = "ryze_admin_access_token"

// AdminStageTokenCookieName is the HttpOnly cookie carrying the temporary
// authentication state issued after the username/password stage of the admin
// login flow. It is never accepted as an admin session.
const AdminStageTokenCookieName = "ryze_admin_stage_token"

// accessTokenCookie builds the ryze_access_token cookie shared by login and
// logout so the two endpoints always use the same attributes. A negative
// maxAge clears the cookie (Max-Age=0 plus an expired Expires).
func accessTokenCookie(value string, maxAge int, secure bool) *http.Cookie {
	return authCookie(AccessTokenCookieName, value, maxAge, secure)
}

// adminAccessTokenCookie builds the ryze_admin_access_token cookie for the
// admin login flow using the same secure attributes as the user cookie.
func adminAccessTokenCookie(value string, maxAge int, secure bool) *http.Cookie {
	return authCookie(AdminAccessTokenCookieName, value, maxAge, secure)
}

// adminStageTokenCookie builds the short-lived ryze_admin_stage_token cookie
// holding the temporary authentication state between the two admin login
// stages.
func adminStageTokenCookie(value string, maxAge int, secure bool) *http.Cookie {
	return authCookie(AdminStageTokenCookieName, value, maxAge, secure)
}

// authCookie builds an HttpOnly session cookie shared by every authentication
// flow. A negative maxAge clears the cookie (Max-Age=0 plus an expired
// Expires).
func authCookie(name, value string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// accessTokenLifetime returns the Max-Age in seconds for the configured TTL.
func accessTokenLifetime(ttl time.Duration) int {
	return int(ttl.Seconds())
}
