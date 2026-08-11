package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/middleware/adminroles"
)

// RequireAdminRole returns authorization middleware that MUST be mounted after
// AdminAuthenticate. It reads the authenticated admin identity from the
// context, resolves its role and allows the request only when the role is one
// of the required roles (at least one match is enough). The role is always
// derived from the authenticated identity stored by AdminAuthenticate, never
// from client input. Every denial maps to the same indistinguishable HTTP 403
// FORBIDDEN error without revealing the required roles or the admin's role.
func RequireAdminRole(required ...adminroles.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID, err := adminauthcontext.AdminIdentityFromContext(c)
		if err != nil {
			auth.RespondError(c, http.StatusForbidden, "FORBIDDEN", "Forbidden.", nil)
			c.Abort()
			return
		}
		if !adminroles.HasRole(adminID, required...) {
			auth.RespondError(c, http.StatusForbidden, "FORBIDDEN", "Forbidden.", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
