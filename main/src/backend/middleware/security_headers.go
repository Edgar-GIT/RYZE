package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets defensive headers on every API response. The API only
// returns JSON (and authenticated file downloads), so it is never framed,
// never content-sniffed and never leaks its URLs through Referer. Responses on
// account, trainer and admin paths carry personal data and are never stored
// by browsers or shared caches.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Writer.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Cross-Origin-Resource-Policy", "same-site")
		if isPrivatePath(c.Request.URL.Path) {
			header.Set("Cache-Control", "no-store")
		}
		c.Next()
	}
}

func isPrivatePath(path string) bool {
	for _, prefix := range []string{"/api/v1/me", "/api/v1/trainer", "/api/v1/admin", "/api/v1/auth", "/api/v1/programs/generic", "/api/v1/programs/premium-level-"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
