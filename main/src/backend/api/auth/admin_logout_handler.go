package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminLogoutHandler clears the administrator authentication cookies.
type AdminLogoutHandler struct {
	secure bool
}

func NewAdminLogoutHandler(secure bool) *AdminLogoutHandler {
	return &AdminLogoutHandler{secure: secure}
}

// Logout invalidates ryze_admin_access_token and ryze_admin_stage_token
// regardless of whether they are missing, expired or malformed. It performs no
// database access and does not validate the existing token, so an expired admin
// session can always end cleanly. It sits outside the AdminAuthenticate
// middleware for the same reason.
func (h *AdminLogoutHandler) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, adminAccessTokenCookie("", -1, h.secure))
	http.SetCookie(c.Writer, adminStageTokenCookie("", -1, h.secure))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logout successful.",
		"data":    gin.H{},
	})
}