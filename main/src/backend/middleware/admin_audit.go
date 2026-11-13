package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/services/admin_audit"
	"ryze/backend/services/token"
)

// AuditRecorder stores one administrative audit entry.
type AuditRecorder interface {
	Record(ctx context.Context, entry admin_audit.Entry) error
}

// auditWriteTimeout bounds the audit write that follows a request. The write
// runs on a context detached from the request so a client that disconnects
// cannot cancel it.
const auditWriteTimeout = 3 * time.Second

// AdminAudit records every auditable request (see admin_audit.ShouldAudit)
// made by an authenticated administrator. It must run after AdminAuthenticate
// and before any permission check, so denied attempts are recorded as well.
// A failed write is logged without request data and never changes the
// response.
func AdminAudit(recorder AuditRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		route := c.FullPath()
		if recorder == nil || !admin_audit.ShouldAudit(c.Request.Method, route) {
			c.Next()
			return
		}
		adminID, err := adminauthcontext.AdminIdentityFromContext(c)
		if err != nil {
			c.Next()
			return
		}

		c.Next()

		recordAudit(c, recorder, admin_audit.Entry{
			AdminID:    adminID,
			Method:     c.Request.Method,
			Route:      route,
			TargetID:   auditTarget(c),
			StatusCode: c.Writer.Status(),
			ClientIP:   c.ClientIP(),
		})
	}
}

// AdminSignInAudit records second-factor sign-in attempts. The pending login
// is identified by the stage cookie set by the first step; requests without a
// valid stage cookie are not attributable to an administrator and are not
// recorded (the rate limiter already bounds them).
func AdminSignInAudit(recorder AuditRecorder, tokens token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if recorder == nil {
			c.Next()
			return
		}
		stage, err := c.Cookie(auth.AdminStageTokenCookieName)
		if err != nil || stage == "" {
			c.Next()
			return
		}
		adminID, err := tokens.ValidateAdminStageToken(stage)
		if err != nil {
			c.Next()
			return
		}

		c.Next()

		recordAudit(c, recorder, admin_audit.Entry{
			AdminID:    adminID,
			Action:     admin_audit.ActionSignIn,
			Method:     c.Request.Method,
			Route:      c.FullPath(),
			StatusCode: c.Writer.Status(),
			ClientIP:   c.ClientIP(),
		})
	}
}

func recordAudit(c *gin.Context, recorder AuditRecorder, entry admin_audit.Entry) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), auditWriteTimeout)
	defer cancel()
	if err := recorder.Record(ctx, entry); err != nil {
		slog.Error("admin audit write failed", "action", admin_audit.ActionFor(entry.Method, entry.Route), "error", err)
	}
}

func auditTarget(c *gin.Context) string {
	for _, name := range admin_audit.TargetParams {
		if value := c.Param(name); value != "" {
			return value
		}
	}
	return ""
}
