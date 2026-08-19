package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/admin_program_pricing"
)

// adminProgramPricingResponse is the safe program representation returned by
// the admin pricing endpoint. It carries only public product metadata and
// never exposes internal data.
type adminProgramPricingResponse struct {
	ID              string    `json:"id"`
	TrainerID       string    `json:"trainer_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	PriceMinorUnits int64     `json:"price_minor_units"`
	Currency        string    `json:"currency"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func newAdminProgramPricingResponse(program *admin_program_pricing.Program) adminProgramPricingResponse {
	return adminProgramPricingResponse{
		ID:              program.ID,
		TrainerID:       program.TrainerID,
		Name:            program.Name,
		Description:     program.Description,
		Type:            program.Type,
		Status:          program.Status,
		PriceMinorUnits: program.PriceMinorUnits,
		Currency:        program.Currency,
		CreatedAt:       program.CreatedAt,
		UpdatedAt:       program.UpdatedAt,
	}
}

// updatePricingRequest is the request DTO for PATCH /api/v1/admin/programs/:programID/pricing.
type updatePricingRequest struct {
	PriceMinorUnits int64  `json:"price_minor_units"`
	Currency        string `json:"currency"`
}

// AdminProgramPricingHandler exposes the administrator program pricing
// operations. It never performs authentication or authorization itself: those
// are enforced by the AdminAuthenticate and RequireAdminPermission middleware
// mounted on the route.
type AdminProgramPricingHandler struct {
	service admin_program_pricing.Service
}

func NewAdminProgramPricingHandler(svc admin_program_pricing.Service) *AdminProgramPricingHandler {
	return &AdminProgramPricingHandler{service: svc}
}

// GetProgram returns one active program by its id. The program id in the path
// only identifies the requested resource; there is no ownership scoping.
func (h *AdminProgramPricingHandler) GetProgram(c *gin.Context) {
	program, err := h.service.GetProgram(c.Request.Context(), c.Param("programID"))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program retrieved successfully.",
		"data":    newAdminProgramPricingResponse(program),
	})
}

// UpdatePricing updates the price of one active program. The program id in the
// path only identifies the requested resource; there is no ownership scoping.
func (h *AdminProgramPricingHandler) UpdatePricing(c *gin.Context) {
	var req updatePricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	program, err := h.service.UpdatePricing(c.Request.Context(), c.Param("programID"), admin_program_pricing.UpdatePricingInput{
		PriceMinorUnits: req.PriceMinorUnits,
		Currency:        req.Currency,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Program pricing updated successfully.",
		"data":    newAdminProgramPricingResponse(program),
	})
}

// respondError maps admin program pricing service errors to API responses.
// Internal error details are never exposed to the client.
func (h *AdminProgramPricingHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, admin_program_pricing.ErrProgramNotFound):
		RespondError(c, http.StatusNotFound, "PROGRAM_NOT_FOUND", "Program not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
