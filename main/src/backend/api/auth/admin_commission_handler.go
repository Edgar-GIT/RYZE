package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/commission_rules"
)

// adminCommissionRuleResponse is the safe commission rule representation
// returned by the admin commission endpoints. It carries only public metadata
// and never exposes internal data.
type adminCommissionRuleResponse struct {
	ID            string     `json:"id"`
	TrainerID     string     `json:"trainer_id"`
	CommissionBPS uint32     `json:"commission_bps"`
	ValidFrom     time.Time  `json:"valid_from"`
	ValidUntil    *time.Time `json:"valid_until"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func newAdminCommissionRuleResponse(rule *commission_rules.CommissionRule) adminCommissionRuleResponse {
	return adminCommissionRuleResponse{
		ID:            rule.ID,
		TrainerID:     rule.TrainerID,
		CommissionBPS: rule.CommissionBPS,
		ValidFrom:     rule.ValidFrom,
		ValidUntil:    rule.ValidUntil,
		CreatedAt:     rule.CreatedAt,
		UpdatedAt:     rule.UpdatedAt,
	}
}

// adminCommissionResolutionResponse is the safe commission resolution
// representation returned by the admin commission resolution endpoint.
type adminCommissionResolutionResponse struct {
	CommissionBPS uint32 `json:"commission_bps"`
	IsOverride    bool   `json:"is_override"`
}

// upsertCommissionRuleRequest is the request DTO for
// PATCH /api/v1/admin/trainers/:trainerID/commission.
type upsertCommissionRuleRequest struct {
	CommissionBPS uint32 `json:"commission_bps"`
}

// AdminCommissionHandler exposes the administrator commission rule operations.
// It never performs authentication or authorization itself: those are enforced
// by the AdminAuthenticate and RequireAdminPermission middleware mounted on the
// route.
type AdminCommissionHandler struct {
	service commission_rules.Service
}

func NewAdminCommissionHandler(svc commission_rules.Service) *AdminCommissionHandler {
	return &AdminCommissionHandler{service: svc}
}

// GetCommissionRule returns the active commission rule for a given trainer. A
// missing rule maps to a 404 response.
func (h *AdminCommissionHandler) GetCommissionRule(c *gin.Context) {
	rule, err := h.service.GetCommissionRule(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Commission rule retrieved successfully.",
		"data":    newAdminCommissionRuleResponse(rule),
	})
}

// UpsertCommissionRule creates or replaces the active commission rule for a
// given trainer.
func (h *AdminCommissionHandler) UpsertCommissionRule(c *gin.Context) {
	var req upsertCommissionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	rule, err := h.service.UpsertCommissionRule(c.Request.Context(), c.Param("id"), req.CommissionBPS)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Commission rule updated successfully.",
		"data":    newAdminCommissionRuleResponse(rule),
	})
}

// DeleteCommissionRule removes the active commission rule for a given trainer.
// The trainer falls back to the global default commission after deletion.
func (h *AdminCommissionHandler) DeleteCommissionRule(c *gin.Context) {
	if err := h.service.DeleteCommissionRule(c.Request.Context(), c.Param("id")); err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Commission rule deleted successfully.",
		"data":    nil,
	})
}

// GetCommissionResolution returns the resolved commission for a given trainer,
// whether from a trainer-specific override or the global default.
func (h *AdminCommissionHandler) GetCommissionResolution(c *gin.Context) {
	resolution, err := h.service.ResolveCommission(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Commission resolution retrieved successfully.",
		"data": adminCommissionResolutionResponse{
			CommissionBPS: resolution.CommissionBPS,
			IsOverride:    resolution.IsOverride,
		},
	})
}

// respondError maps commission rule service errors to API responses. Internal
// error details are never exposed to the client.
func (h *AdminCommissionHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, commission_rules.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid commission rule input.", nil)
	case errors.Is(err, commission_rules.ErrTrainerNotFound):
		RespondError(c, http.StatusNotFound, "TRAINER_NOT_FOUND", "Trainer not found.", nil)
	case errors.Is(err, commission_rules.ErrCommissionRuleNotFound):
		RespondError(c, http.StatusNotFound, "COMMISSION_RULE_NOT_FOUND", "Commission rule not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
