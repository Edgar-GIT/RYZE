package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/google_auth"
	"ryze/backend/services/token"
)

// GoogleStateCookieName is the short-lived HttpOnly cookie binding a Google
// callback to the browser that started the sign-in.
const GoogleStateCookieName = "ryze_google_state"

// SessionVersions resolves the current session version of an account.
type SessionVersions interface {
	GetSessionVersion(ctx context.Context, userID string) (int, error)
}

// GoogleHandler exposes Google Sign-In. Start and Callback are browser
// navigations, so they answer with redirects to the web app rather than JSON.
type GoogleHandler struct {
	google     google_auth.Service
	tokens     token.Service
	sessions   SessionVersions
	ttl        time.Duration
	secure     bool
	appURL     string
	cookiePath string
}

func NewGoogleHandler(google google_auth.Service, tokens token.Service, sessions SessionVersions, ttl time.Duration, secure bool, appURL, cookiePath string) *GoogleHandler {
	if cookiePath == "" {
		cookiePath = "/"
	}
	return &GoogleHandler{google: google, tokens: tokens, sessions: sessions, ttl: ttl, secure: secure, appURL: appURL, cookiePath: cookiePath}
}

// Providers lists the external sign-in providers available right now.
func (h *GoogleHandler) Providers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sign-in providers retrieved.",
		"data":    gin.H{"google": h.google.Enabled()},
	})
}

func (h *GoogleHandler) stateCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     GoogleStateCookieName,
		Value:    value,
		Path:     h.cookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secure,
		// Lax: the cookie must travel on the top-level redirect back from
		// accounts.google.com.
		SameSite: http.SameSiteLaxMode,
	}
}

// redirectToApp sends the browser to a path of the web app.
func (h *GoogleHandler) redirectToApp(c *gin.Context, path string) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Redirect(http.StatusFound, h.appURL+path)
}

// sessionUser returns the account signed in on this request, or "" when the
// request carries no valid, current session.
func (h *GoogleHandler) sessionUser(c *gin.Context) string {
	return currentSessionUser(c, h.tokens, h.sessions)
}

// currentSessionUser validates the access-token cookie of a request and its
// session version, returning the account id or "" (no valid, current
// session). Long-lived requests use it to notice a session that ended.
func currentSessionUser(c *gin.Context, tokens token.Service, sessions SessionVersions) string {
	raw, err := c.Cookie(AccessTokenCookieName)
	if err != nil || raw == "" {
		return ""
	}
	claims, err := tokens.ValidateAccessToken(raw)
	if err != nil {
		return ""
	}
	version, err := sessions.GetSessionVersion(c.Request.Context(), claims.UserID)
	if err != nil || version != claims.SessionVersion {
		return ""
	}
	return claims.UserID
}

// Start begins a sign-in (mode=login) or connects Google to the signed-in
// account (mode=link), then redirects to Google's consent screen.
func (h *GoogleHandler) Start(c *gin.Context) {
	if !h.google.Enabled() {
		if h.appURL == "" {
			RespondError(c, http.StatusNotFound, "GOOGLE_SIGN_IN_UNAVAILABLE", "Google sign-in is not available.", nil)
			return
		}
		h.redirectToApp(c, "/login?oauth=unavailable")
		return
	}
	mode := google_auth.Mode(c.DefaultQuery("mode", string(google_auth.ModeLogin)))
	userID := ""
	if mode == google_auth.ModeLink {
		userID = h.sessionUser(c)
		if userID == "" {
			h.redirectToApp(c, "/login?oauth="+google_auth.ResultExpired)
			return
		}
	}
	start, err := h.google.Begin(mode, userID, c.Query("return"), c.Query("lang"))
	if err != nil {
		h.redirectToApp(c, "/login?oauth="+google_auth.ResultFailed)
		return
	}
	http.SetCookie(c.Writer, h.stateCookie(start.StateCookie, int((10*time.Minute).Seconds())))
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Redirect(http.StatusFound, start.URL)
}

// Callback finishes the flow Google redirects back to. The state cookie is
// always cleared, so each attempt can be completed once.
func (h *GoogleHandler) Callback(c *gin.Context) {
	stateValue, _ := c.Cookie(GoogleStateCookieName)
	http.SetCookie(c.Writer, h.stateCookie("", -1))

	outcome := h.google.Complete(c.Request.Context(), stateValue, google_auth.Callback{
		State: c.Query("state"),
		Code:  c.Query("code"),
		Error: c.Query("error"),
	}, h.sessionUser(c))

	if outcome.User != nil {
		accessToken, err := h.tokens.GenerateAccessToken(outcome.User.ID, outcome.User.SessionVersion)
		if err != nil {
			h.redirectToApp(c, "/login?oauth="+google_auth.ResultFailed)
			return
		}
		http.SetCookie(c.Writer, accessTokenCookie(accessToken, accessTokenLifetime(h.ttl), h.secure))
	}
	h.redirectToApp(c, outcome.Path)
}

type identityResponse struct {
	Provider   string     `json:"provider"`
	Email      string     `json:"email"`
	LinkedAt   time.Time  `json:"linked_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// ListIdentities returns the signed-in account's sign-in methods.
func (h *GoogleHandler) ListIdentities(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	summary, err := h.google.Summary(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, google_auth.ErrUserNotFound) {
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	identities := make([]identityResponse, 0, len(summary.Identities))
	for _, identity := range summary.Identities {
		identities = append(identities, identityResponse{
			Provider:   identity.Provider,
			Email:      identity.Email,
			LinkedAt:   identity.CreatedAt,
			LastUsedAt: identity.LastUsedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sign-in methods retrieved.",
		"data": gin.H{
			"password_set":     summary.PasswordSet,
			"google_available": h.google.Enabled(),
			"identities":       identities,
		},
	})
}

// UnlinkGoogle disconnects Google from the signed-in account.
func (h *GoogleHandler) UnlinkGoogle(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	switch err := h.google.Unlink(c.Request.Context(), userID); {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Google disconnected.", "data": gin.H{}})
	case errors.Is(err, google_auth.ErrPasswordRequired):
		RespondError(c, http.StatusConflict, "PASSWORD_REQUIRED", "Set a password before disconnecting Google.", nil)
	case errors.Is(err, google_auth.ErrNotLinked):
		RespondError(c, http.StatusNotFound, "GOOGLE_NOT_CONNECTED", "Google is not connected to this account.", nil)
	case errors.Is(err, google_auth.ErrUserNotFound):
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
