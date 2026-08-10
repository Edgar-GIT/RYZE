package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LogoutHandler clears the browser authentication cookie.
type LogoutHandler struct {
	secure bool
}

func NewLogoutHandler(secure bool) *LogoutHandler {
	return &LogoutHandler{secure: secure}
}

// Logout invalidates ryze_access_token regardless of whether the cookie is
// missing, expired or malformed. It performs no database access and does not
// validate the existing token.
func (h *LogoutHandler) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, accessTokenCookie("", -1, h.secure))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logout successful.",
		"data":    gin.H{},
	})
}
