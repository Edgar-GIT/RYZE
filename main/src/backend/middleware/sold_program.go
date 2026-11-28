package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/trainercontext"
)

// RefuseSoldProgram blocks direct changes to a sold programme (v2.133.0, owner
// decision 2026-11-27: a buyer keeps the version they bought). The trainer
// edits a sold package through a new version (POST
// /trainer/programs/:programID/versions). Ownership is still checked by the
// handler; this gate answers "has this trainer's own programme been sold", so
// it never tells anything about another trainer's programmes.
func RefuseSoldProgram(isSold func(ctx context.Context, trainerID, programID string) (bool, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		programID := c.Param("programID")
		trainerID, err := trainercontext.TrainerIDFromContext(c)
		if programID == "" || err != nil {
			c.Next()
			return
		}
		sold, err := isSold(c.Request.Context(), trainerID, programID)
		if err != nil {
			auth.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
			c.Abort()
			return
		}
		if sold {
			auth.RespondError(c, http.StatusConflict, "PROGRAM_SOLD",
				"This package has been sold: edit it through a new version; buyers keep the version they bought.", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
