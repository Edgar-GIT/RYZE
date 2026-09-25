package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/logs"
	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/services/test_mode"
	"ryze/backend/services/token"
)

// testModeEnterRequest is the request DTO for entering Test Mode. It carries
// no validation tags: semantic validation is owned by the service.
type testModeEnterRequest struct {
	Persona    string `json:"persona"`
	ReturnPath string `json:"return_path"`
}

// sessionVersionProvider is the narrow data-access surface the handler needs
// to mint a persona access token bound to the current session version.
type sessionVersionProvider interface {
	GetSessionVersion(ctx context.Context, id string) (int, error)
}

// TestModeHandler exposes the Test Mode lifecycle. Enter is only reachable by
// an authenticated TECHNICAL_ADMINISTRATOR (enforced by the middleware mounted
// on the route) so the effective identity is always established server-side.
// Exit deliberately has no admin guard: the admin session is cleared on enter,
// so the only trustworthy marker of the active Test Mode is the session cookie.
type TestModeHandler struct {
	testMode     test_mode.Service
	tokens       token.Service
	sessions     sessionVersionProvider
	tokenTTL     time.Duration
	cookieSecure bool
}

func NewTestModeHandler(
	svc test_mode.Service,
	tokens token.Service,
	sessions sessionVersionProvider,
	tokenTTL time.Duration,
	cookieSecure bool,
) *TestModeHandler {
	return &TestModeHandler{testMode: svc, tokens: tokens, sessions: sessions, tokenTTL: tokenTTL, cookieSecure: cookieSecure}
}

// Enter starts a Test Mode session for the requested persona on behalf of the
// authenticated administrator. On success it mints a real persona access
// token, stores the session token in the ryze_test_session cookie, clears the
// admin session and signs the persona in. The test session and persona tokens
// share the access token lifetime so both expire together.
func (h *TestModeHandler) Enter(c *gin.Context) {
	adminID, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	var req testModeEnterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	result, err := h.testMode.Enter(c.Request.Context(), adminID, req.Persona, req.ReturnPath)
	if err != nil {
		h.respondTestModeError(c, err)
		return
	}

	sessionVersion, err := h.sessions.GetSessionVersion(c.Request.Context(), result.PersonaUserID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}

	personaToken, err := h.tokens.GenerateAccessToken(result.PersonaUserID, sessionVersion)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}

	maxAge := accessTokenLifetime(h.tokenTTL)
	http.SetCookie(c.Writer, adminAccessTokenCookie("", -1, h.cookieSecure))
	http.SetCookie(c.Writer, testSessionTokenCookie(result.Token, maxAge, h.cookieSecure))
	http.SetCookie(c.Writer, accessTokenCookie(personaToken, maxAge, h.cookieSecure))

	logs.Info("test_mode.entered",
		slog.String("admin_identity", adminID),
		slog.String("persona", result.Persona),
		slog.String("persona_user_id", result.PersonaUserID),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test Mode entered successfully.",
		"data": gin.H{
			"persona":     result.Persona,
			"return_path": result.ReturnPath,
		},
	})
}

// Exit ends the active Test Mode session, restores the original administrator
// identity by re-issuing a fresh admin session cookie and redirects out of the
// persona session. Only the session cookie is required, so an exited
// administrator is signed back into the admin session without any admin
// credential being replayed.
func (h *TestModeHandler) Exit(c *gin.Context) {
	rawToken, err := c.Cookie(TestSessionTokenCookieName)
	if err != nil {
		RespondError(c, http.StatusForbidden, "TEST_MODE_INACTIVE", "No active Test Mode session.", nil)
		return
	}

	result, err := h.testMode.Exit(c.Request.Context(), rawToken)
	if err != nil {
		h.respondTestModeError(c, err)
		return
	}

	adminToken, err := h.tokens.GenerateAdminToken(result.AdminIdentity)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}

	maxAge := accessTokenLifetime(h.tokenTTL)
	http.SetCookie(c.Writer, testSessionTokenCookie("", -1, h.cookieSecure))
	http.SetCookie(c.Writer, accessTokenCookie("", -1, h.cookieSecure))
	http.SetCookie(c.Writer, adminAccessTokenCookie(adminToken, maxAge, h.cookieSecure))

	logs.Info("test_mode.exited",
		slog.String("admin_identity", result.AdminIdentity),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test Mode exited successfully.",
		"data": gin.H{
			"return_path": result.ReturnPath,
		},
	})
}

// Status reports whether a Test Mode session is active and, when present, the
// persona currently impersonated. It is a public read-only endpoint that never
// reveals the session token.
func (h *TestModeHandler) Status(c *gin.Context) {
	rawToken, err := c.Cookie(TestSessionTokenCookieName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Test Mode status retrieved successfully.",
			"data":    gin.H{"active": false},
		})
		return
	}

	session, err := h.testMode.ActivePersona(c.Request.Context(), rawToken)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Test Mode status retrieved successfully.",
			"data":    gin.H{"active": false},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test Mode status retrieved successfully.",
		"data": gin.H{
			"active":  true,
			"persona": session.Persona,
		},
	})
}

func (h *TestModeHandler) respondTestModeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, test_mode.ErrDisabled):
		RespondError(c, http.StatusForbidden, "TEST_MODE_DISABLED", "Test Mode is disabled.", nil)
	case errors.Is(err, test_mode.ErrUnknownPersona),
		errors.Is(err, test_mode.ErrInvalidInput),
		errors.Is(err, test_mode.ErrInvalidReturnPath):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, test_mode.ErrNoActiveSession):
		RespondError(c, http.StatusForbidden, "TEST_MODE_INACTIVE", "No active Test Mode session.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
