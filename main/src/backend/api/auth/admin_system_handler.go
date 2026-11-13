package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/version"
)

// SystemFeatures is the non-secret summary of what this deployment has
// switched on. It never carries credentials, hosts or keys.
type SystemFeatures struct {
	EmailDelivery     bool     `json:"email_delivery"`
	GoogleSignIn      bool     `json:"google_sign_in"`
	PaymentMethods    []string `json:"payment_methods"`
	ExerciseMedia     bool     `json:"exercise_media"`
	RateLimiting      bool     `json:"rate_limiting"`
	TestMode          bool     `json:"test_mode"`
	ContactDelivery   bool     `json:"contact_delivery"`
	SecureCookies     bool     `json:"secure_cookies"`
	DefaultCommission uint32   `json:"default_platform_commission_bps"`
	SessionSeconds    int64    `json:"session_lifetime_seconds"`
}

// AdminSystemHandler serves the Technical Administrator's system view.
type AdminSystemHandler struct {
	db       *gorm.DB
	features SystemFeatures
	started  time.Time
}

func NewAdminSystemHandler(db *gorm.DB, features SystemFeatures) *AdminSystemHandler {
	if features.PaymentMethods == nil {
		features.PaymentMethods = []string{}
	}
	return &AdminSystemHandler{db: db, features: features, started: time.Now().UTC()}
}

type queueCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// Status reports the API version, uptime, database reachability and schema
// version, the email queue by status, and the enabled features.
func (h *AdminSystemHandler) Status(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	database := gin.H{"reachable": false}
	if sqlDB, err := h.db.DB(); err == nil && sqlDB.PingContext(ctx) == nil {
		database["reachable"] = true
		var schemaVersion int64
		var dirty bool
		if err := h.db.WithContext(ctx).Raw("SELECT version, dirty FROM schema_migrations LIMIT 1").Row().Scan(&schemaVersion, &dirty); err == nil {
			database["schema_version"] = schemaVersion
			database["schema_dirty"] = dirty
		}
		stats := sqlDB.Stats()
		database["open_connections"] = stats.OpenConnections
	}

	queue := []queueCount{}
	_ = h.db.WithContext(ctx).Raw("SELECT status, COUNT(*) AS count FROM email_outbox GROUP BY status ORDER BY status").Scan(&queue).Error

	var visibleNotifications, openSupport int64
	_ = h.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM notifications WHERE expires_at > now()").Row().Scan(&visibleNotifications)
	_ = h.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM support_conversations WHERE status = 'open'").Row().Scan(&openSupport)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "System status retrieved.",
		"data": gin.H{
			"version":               version.Version,
			"started_at":            h.started,
			"uptime_seconds":        int64(time.Since(h.started).Seconds()),
			"database":              database,
			"email_queue":           queue,
			"visible_notifications": visibleNotifications,
			"open_support":          openSupport,
			"features":              h.features,
		},
	})
}
