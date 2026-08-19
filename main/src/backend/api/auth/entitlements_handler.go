package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/entitlements"
)

// entitlementProgramResponse is the safe program summary exposed inside
// entitlement metadata. It carries only public product metadata and never
// exposes the owning trainer, parent identifiers, deletion markers or any
// internal data.
type entitlementProgramResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// entitlementResponse is the safe representation of a purchase-backed right
// to access a program. It carries only public metadata and never exposes
// internal identifiers beyond the entitlement and program id.
type entitlementResponse struct {
	ID        string                     `json:"id"`
	ProgramID string                     `json:"program_id"`
	Program   entitlementProgramResponse `json:"program"`
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

func newEntitlementResponse(ent *entitlements.Entitlement) entitlementResponse {
	return entitlementResponse{
		ID:        ent.ID,
		ProgramID: ent.ProgramID,
		Program: entitlementProgramResponse{
			ID:          ent.Program.ID,
			Name:        ent.Program.Name,
			Description: ent.Program.Description,
			Type:        ent.Program.Type,
			Status:      ent.Program.Status,
			CreatedAt:   ent.Program.CreatedAt,
			UpdatedAt:   ent.Program.UpdatedAt,
		},
		CreatedAt: ent.CreatedAt,
		UpdatedAt: ent.UpdatedAt,
	}
}

// EntitlementsHandler exposes the authenticated client's entitlement read
// operation. It never performs authentication or authorization itself: those
// are enforced by the Authenticate middleware mounted on the route. The user
// identity always comes exclusively from the authentication context; query
// parameters, body, headers or any client-supplied identity can never influence
// which entitlements are returned.
type EntitlementsHandler struct {
	service entitlements.Service
}

func NewEntitlementsHandler(svc entitlements.Service) *EntitlementsHandler {
	return &EntitlementsHandler{service: svc}
}

// ListEntitlements returns the safe entitlement metadata for every active
// entitlement held by the authenticated user. The identity comes exclusively
// from the authentication context.
func (h *EntitlementsHandler) ListEntitlements(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	ents, err := h.service.ListEntitlements(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	response := make([]entitlementResponse, 0, len(ents))
	for i := range ents {
		response = append(response, newEntitlementResponse(&ents[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Entitlements retrieved successfully.",
		"data":    response,
	})
}

func (h *EntitlementsHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, entitlements.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
