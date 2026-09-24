package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/logs"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/purchases"
	"ryze/backend/services/test_mode"
)

// TestModePurchaseHandler provisions completed test purchases for a persona
// inside an active Test Mode session. It never accepts an arbitrary user: the
// persona identity must match the authenticated user, which is guaranteed
// server-side because only the enter flow can mint a persona access token.
type TestModePurchaseHandler struct {
	testMode  test_mode.Service
	purchases purchases.Service
}

func NewTestModePurchaseHandler(svc test_mode.Service, purchasesSvc purchases.Service) *TestModePurchaseHandler {
	return &TestModePurchaseHandler{testMode: svc, purchases: purchasesSvc}
}

// Purchase completes a zero-price purchase and its entitlement for the given
// program as the persona currently in Test Mode. A duplicate or an already
// owned program is reported as DUPLICATE_ENTITLEMENT.
func (h *TestModePurchaseHandler) Purchase(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	programID := c.Param("programID")
	if programID == "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	rawToken, err := c.Cookie(TestSessionTokenCookieName)
	if err != nil {
		RespondError(c, http.StatusForbidden, "TEST_MODE_INACTIVE", "No active Test Mode session.", nil)
		return
	}

	session, err := h.testMode.ActivePersona(c.Request.Context(), rawToken)
	if err != nil {
		RespondError(c, http.StatusForbidden, "TEST_MODE_INACTIVE", "No active Test Mode session.", nil)
		return
	}

	if session.PersonaUserID != userID {
		RespondError(c, http.StatusForbidden, "TEST_MODE_INACTIVE", "No active Test Mode session.", nil)
		return
	}

	purchase, err := h.purchases.CompleteTestPurchase(c.Request.Context(), userID, programID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	logs.Info("test_mode.purchase_completed",
		slog.String("persona", session.Persona),
		slog.String("persona_user_id", userID),
		slog.String("program_id", programID),
	)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Test purchase completed successfully.",
		"data":    newPurchaseResponse(purchase),
	})
}

func (h *TestModePurchaseHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, purchases.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, purchases.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	case errors.Is(err, purchases.ErrProgramNotPurchasable):
		RespondError(c, http.StatusConflict, "PROGRAM_NOT_PURCHASABLE", "Program is not purchasable.", nil)
	case errors.Is(err, purchases.ErrDuplicateEntitlement):
		RespondError(c, http.StatusConflict, "DUPLICATE_ENTITLEMENT", "You already own this program.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
