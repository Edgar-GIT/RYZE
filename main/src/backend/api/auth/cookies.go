package auth

import (
	"net/http"
	"time"
)

// AccessTokenCookieName is the HttpOnly cookie carrying the access token.
const AccessTokenCookieName = "ryze_access_token"

// accessTokenCookie builds the ryze_access_token cookie shared by login and
// logout so the two endpoints always use the same attributes. A negative
// maxAge clears the cookie (Max-Age=0 plus an expired Expires).
func accessTokenCookie(value string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     AccessTokenCookieName,
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
