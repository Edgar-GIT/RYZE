package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/middleware/adminroles"
)

// AdminMeHandler returns the authenticated administrator's identity. It never
// performs authentication or authorization itself: the AdminAuthenticate
// middleware guarantees the identity is valid before this handler runs.
type AdminMeHandler struct{}

func NewAdminMeHandler() *AdminMeHandler {
	return &AdminMeHandler{}
}

// GetMe resolves the authenticated admin identity from the context (set by the
// AdminAuthenticate middleware) and returns the administrator's identifier and
// role. The role is always derived server-side from the configured identity,
// never from client input, so the frontend can never self-assign privileges.
func (h *AdminMeHandler) GetMe(c *gin.Context) {
	adminID, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	role, err := adminroles.RoleForAdminID(adminID)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Admin identity retrieved successfully.",
		"data": gin.H{
			"id":   adminID,
			"role": role,
		},
	})
}