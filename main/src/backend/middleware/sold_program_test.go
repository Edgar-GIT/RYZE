package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware"
	"ryze/backend/middleware/trainercontext"
)

func TestRefuseSoldProgram(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sold := map[string]bool{"sold": true}
	const trainerID = "6f1c2a3b-0000-4000-8000-000000000002"
	router := gin.New()
	router.Use(func(c *gin.Context) {
		trainercontext.SetIdentity(c, trainercontext.Identity{UserID: "6f1c2a3b-0000-4000-8000-000000000001", TrainerID: trainerID})
		c.Next()
	})
	router.PATCH("/programs/:programID", middleware.RefuseSoldProgram(func(_ context.Context, owner, programID string) (bool, error) {
		return owner == trainerID && sold[programID], nil
	}), func(c *gin.Context) { c.Status(http.StatusOK) })

	for id, want := range map[string]int{"sold": http.StatusConflict, "fresh": http.StatusOK} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/programs/"+id, nil))
		if rec.Code != want {
			t.Fatalf("%s: expected %d, got %d (%s)", id, want, rec.Code, rec.Body.String())
		}
	}
}
