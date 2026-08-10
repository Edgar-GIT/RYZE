package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/repositories"
	"ryze/backend/services/token"
)

// SessionProvider resolves the current session version of an authenticated
// user so access tokens issued for a revoked session (e.g. after a password
// change) can be rejected.
type SessionProvider interface {
	GetSessionVersion(ctx context.Context, userID string) (int, error)
}

// Authenticate returns Gin middleware that authenticates requests through the
// HttpOnly access-token cookie. The JWT is validated with the injected Token
// Service and its session version is compared against the current one resolved
// by the SessionProvider; a token issued for an older session is rejected. The
// resulting user UUID is stored in the request context. All authentication
// failures map to the same indistinguishable HTTP 401 AUTHENTICATION_REQUIRED
// error.
func Authenticate(tokens token.Service, sessions SessionProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtValue, err := c.Cookie(auth.AccessTokenCookieName)
		if err != nil {
			auth.RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			c.Abort()
			return
		}

		claims, err := tokens.ValidateAccessToken(jwtValue)
		if err != nil {
			auth.RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			c.Abort()
			return
		}

		currentVersion, err := sessions.GetSessionVersion(c.Request.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, repositories.ErrUserNotFound) {
				auth.RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			} else {
				auth.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
			}
			c.Abort()
			return
		}

		if claims.SessionVersion != currentVersion {
			auth.RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			c.Abort()
			return
		}

		c.Set(authcontext.UserIDContextKey, claims.UserID)
		c.Next()
	}
}
