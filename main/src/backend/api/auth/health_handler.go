package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/version"
)

// healthPingTimeout bounds the database probe performed by the health endpoint,
// so an unreachable database cannot make the health check itself hang.
const healthPingTimeout = 2 * time.Second

// DatabasePinger reports whether the database is reachable. A *sql.DB satisfies
// it. It is kept as a narrow interface so the health handler can be tested
// without a real database.
type DatabasePinger interface {
	PingContext(ctx context.Context) error
}

// HealthHandler serves the public monitoring endpoints. It exposes no internal
// information: the status is a fixed word and the version is the build version.
type HealthHandler struct {
	db DatabasePinger
}

// NewHealthHandler returns a health handler. When db is nil the status endpoint
// reports the process as healthy without probing a database, which keeps the
// public endpoint available even before the database handle is wired.
func NewHealthHandler(db DatabasePinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// Status reports the API status. When a database handle is configured the
// database is probed, and an unreachable database answers 503
// SERVICE_UNAVAILABLE so an orchestrator does not route traffic to an instance
// that cannot serve requests.
func (h *HealthHandler) Status(c *gin.Context) {
	if h.db != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), healthPingTimeout)
		defer cancel()

		if err := h.db.PingContext(ctx); err != nil {
			RespondError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service unavailable.", nil)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service is healthy.",
		"data":    gin.H{"status": "ok"},
	})
}

// Version reports the running API version the binary was built with.
func (h *HealthHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Version retrieved successfully.",
		"data":    gin.H{"version": version.Version},
	})
}
