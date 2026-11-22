package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ryze/backend/email"
	"ryze/backend/logs"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/purchases"
	"ryze/backend/services/taxid"
	"ryze/backend/services/test_mode"
)

type testModePurchaseRequest struct {
	Email string `json:"email"`
	// TaxID is the optional taxpayer number (NIF) for the purchase document.
	TaxID string `json:"tax_id"`
}

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
// program as the persona currently in Test Mode. The required body carries the
// administrator-supplied address that receives the normal lifecycle emails for
// the rest of the session. A duplicate or an already owned program is reported
// as DUPLICATE_ENTITLEMENT.
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

	// ActivePersona checks the global kill switch before the token, so a
	// disabled service answers the documented error even without a cookie.
	rawToken, _ := c.Cookie(TestSessionTokenCookieName)
	session, err := h.testMode.ActivePersona(c.Request.Context(), rawToken)
	if err != nil {
		if errors.Is(err, test_mode.ErrDisabled) {
			RespondError(c, http.StatusForbidden, "TEST_MODE_DISABLED", "Test Mode is disabled.", nil)
			return
		}
		RespondError(c, http.StatusForbidden, "TEST_MODE_INACTIVE", "No active Test Mode session.", nil)
		return
	}

	if session.PersonaUserID != userID {
		RespondError(c, http.StatusForbidden, "TEST_MODE_INACTIVE", "No active Test Mode session.", nil)
		return
	}

	var req testModePurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil || !email.IsValidAddress(email.NormalizeAddress(req.Email)) {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "A valid email address is required.", nil)
		return
	}
	if strings.TrimSpace(req.TaxID) != "" {
		if _, ok := taxid.Normalize(req.TaxID); !ok {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid taxpayer number.", nil)
			return
		}
	}
	if err := h.testMode.SetRecipientEmail(c.Request.Context(), rawToken, req.Email); err != nil {
		h.respondRecipientError(c, err)
		return
	}

	purchase, err := h.purchases.CompleteTestPurchase(c.Request.Context(), userID, programID, purchases.CheckoutDetails{TaxID: req.TaxID})
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

func (h *TestModePurchaseHandler) respondRecipientError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, test_mode.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "A valid email address is required.", nil)
	case errors.Is(err, test_mode.ErrDisabled):
		RespondError(c, http.StatusForbidden, "TEST_MODE_DISABLED", "Test Mode is disabled.", nil)
	case errors.Is(err, test_mode.ErrNoActiveSession):
		RespondError(c, http.StatusForbidden, "TEST_MODE_INACTIVE", "No active Test Mode session.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}

func (h *TestModePurchaseHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, purchases.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, purchases.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	case errors.Is(err, purchases.ErrProgramNotPurchasable):
		RespondError(c, http.StatusConflict, "PROGRAM_NOT_PURCHASABLE", "Program is not purchasable.", nil)
	case errors.Is(err, purchases.ErrPrerequisiteNotMet):
		// The questionnaire gate applies to Test Mode exactly as it does to a
		// paid checkout. No purchase row was created.
		RespondError(c, http.StatusConflict, "PURCHASE_PREREQUISITE_NOT_MET", "This program requires additional information before it can be purchased.", nil)
	case errors.Is(err, purchases.ErrPremiumLevelConflict):
		RespondError(c, http.StatusConflict, "PREMIUM_LEVEL_CONFLICT", "You cannot hold Premium Level 1 and Premium Level 2 at the same time.", nil)
	case errors.Is(err, purchases.ErrDuplicateEntitlement):
		RespondError(c, http.StatusConflict, "DUPLICATE_ENTITLEMENT", "You already own this program.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
