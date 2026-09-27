package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/payments"
)

// paymentMethodResponse is the safe public representation of one available
// payment method. It carries only the method identifier and its display label.
type paymentMethodResponse struct {
	Method string `json:"method"`
	Label  string `json:"label"`
}

// PaymentMethodsHandler exposes the payment methods the backend currently has a
// configured provider for. Availability is derived server-side: a method is
// advertised only when its provider was configured at startup, so the frontend
// never offers a payment method that would fail at initiation.
type PaymentMethodsHandler struct {
	available []paymentMethodResponse
}

// NewPaymentMethodsHandler wires the handler to the startup method-to-provider
// mapping. The available set is fixed at startup and never changes at runtime.
func NewPaymentMethodsHandler(methodMap *payments.MethodProviderMap) *PaymentMethodsHandler {
	labels := map[payments.PaymentMethod]string{
		payments.PaymentMethodCard:   "Card",
		payments.PaymentMethodMBWay:  "MB WAY",
		payments.PaymentMethodPayPal: "PayPal",
	}

	methods := methodMap.AvailableMethods()
	available := make([]paymentMethodResponse, 0, len(methods))
	for _, method := range methods {
		available = append(available, paymentMethodResponse{
			Method: string(method),
			Label:  labels[method],
		})
	}
	return &PaymentMethodsHandler{available: available}
}

// List returns the currently available payment methods. The response never
// reveals secrets or disabled providers.
func (h *PaymentMethodsHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Payment methods retrieved successfully.",
		"data":    h.available,
	})
}
