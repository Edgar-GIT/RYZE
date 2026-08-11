package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/services/token"
)

// AdminAuthenticate returns Gin middleware that authenticates requests through
// the admin HttpOnly session cookie (ryze_admin_access_token). The JWT is
// validated with the injected Token Service, which enforces the final admin
// token kind and rejects stage tokens, user tokens, malformed, expired,
// tampered and wrong-secret tokens. The resulting admin identity is
// additionally checked against the configured administrator identities
// (ADMIN_1/ADMIN_2) before it is stored in the request context. Every failure
// maps to the same indistinguishable HTTP 401 AUTHENTICATION_REQUIRED error.
// The middleware only depends on the Token Service and never touches GORM,
// MariaDB, the JWT secret or the admin credentials directly.
func AdminAuthenticate(tokens token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtValue, err := c.Cookie(auth.AdminAccessTokenCookieName)
		if err != nil {
			auth.RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			c.Abort()
			return
		}

		adminID, err := tokens.ValidateAdminToken(jwtValue)
		if err != nil {
			auth.RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			c.Abort()
			return
		}

		if !adminauthcontext.IsValidAdminIdentity(adminID) {
			auth.RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			c.Abort()
			return
		}

		c.Set(adminauthcontext.AdminIdentityContextKey, adminID)
		c.Next()
	}
}
