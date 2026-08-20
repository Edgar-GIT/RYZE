package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/purchases"
)

// purchaseResponse is the safe representation of a pending purchase transaction.
// It carries only public commercial metadata and never exposes internal data
// beyond the purchase and program id.
type purchaseResponse struct {
	ID              string `json:"id"`
	UserID          string `json:"-"`
	ProgramID       string `json:"program_id"`
	PriceMinorUnits int64  `json:"price_minor_units"`
	Currency        string `json:"currency"`
	CommissionBPS   uint32 `json:"commission_bps"`
	PlatformAmount  int64  `json:"platform_amount"`
	TrainerAmount   int64  `json:"trainer_amount"`
	Status          string `json:"status"`
}

// paymentInitiationResponse is the safe representation of a payment initiation
// result. It carries only public payment metadata required by the client to
// redirect the user when a checkout URL is present.
type paymentInitiationResponse struct {
	PaymentID   string `json:"payment_id"`
	CheckoutURL string `json:"checkout_url,omitempty"`
	Status      string `json:"status"`
	PurchaseID  string `json:"purchase_id"`
}

func newPurchaseResponse(p *purchases.Purchase) purchaseResponse {
	return purchaseResponse{
		ID:              p.ID,
		ProgramID:       p.ProgramID,
		PriceMinorUnits: p.PriceMinorUnits,
		Currency:        p.Currency,
		CommissionBPS:   p.CommissionBPS,
		PlatformAmount:  p.PlatformAmount,
		TrainerAmount:   p.TrainerAmount,
		Status:          p.Status,
	}
}

// PurchaseHandler exposes the authenticated client's purchase creation
// operation. It never performs authentication or authorization itself: those
// are enforced by the Authenticate middleware mounted on the route. The user
// identity always comes exclusively from the authentication context; query
// parameters, body, headers or any client-supplied identity can never influence
// which purchase is created.
type PurchaseHandler struct {
	service purchases.Service
}

func NewPurchaseHandler(svc purchases.Service) *PurchaseHandler {
	return &PurchaseHandler{service: svc}
}

// CreatePurchase validates the program, snapshots the current price, resolves
// the applicable commission, and persists a pending purchase record. The user
// identity comes exclusively from the authentication context.
func (h *PurchaseHandler) CreatePurchase(c *gin.Context) {
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

	purchase, err := h.service.CreatePurchaseIntent(c.Request.Context(), userID, programID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Purchase intent created successfully.",
		"data":    newPurchaseResponse(purchase),
	})
}

// respondError maps purchase service errors to API responses. Internal error
// details are never exposed to the client.
func (h *PurchaseHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, purchases.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, purchases.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	case errors.Is(err, purchases.ErrProgramNotPurchasable):
		RespondError(c, http.StatusConflict, "PROGRAM_NOT_PURCHASABLE", "This program cannot be purchased.", nil)
	case errors.Is(err, purchases.ErrDuplicateEntitlement):
		RespondError(c, http.StatusConflict, "DUPLICATE_ENTITLEMENT", "You already own this program.", nil)
	case errors.Is(err, purchases.ErrDuplicatePurchase):
		RespondError(c, http.StatusConflict, "DUPLICATE_PURCHASE", "A purchase for this program is already in progress.", nil)
	case errors.Is(err, purchases.ErrPurchaseNotFound):
		RespondError(c, http.StatusNotFound, "PURCHASE_NOT_FOUND", "Purchase not found.", nil)
	case errors.Is(err, purchases.ErrPurchaseNotPending):
		RespondError(c, http.StatusConflict, "PURCHASE_NOT_PENDING", "This purchase is not pending.", nil)
	case errors.Is(err, purchases.ErrPaymentProvider):
		RespondError(c, http.StatusBadGateway, "PAYMENT_PROVIDER_ERROR", "Payment provider unavailable. Please try again later.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}

// InitiatePayment requests a provider payment for an existing pending purchase.
// The purchase must belong to the authenticated user and be in pending status.
// The immutable purchase snapshot is used to construct the provider request; no
// client-supplied commercial values are accepted. The client must provide a
// valid payment method in the request body. The purchase status is NOT modified
// during initiation.
func (h *PurchaseHandler) InitiatePayment(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	purchaseID := c.Param("purchaseID")
	if purchaseID == "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	var body struct {
		PaymentMethod string `json:"payment_method"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	if body.PaymentMethod == "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	result, err := h.service.InitiatePayment(c.Request.Context(), userID, purchaseID, body.PaymentMethod)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Payment initiated successfully.",
		"data": paymentInitiationResponse{
			PaymentID:   result.PaymentID,
			CheckoutURL: result.CheckoutURL,
			Status:      string(result.Status),
			PurchaseID:  result.PurchaseID,
		},
	})
}
