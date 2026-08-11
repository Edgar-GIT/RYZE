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

// accessTokenCookie builds the ryze_access_token cookie shared by login and
// logout so the two endpoints always use the same attributes. A negative
// maxAge clears the cookie (Max-Age=0 plus an expired Expires).
func accessTokenCookie(value string, maxAge int, secure bool) *http.Cookie {
	return authCookie(AccessTokenCookieName, value, maxAge, secure)
}

// adminAccessTokenCookie builds the ryze_admin_access_token cookie for the
// admin login endpoint using the same secure attributes as the user cookie.
func adminAccessTokenCookie(value string, maxAge int, secure bool) *http.Cookie {
	return authCookie(AdminAccessTokenCookieName, value, maxAge, secure)
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
