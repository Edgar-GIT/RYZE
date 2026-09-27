package auth

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/admin_commerce"
)

// adminCustomerResponse is the safe buyer identity exposed to administrators.
// It carries the display name and contact email only.
type adminCustomerResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// adminProgramResponse is the safe program summary exposed inside admin
// commerce views. It carries public product metadata plus a deleted flag so a
// retired program keeps its historical identity.
type adminProgramResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Status           string    `json:"status"`
	Level            *string   `json:"level"`
	DurationWeeks    *int      `json:"duration_weeks"`
	FrequencyPerWeek *int      `json:"frequency_per_week"`
	TrainingType     *string   `json:"training_type"`
	PriceMinorUnits  int64     `json:"price_minor_units"`
	Currency         string    `json:"currency"`
	Deleted          bool      `json:"deleted"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// adminPurchaseResponse is the safe representation of one purchase record in
// the admin commerce view. Commission and payout data, the owning user id and
// all internal identifiers are never exposed.
type adminPurchaseResponse struct {
	ID              string                `json:"id"`
	ProgramID       string                `json:"program_id"`
	PriceMinorUnits int64                 `json:"price_minor_units"`
	Currency        string                `json:"currency"`
	Status          string                `json:"status"`
	Test            bool                  `json:"test"`
	Access          bool                  `json:"access"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Customer        adminCustomerResponse `json:"customer"`
	Program         adminProgramResponse  `json:"program"`
}

// adminSalesSummaryResponse aggregates the commerce dashboard metrics. Test
// purchases are reported separately and are never part of real revenue.
type adminSalesSummaryResponse struct {
	CompletedCount        int64 `json:"completed_count"`
	PendingCount          int64 `json:"pending_count"`
	FailedCount           int64 `json:"failed_count"`
	RealRevenueMinorUnits int64 `json:"real_revenue_minor_units"`
	TestPurchaseCount     int64 `json:"test_purchase_count"`
	ProgramsWithSales     int64 `json:"programs_with_sales"`
}

// adminProgramSaleResponse aggregates the purchase counts and revenue of a
// single program. RealSales is the completed non-test purchase count: Test
// Mode purchases are reported separately and never inflate the real sales
// figure, while CompletedSales keeps the v2.75 counting including test
// purchases for backward compatibility.
type adminProgramSaleResponse struct {
	Program           adminProgramResponse `json:"program"`
	CompletedSales    int64                `json:"completed_sales"`
	RealSales         int64                `json:"real_sales"`
	RevenueMinorUnits int64                `json:"revenue_minor_units"`
	TestPurchases     int64                `json:"test_purchases"`
	PendingPurchases  int64                `json:"pending_purchases"`
	FailedPurchases   int64                `json:"failed_purchases"`
}

func newAdminCustomerResponse(c *admin_commerce.Customer) adminCustomerResponse {
	return adminCustomerResponse{Name: c.Name, Email: c.Email}
}

func newAdminProgramResponse(p *admin_commerce.Program) adminProgramResponse {
	return adminProgramResponse{
		ID:               p.ID,
		Name:             p.Name,
		Type:             p.Type,
		Status:           p.Status,
		Level:            p.Level,
		DurationWeeks:    p.DurationWeeks,
		FrequencyPerWeek: p.FrequencyPerWeek,
		TrainingType:     p.TrainingType,
		PriceMinorUnits:  p.PriceMinorUnits,
		Currency:         p.Currency,
		Deleted:          p.Deleted,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

func newAdminPurchaseResponse(p *admin_commerce.Purchase) adminPurchaseResponse {
	return adminPurchaseResponse{
		ID:              p.ID,
		ProgramID:       p.ProgramID,
		PriceMinorUnits: p.PriceMinorUnits,
		Currency:        p.Currency,
		Status:          p.Status,
		Test:            p.Test,
		Access:          p.Access,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
		Customer:        newAdminCustomerResponse(&p.Customer),
		Program:         newAdminProgramResponse(&p.Program),
	}
}

func newAdminSalesSummaryResponse(s admin_commerce.SalesSummary) adminSalesSummaryResponse {
	return adminSalesSummaryResponse{
		CompletedCount:        s.CompletedCount,
		PendingCount:          s.PendingCount,
		FailedCount:           s.FailedCount,
		RealRevenueMinorUnits: s.RealRevenueMinorUnits,
		TestPurchaseCount:     s.TestPurchaseCount,
		ProgramsWithSales:     s.ProgramsWithSales,
	}
}

func newAdminProgramSaleResponse(s *admin_commerce.ProgramSale) adminProgramSaleResponse {
	return adminProgramSaleResponse{
		Program:           newAdminProgramResponse(&s.Program),
		CompletedSales:    s.CompletedSales,
		RealSales:         s.RealSales,
		RevenueMinorUnits: s.RevenueMinorUnits,
		TestPurchases:     s.TestPurchases,
		PendingPurchases:  s.PendingPurchases,
		FailedPurchases:   s.FailedPurchases,
	}
}

// AdminCommerceHandler exposes the administrator commerce view over the
// platform's purchases. It never performs authentication or authorization
// itself: those are enforced by the AdminAuthenticate and
// RequireAdminPermission middleware mounted on the routes. Commerce is a
// Management-Administrator responsibility.
type AdminCommerceHandler struct {
	service admin_commerce.Service
}

func NewAdminCommerceHandler(svc admin_commerce.Service) *AdminCommerceHandler {
	return &AdminCommerceHandler{service: svc}
}

// ListPurchases returns one page of purchases matching the optional filters,
// newest first, together with the filtered sales summary and pagination
// metadata.
func (h *AdminCommerceHandler) ListPurchases(c *gin.Context) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	filter, ok := h.parseFilters(c)
	if !ok {
		return
	}

	result, err := h.service.ListPurchases(c.Request.Context(), filter, page, limit)
	if err != nil {
		h.respondError(c, err)
		return
	}

	purchases := make([]adminPurchaseResponse, 0, len(result.Purchases))
	for i := range result.Purchases {
		purchases = append(purchases, newAdminPurchaseResponse(&result.Purchases[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Purchases retrieved successfully.",
		"data": gin.H{
			"purchases": purchases,
			"summary":   newAdminSalesSummaryResponse(result.Summary),
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// GetPurchase returns the safe representation of one purchase by its id,
// including the buyer identity and program summary.
func (h *AdminCommerceHandler) GetPurchase(c *gin.Context) {
	purchase, err := h.service.GetPurchase(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Purchase retrieved successfully.",
		"data":    gin.H{"purchase": newAdminPurchaseResponse(purchase)},
	})
}

// ListProgramSales returns one page of per-program sales matching the optional
// filters, most completed sales first, with pagination metadata.
func (h *AdminCommerceHandler) ListProgramSales(c *gin.Context) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	filter, ok := h.parseFilters(c)
	if !ok {
		return
	}

	result, err := h.service.ListProgramSales(c.Request.Context(), filter, page, limit)
	if err != nil {
		h.respondError(c, err)
		return
	}

	sales := make([]adminProgramSaleResponse, 0, len(result.Sales))
	for i := range result.Sales {
		sales = append(sales, newAdminProgramSaleResponse(&result.Sales[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program sales retrieved successfully.",
		"data": gin.H{
			"sales": sales,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// parseFilters builds the commerce filter from the optional query parameters.
// program_id must be valid, program_ids is a comma-separated batch of program
// ids, status must be valid, test must parse as a boolean and the from/to dates
// must be in YYYY-MM-DD form. Validation failures respond directly; program id
// syntax is validated by the service before any database access.
func (h *AdminCommerceHandler) parseFilters(c *gin.Context) (admin_commerce.PurchaseFilter, bool) {
	filter := admin_commerce.PurchaseFilter{
		ProgramID: c.Query("program_id"),
		Status:    c.Query("status"),
	}

	if value := c.Query("program_ids"); value != "" {
		parts := strings.Split(value, ",")
		ids := make([]string, 0, len(parts))
		for _, part := range parts {
			if id := strings.TrimSpace(part); id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			filter.ProgramIDs = ids
		}
	}

	if value := c.Query("test"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
			return filter, false
		}
		filter.Test = &parsed
	}

	if value := c.Query("from"); value != "" {
		date, err := parseCommerceDate(value)
		if err != nil {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
			return filter, false
		}
		filter.CreatedFrom = &date
	}

	if value := c.Query("to"); value != "" {
		// The upper bound is exclusive: to = YYYY-MM-DD includes the whole day.
		date, err := parseCommerceDate(value)
		if err != nil {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
			return filter, false
		}
		date = date.Add(24 * time.Hour)
		filter.CreatedTo = &date
	}

	return filter, true
}

// parseCommerceDate parses a YYYY-MM-DD date as the start of its day in UTC.
func parseCommerceDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

// respondError maps admin commerce service errors to API responses. Internal
// error details are never exposed to the client.
func (h *AdminCommerceHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, admin_commerce.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, admin_commerce.ErrPurchaseNotFound):
		RespondError(c, http.StatusNotFound, "PURCHASE_NOT_FOUND", "Purchase not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
