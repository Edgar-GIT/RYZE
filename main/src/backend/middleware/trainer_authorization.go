package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/trainercontext"
	"ryze/backend/middleware/trainerroles"
)

// RequireTrainerPermission returns authorization middleware that MUST be
// mounted after TrainerAuthenticate. It reads the authenticated trainer
// identity from trainercontext and allows the request only when the identity
// holds at least one of the required permissions. The identity is always
// derived from the trainercontext stored by TrainerAuthenticate, never from
// client input (no user IDs, trainer IDs, permissions, roles, headers, query
// or body values are trusted).
//
// Missing or invalid trainer identity maps to the same indistinguishable HTTP
// 401 AUTHENTICATION_REQUIRED error, while missing permissions map to HTTP 403
// FORBIDDEN. Every denial uses the generic response envelope and never reveals
// the required permissions or the trainer identity.
func RequireTrainerPermission(required ...trainerroles.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := trainercontext.IdentityFromContext(c); err != nil {
			auth.RespondError(
				c,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"Authentication required.",
				nil,
			)
			c.Abort()
			return
		}

		for _, permission := range required {
			if trainerroles.HasPermission(permission) {
				c.Next()
				return
			}
		}

		auth.RespondError(c, http.StatusForbidden, "FORBIDDEN", "Forbidden.", nil)
		c.Abort()
	}
}
