package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
)

// OriginGuard rejects browser-originated state-changing requests whose origin
// is neither the request's own host nor an explicitly configured origin. It is
// defense-in-depth on top of the SameSite session cookies: SameSite already
// blocks cross-site cookie submission in modern browsers, but it treats every
// subdomain of the same site as trusted, so a compromised sibling host could
// otherwise drive authenticated state changes without the user's intent.
//
// Requests without any browser origin context (no Origin and no Referer) are
// allowed: non-browser clients such as curl, provider webhooks and internal
// tooling never carry origin headers, and CSRF is a browser-only attack class.
// A malformed, opaque ("null") or untrusted origin is rejected, and the
// rejection never depends on whether a valid session cookie is present.
func OriginGuard(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if normalized := normalizeOrigin(origin); normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		origin := requestOrigin(c.Request)
		if origin == "" {
			c.Next()
			return
		}
		if _, ok := allowed[origin]; ok {
			c.Next()
			return
		}
		if sameHostOrigin(c.Request.Host, origin) {
			c.Next()
			return
		}

		auth.RespondError(c, http.StatusForbidden, "CROSS_ORIGIN_FORBIDDEN", "Requests from this origin are not allowed.", nil)
		c.Abort()
	}
}

// isSafeMethod reports whether the method is defined as a safe, read-only
// method by HTTP semantics and therefore exempt from origin checking.
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// requestOrigin returns the normalized origin of a browser request, or an
// empty string when the request carries no browser origin context at all. A
// present but unparsable Origin or Referer is returned as a value that can
// never match an allowed origin, so it is rejected instead of being mistaken
// for a non-browser client.
func requestOrigin(r *http.Request) string {
	if raw := strings.TrimSpace(r.Header.Get("Origin")); raw != "" {
		if normalized := normalizeOrigin(raw); normalized != "" {
			return normalized
		}
		return "invalid"
	}

	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer == "" {
		return ""
	}
	parsed, err := url.Parse(referer)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "invalid"
	}
	if normalized := normalizeOrigin(parsed.Scheme + "://" + parsed.Host); normalized != "" {
		return normalized
	}
	return "invalid"
}

// normalizeOrigin reduces an origin to lowercase scheme and host. Missing or
// non-HTTP(S) schemes and origins without a host are not valid and return an
// empty string.
func normalizeOrigin(origin string) string {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	return scheme + "://" + strings.ToLower(parsed.Host)
}

// sameHostOrigin reports whether an origin names the request's own host. The
// scheme is not compared because a TLS-terminating reverse proxy may forward
// plain HTTP to the application while the browser used HTTPS; the host and
// port must still match exactly.
func sameHostOrigin(host, origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, host)
}
