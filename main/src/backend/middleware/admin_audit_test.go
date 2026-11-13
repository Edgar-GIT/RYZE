package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/middleware"
	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/services/admin_audit"
	"ryze/backend/services/token"
)

type recordingAuditor struct {
	mu      sync.Mutex
	entries []admin_audit.Entry
}

func (r *recordingAuditor) Record(_ context.Context, entry admin_audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, entry)
	return nil
}

func auditEngine(recorder middleware.AuditRecorder, adminID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api/v1/admin")
	group.Use(func(c *gin.Context) {
		if adminID != "" {
			c.Set(adminauthcontext.AdminIdentityContextKey, adminID)
		}
		c.Next()
	})
	group.Use(middleware.AdminAudit(recorder))
	group.GET("/users", func(c *gin.Context) { c.Status(http.StatusOK) })
	group.PATCH("/users/:id/disable", func(c *gin.Context) { c.Status(http.StatusOK) })
	group.POST("/trainer-applications/:id/approve", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusForbidden)
	})
	group.GET("/trainer-applications/:id/documents/:documentID/download", func(c *gin.Context) { c.Status(http.StatusOK) })
	return engine
}

func TestAdminAuditRecordsMutationsAndDownloadsOnly(t *testing.T) {
	recorder := &recordingAuditor{}
	engine := auditEngine(recorder, config.Admin1ID)

	for _, request := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/admin/users"},
		{http.MethodPatch, "/api/v1/admin/users/u-1/disable"},
		{http.MethodPost, "/api/v1/admin/trainer-applications/app-9/approve"},
		{http.MethodGet, "/api/v1/admin/trainer-applications/app-9/documents/doc-3/download"},
	} {
		engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(request.method, request.path, nil))
	}

	if len(recorder.entries) != 3 {
		t.Fatalf("recorded %d entries, want 3 (reads are not audited): %+v", len(recorder.entries), recorder.entries)
	}
	disable := recorder.entries[0]
	if disable.AdminID != config.Admin1ID || disable.Route != "/api/v1/admin/users/:id/disable" || disable.TargetID != "u-1" || disable.StatusCode != http.StatusOK {
		t.Fatalf("unexpected disable entry: %+v", disable)
	}
	if denied := recorder.entries[1]; denied.StatusCode != http.StatusForbidden || admin_audit.OutcomeFor(denied.StatusCode) != "denied" {
		t.Fatalf("a denied attempt must be recorded with its status: %+v", denied)
	}
	if download := recorder.entries[2]; download.TargetID != "app-9" || admin_audit.ActionFor(download.Method, download.Route) != "trainer_application.document_download" {
		t.Fatalf("unexpected download entry: %+v", download)
	}
}

func TestAdminAuditSkipsRequestsWithoutAdminIdentity(t *testing.T) {
	recorder := &recordingAuditor{}
	engine := auditEngine(recorder, "")
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/u-1/disable", nil))
	if len(recorder.entries) != 0 {
		t.Fatalf("an unauthenticated request must not be attributed: %+v", recorder.entries)
	}
}

func TestAdminSignInAuditUsesTheStageCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens := token.NewService([]byte("audit-test-secret-that-is-longer-than-32-bytes"), 15*time.Minute)
	tokens.RegisterAdminCredentials(config.Admin2ID, "admin-2-password", "admin-2-access-code")
	recorder := &recordingAuditor{}

	status := http.StatusUnauthorized
	engine := gin.New()
	engine.POST("/api/v1/admin/auth/verify", middleware.AdminSignInAudit(recorder, tokens), func(c *gin.Context) {
		c.Status(status)
	})

	stage, err := tokens.GenerateAdminStageToken(config.Admin2ID)
	if err != nil {
		t.Fatalf("stage token: %v", err)
	}
	send := func(cookie string) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/verify", nil)
		if cookie != "" {
			request.AddCookie(&http.Cookie{Name: auth.AdminStageTokenCookieName, Value: cookie})
		}
		engine.ServeHTTP(httptest.NewRecorder(), request)
	}

	send("")            // no pending login: not attributable
	send("not-a-token") // forged cookie: not attributable
	send(stage)         // wrong access code
	status = http.StatusOK
	send(stage) // completed sign-in

	if len(recorder.entries) != 2 {
		t.Fatalf("recorded %d entries, want 2: %+v", len(recorder.entries), recorder.entries)
	}
	for i, want := range []int{http.StatusUnauthorized, http.StatusOK} {
		entry := recorder.entries[i]
		if entry.AdminID != config.Admin2ID || entry.Action != admin_audit.ActionSignIn || entry.StatusCode != want {
			t.Fatalf("entry %d = %+v, want ADMIN_2 sign-in with status %d", i, entry, want)
		}
	}
}
