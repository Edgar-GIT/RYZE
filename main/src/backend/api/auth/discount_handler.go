package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/discounts"
)

// discountErrorReason maps a discount refusal to the reason the client shows
// ("" when err is not one).
func discountErrorReason(err error) string {
	switch {
	case errors.Is(err, discounts.ErrNotFound):
		return "not_found"
	case errors.Is(err, discounts.ErrExpired):
		return "expired"
	case errors.Is(err, discounts.ErrNotApplicable):
		return "not_applicable"
	case errors.Is(err, discounts.ErrUsedUp):
		return "used_up"
	case errors.Is(err, discounts.ErrAlreadyUsed):
		return "already_used"
	case errors.Is(err, discounts.ErrLocked):
		return "locked"
	}
	return ""
}

// ProgramReader finds a published plan (to preview a code against it).
type ProgramReader interface {
	FindPublishedByID(ctx context.Context, programID string) (*models.Program, error)
}

// DiscountHandler: administrators create, list and disable discount codes;
// buyers preview a code at checkout (v2.131.0).
type DiscountHandler struct {
	service  *discounts.Service
	programs ProgramReader
}

func NewDiscountHandler(service *discounts.Service, programs ProgramReader) *DiscountHandler {
	return &DiscountHandler{service: service, programs: programs}
}

type discountCodeResponse struct {
	ID         string     `json:"id"`
	Code       string     `json:"code"`
	PercentOff int        `json:"percent_off"`
	AppliesTo  string     `json:"applies_to"`
	StartsAt   time.Time  `json:"starts_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	MaxUses    *int       `json:"max_uses"`
	Uses       int64      `json:"uses"`
	DisabledAt *time.Time `json:"disabled_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func newDiscountCodeResponse(code models.DiscountCode, uses int64) discountCodeResponse {
	return discountCodeResponse{
		ID: code.ID, Code: code.Code, PercentOff: code.PercentOff, AppliesTo: code.AppliesTo, StartsAt: code.StartsAt,
		ExpiresAt: code.ExpiresAt, MaxUses: code.MaxUses, Uses: uses, DisabledAt: code.DisabledAt, CreatedAt: code.CreatedAt,
	}
}

type createDiscountRequest struct {
	Code          string `json:"code"`
	PercentOff    int    `json:"percent_off"`
	AppliesTo     string `json:"applies_to"`
	DurationHours int    `json:"duration_hours"`
	MaxUses       *int   `json:"max_uses"`
}

func (h *DiscountHandler) Create(c *gin.Context) {
	adminID, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var body createDiscountRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	created, err := h.service.Create(c.Request.Context(), adminID, discounts.CreateInput{
		Code: body.Code, PercentOff: body.PercentOff, AppliesTo: body.AppliesTo, DurationHours: body.DurationHours, MaxUses: body.MaxUses,
	})
	switch {
	case errors.Is(err, discounts.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Codes have 3 to 32 capital letters, digits, - or _; 1% to 90% off; usable 1 hour to 30 days.", nil)
	case errors.Is(err, discounts.ErrCodeTaken):
		RespondError(c, http.StatusConflict, "DISCOUNT_CODE_TAKEN", "This code already exists.", nil)
	case err != nil:
		slog.Error("discount code creation failed", slog.String("error", err.Error()))
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	default:
		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Discount code created.", "data": newDiscountCodeResponse(*created, 0)})
	}
}

func (h *DiscountHandler) List(c *gin.Context) {
	rows, err := h.service.List(c.Request.Context())
	if err != nil {
		slog.Error("discount code list failed", slog.String("error", err.Error()))
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	codes := make([]discountCodeResponse, 0, len(rows))
	for _, row := range rows {
		codes = append(codes, newDiscountCodeResponse(row.DiscountCode, row.Uses))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Discount codes.", "data": gin.H{"codes": codes}})
}

func (h *DiscountHandler) Disable(c *gin.Context) {
	if err := h.service.Disable(c.Request.Context(), c.Param("id")); err != nil {
		if errors.Is(err, discounts.ErrNotFound) {
			RespondError(c, http.StatusNotFound, "DISCOUNT_CODE_NOT_FOUND", "Discount code not found.", nil)
			return
		}
		slog.Error("discount code disable failed", slog.String("error", err.Error()))
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Discount code disabled.", "data": nil})
}

// Preview checks a code against a plan for the signed-in buyer:
// GET /me/discount-codes/preview?code=...&program_id=...
func (h *DiscountHandler) Preview(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	program, err := h.programs.FindPublishedByID(c.Request.Context(), c.Query("program_id"))
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	var trainerID *string
	if program.TrainerID != "" {
		trainerID = &program.TrainerID
	}
	preview, err := h.service.PreviewFor(c.Request.Context(), userID, c.Query("code"),
		&repositories.DiscountTarget{ProductType: program.ProductType, TrainerID: trainerID}, program.PriceMinorUnits)
	if reason := discountErrorReason(err); reason != "" {
		RespondError(c, http.StatusConflict, "DISCOUNT_INVALID", "The discount code cannot be used.", []string{reason})
		return
	}
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Discount code valid.", "data": gin.H{
		"code": preview.Code, "percent_off": preview.PercentOff, "discount_minor_units": preview.DiscountMinor,
		"total_minor_units": preview.TotalMinor, "currency": program.Currency, "expires_at": preview.ExpiresAt,
		"original_minor_units": program.PriceMinorUnits,
	}})
}
