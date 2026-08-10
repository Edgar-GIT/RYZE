package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/token"
)

// Authenticate returns Gin middleware that authenticates requests through the
// HttpOnly access-token cookie. The JWT is validated with the injected Token
// Service (no GORM or database access) and the resulting user UUID is stored
// in the request context. All authentication failures map to the same
// indistinguishable HTTP 401 AUTHENTICATION_REQUIRED error.
func Authenticate(tokens token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtValue, err := c.Cookie(auth.AccessTokenCookieName)
		if err != nil {
			auth.RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			c.Abort()
			return
		}

		userID, err := tokens.ValidateAccessToken(jwtValue)
		if err != nil {
			auth.RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			c.Abort()
			return
		}

		c.Set(authcontext.UserIDContextKey, userID)
		c.Next()
	}
}
