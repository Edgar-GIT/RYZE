package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/models"
	"ryze/backend/services/commission_rules"
)

// AdminSettingsHandler reports the commercial parameters the server enforces.
// They come from the deployment configuration, so the console shows what is
// actually in force instead of a copy that can drift.
type AdminSettingsHandler struct {
	minPriceMinorUnits int64
	defaultCommission  uint32
	paymentMethods     func() []string
}

func NewAdminSettingsHandler(minPriceMinorUnits int64, defaultCommission uint32, paymentMethods func() []string) *AdminSettingsHandler {
	return &AdminSettingsHandler{minPriceMinorUnits: minPriceMinorUnits, defaultCommission: defaultCommission, paymentMethods: paymentMethods}
}

// GetCommerce returns currency, price floor, commission default and cap, and
// the payment methods currently enabled.
func (h *AdminSettingsHandler) GetCommerce(c *gin.Context) {
	methods := []string{}
	if h.paymentMethods != nil {
		methods = nonNil(h.paymentMethods())
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Commercial settings retrieved.",
		"data": gin.H{
			"currency":                        string(models.ProgramCurrencyEUR),
			"min_program_price_minor_units":   h.minPriceMinorUnits,
			"default_platform_commission_bps": h.defaultCommission,
			"max_commission_bps":              commission_rules.MaxCommissionBPS,
			"payment_methods":                 methods,
			// Fixed catalogue prices (owner decision, 2026-11-14): nobody sets
			// a price; trainer premium/personalized plans follow Level 1/2.
			"catalog_prices": gin.H{
				models.ProgramProductTypeGeneric:       models.CatalogPriceMinorUnits(models.ProgramProductTypeGeneric, models.ProgramTypeFree),
				models.ProgramProductTypePremiumLevel1: models.PremiumLevel1PriceMinorUnits,
				models.ProgramProductTypePremiumLevel2: models.PremiumLevel2PriceMinorUnits,
			},
		},
	})
}
