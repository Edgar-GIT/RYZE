package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/services/invoices"
)

// InvoiceDocumentType names the document RYZE issues today: a purchase
// receipt with no tax value (until certified invoicing software is connected).
const InvoiceDocumentType = "purchase_receipt"

// invoiceResponse is the purchase document as the client and the commerce
// administrators see it: the issued snapshot, never the owner's account id.
type invoiceResponse struct {
	models.Invoice
	DocumentType string `json:"document_type"`
	FiscalValue  bool   `json:"fiscal_value"`
}

func newInvoiceResponse(invoice *models.Invoice) invoiceResponse {
	return invoiceResponse{Invoice: *invoice, DocumentType: InvoiceDocumentType, FiscalValue: false}
}

// InvoiceHandler serves purchase documents. The owner is always the
// authenticated user; another account's document reads as not found.
type InvoiceHandler struct {
	service invoices.Service
}

func NewInvoiceHandler(service invoices.Service) *InvoiceHandler {
	return &InvoiceHandler{service: service}
}

// List returns the authenticated user's documents, newest first.
func (h *InvoiceHandler) List(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	list, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	data := make([]invoiceResponse, 0, len(list))
	for index := range list {
		data = append(data, newInvoiceResponse(&list[index]))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Invoices retrieved successfully.", "data": data})
}

// ForPurchase returns the document of one of the authenticated user's
// purchases (issued on demand if it is completed and the document is missing).
func (h *InvoiceHandler) ForPurchase(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	invoice, err := h.service.ForPurchase(c.Request.Context(), userID, c.Param("purchaseID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Invoice retrieved successfully.", "data": newInvoiceResponse(invoice)})
}

// ForPurchaseAdmin returns any purchase's document. The route requires the
// commerce administration permission.
func (h *InvoiceHandler) ForPurchaseAdmin(c *gin.Context) {
	invoice, err := h.service.ForPurchaseAdmin(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Invoice retrieved successfully.", "data": newInvoiceResponse(invoice)})
}

func (h *InvoiceHandler) respondError(c *gin.Context, err error) {
	if errors.Is(err, invoices.ErrNotFound) {
		RespondError(c, http.StatusNotFound, "INVOICE_NOT_FOUND", "Invoice not found.", nil)
		return
	}
	slog.Error("invoice request failed", slog.String("error", err.Error()))
	RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
}
