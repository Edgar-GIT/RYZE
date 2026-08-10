package authcontext

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrNoAuthenticatedUser is returned by UserIDFromContext when the context
// does not carry a valid authenticated user ID.
var ErrNoAuthenticatedUser = errors.New("no authenticated user in context")

// userIDContextKey is an unexported type so no other package can forge the
// context key used to carry the authenticated user UUID.
type userIDContextKey struct{}

// UserIDContextKey is the Gin context key holding the authenticated user UUID.
var UserIDContextKey = userIDContextKey{}

// UserIDFromContext returns the authenticated user UUID stored by the
// Authenticate middleware. It safely rejects missing values, values of the
// wrong type, empty strings and values that are not valid UUIDs. Pass the Gin
// context directly (for example `authcontext.UserIDFromContext(c)`).
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
