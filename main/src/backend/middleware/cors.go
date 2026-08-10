package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

// corsMethods are the HTTP methods allowed for preflighted requests.
var corsMethods = []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions}

// corsHeaders are the request headers allowed for preflighted requests.
var corsHeaders = []string{"Content-Type", "Authorization"}

// CORS returns a minimal cross-origin middleware. It never uses the wildcard
// origin because the auth cookie requires explicit origin allow-listing, and it
// performs no new library dependency. Only origins present in allowedOrigins
// receive CORS headers.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin != "" && slices.Contains(allowedOrigins, origin)

		if origin != "" {
			c.Header("Vary", "Origin")
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			if allowed {
				c.Header("Access-Control-Allow-Methods", join(corsMethods))
				c.Header("Access-Control-Allow-Headers", join(corsHeaders))
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func join(values []string) string {
	result := ""
	for i, value := range values {
		if i > 0 {
			result += ", "
		}
		result += value
	}
	return result
}
