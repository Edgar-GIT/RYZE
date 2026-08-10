package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/api/auth"
	"ryze/backend/services/token"
)

// ErrNoAuthenticatedUser is returned by UserIDFromContext when the context
// does not carry a valid authenticated user ID.
var ErrNoAuthenticatedUser = errors.New("no authenticated user in context")

// userIDContextKey is an unexported type so no other package can forge the
// context key used to carry the authenticated user UUID.
type userIDContextKey struct{}

// UserIDContextKey is the Gin context key holding the authenticated user UUID.
var UserIDContextKey = userIDContextKey{}

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

		c.Set(UserIDContextKey, userID)
		c.Next()
	}
}

// UserIDFromContext returns the authenticated user UUID stored by the
// Authenticate middleware. It safely rejects missing values, values of the
// wrong type, empty strings and values that are not valid UUIDs. Pass the Gin
// context directly (for example `middleware.UserIDFromContext(c)`).
func UserIDFromContext(c *gin.Context) (string, error) {
	value, exists := c.Get(UserIDContextKey)
	if !exists || value == nil {
		return "", ErrNoAuthenticatedUser
	}

	userID, ok := value.(string)
	if !ok || userID == "" {
		return "", ErrNoAuthenticatedUser
	}
	if _, err := uuid.Parse(userID); err != nil {
		return "", ErrNoAuthenticatedUser
	}
	return userID, nil
}
