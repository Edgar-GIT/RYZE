package adminauthcontext

import (
	"errors"

	"github.com/gin-gonic/gin"

	"ryze/backend/config"
)

// ErrNoAuthenticatedAdmin is returned by AdminIdentityFromContext when the
// context does not carry a valid authenticated admin identity.
var ErrNoAuthenticatedAdmin = errors.New("no authenticated admin in context")

// adminIdentityContextKey is an unexported type so no other package can forge
// the context key used to carry the authenticated admin identity.
type adminIdentityContextKey struct{}

// AdminIdentityContextKey is the Gin context key holding the authenticated
// admin identity.
var AdminIdentityContextKey = adminIdentityContextKey{}

// IsValidAdminIdentity reports whether id is a configured administrator
// identity (ADMIN_1 or ADMIN_2). Both the AdminAuthenticate middleware and
// AdminIdentityFromContext use the same check, so an arbitrary string can
// never be treated as a valid admin identity.
func IsValidAdminIdentity(id string) bool {
	return config.IsValidAdminIdentity(id)
}

// AdminIdentityFromContext returns the authenticated admin identity stored by
// the AdminAuthenticate middleware. It safely rejects missing values, values
// of the wrong type, empty strings and values that are not configured admin
// identities. Pass the Gin context directly (for example
// `adminauthcontext.AdminIdentityFromContext(c)`).
func AdminIdentityFromContext(c *gin.Context) (string, error) {
	value, exists := c.Get(AdminIdentityContextKey)
	if !exists || value == nil {
		return "", ErrNoAuthenticatedAdmin
	}

	adminID, ok := value.(string)
	if !ok || adminID == "" {
		return "", ErrNoAuthenticatedAdmin
	}
	if !IsValidAdminIdentity(adminID) {
		return "", ErrNoAuthenticatedAdmin
	}
	return adminID, nil
}
