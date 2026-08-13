package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/middleware/trainercontext"
	"ryze/backend/repositories"
)

// TrainerAuthenticate returns Gin middleware that verifies the already
// authenticated user owns an active trainer profile.
//
// Authentication itself remains the responsibility of Authenticate. This
// middleware only resolves the authenticated user's active Trainer identity
// and stores both UUIDs in trainercontext.
//
// An authenticated user who is not an active trainer receives 403 FORBIDDEN.
// Missing or invalid user authentication is handled by Authenticate and is
// therefore not duplicated here.
func TrainerAuthenticate(trainers repositories.TrainerRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := authcontext.UserIDFromContext(c)
		if err != nil {
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

		trainer, err := trainers.FindByUserID(c.Request.Context(), userID)
		if err != nil {
			if errors.Is(err, repositories.ErrTrainerNotFound) {
				auth.RespondError(
					c,
					http.StatusForbidden,
					"FORBIDDEN",
					"Forbidden.",
					nil,
				)
			} else {
				auth.RespondError(
					c,
					http.StatusInternalServerError,
					"INTERNAL_ERROR",
					"Internal server error.",
					nil,
				)
			}
			c.Abort()
			return
		}

		trainercontext.SetIdentity(c, trainercontext.Identity{
			UserID:    userID,
			TrainerID: trainer.ID,
		})

		c.Next()
	}
}
