package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/trainercontext"
	"ryze/backend/services/trainer_clients"
)

// trainerClientResponse only exposes safe client relationship information: the
// relationship identity, the owning trainer id, the linked user's public
// profile data and the lifecycle timestamps. Password hashes, session versions
// and deletion markers are never exposed.
type trainerClientResponse struct {
	ID        string       `json:"id"`
	TrainerID string       `json:"trainer_id"`
	UserID    string       `json:"user_id"`
	User      userResponse `json:"user"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func newTrainerClientResponse(client *trainer_clients.Client) trainerClientResponse {
	return trainerClientResponse{
		ID:        client.RelationID,
		TrainerID: client.TrainerID,
		UserID:    client.UserID,
		User: userResponse{
			ID:        client.UserID,
			Email:     client.Email,
			FirstName: client.FirstName,
			LastName:  client.LastName,
			CreatedAt: client.UserCreatedAt,
			UpdatedAt: client.UserUpdatedAt,
		},
		CreatedAt: client.RelationCreatedAt,
		UpdatedAt: client.RelationUpdatedAt,
	}
}

// addClientRequest is the request DTO for POST /api/v1/trainer/clients. It only
// carries the client user id; the trainer identity never comes from the body and
// is always resolved from the authenticated trainer context.
type addClientRequest struct {
	UserID string `json:"user_id"`
}

// TrainerClientsHandler exposes the authenticated trainer's client management
// operations. It never performs authentication or authorization itself: those
// are enforced by the Authenticate, TrainerAuthenticate and
// RequireTrainerPermission middleware mounted on the route. The trainer
// identity always comes exclusively from the trainer context; request
// parameters, body, headers or any client-supplied identity can never
// influence which relationship is created, listed, removed or reactivated.
type TrainerClientsHandler struct {
	service trainer_clients.Service
}

func NewTrainerClientsHandler(svc trainer_clients.Service) *TrainerClientsHandler {
	return &TrainerClientsHandler{service: svc}
}

// ListClients returns one page of the authenticated trainer's active clients.
// A client-supplied trainer_id query parameter is deliberately ignored: the
// list is always scoped to the trainer identity from the context.
func (h *TrainerClientsHandler) ListClients(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

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

	result, err := h.service.ListClients(c.Request.Context(), trainerID, page, limit)
	if err != nil {
		h.respondClientsError(c, err)
		return
	}

	clients := make([]trainerClientResponse, 0, len(result.Clients))
	for i := range result.Clients {
		clients = append(clients, newTrainerClientResponse(&result.Clients[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Clients retrieved successfully.",
		"data": gin.H{
			"clients": clients,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

// GetClient returns the safe profile of one of the authenticated trainer's
// active clients. The trainer identity always comes from the trainer context;
// the user id in the path only identifies the requested resource and never
// proves access. Only an active trainer→client relationship grants access, so
// a user that is not the trainer's client, does not exist or is soft-deleted
// is never revealed.
func (h *TrainerClientsHandler) GetClient(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	client, err := h.service.GetClient(c.Request.Context(), trainerID, c.Param("userID"))
	if err != nil {
		h.respondClientsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client profile retrieved successfully.",
		"data":    newTrainerClientResponse(client),
	})
}

// AddClient creates the active relationship between the authenticated trainer
// and the requested active user. Only the client user_id is accepted from the
// request; the trainer identity is the authenticated one and can never be
// supplied by the client.
func (h *TrainerClientsHandler) AddClient(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	var req addClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	client, err := h.service.AddClient(c.Request.Context(), trainerID, req.UserID)
	if err != nil {
		h.respondClientsError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Client added successfully.",
		"data":    newTrainerClientResponse(client),
	})
}

// RemoveClient soft-deletes ONLY the relationship between the authenticated
// trainer and the user identified by the path parameter. The user account and
// the trainer profile are never deleted.
func (h *TrainerClientsHandler) RemoveClient(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	err := h.service.RemoveClient(c.Request.Context(), trainerID, c.Param("userID"))
	if err != nil {
		h.respondClientsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client removed successfully.",
		"data":    gin.H{},
	})
}

// ReactivateClient restores the exact same previously removed relationship row
// between the authenticated trainer and the user identified by the path
// parameter.
func (h *TrainerClientsHandler) ReactivateClient(c *gin.Context) {
	trainerID, ok := requireTrainerContext(c)
	if !ok {
		return
	}

	client, err := h.service.ReactivateClient(c.Request.Context(), trainerID, c.Param("userID"))
	if err != nil {
		h.respondClientsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Client relationship reactivated successfully.",
		"data":    newTrainerClientResponse(client),
	})
}

// requireTrainerContext resolves the authenticated trainer identity from the
// trainer context. When the context is missing or malformed the request is
// rejected and false is returned; the caller must abort.
func requireTrainerContext(c *gin.Context) (string, bool) {
	identity, err := trainercontext.IdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return "", false
	}
	return identity.TrainerID, true
}

// respondClientsError maps trainer-clients service errors to API responses.
// Internal error details are never exposed to the client.
func (h *TrainerClientsHandler) respondClientsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, trainer_clients.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, trainer_clients.ErrClientNotFound):
		RespondError(c, http.StatusNotFound, "CLIENT_NOT_FOUND", "Client not found.", nil)
	case errors.Is(err, trainer_clients.ErrClientAlreadyActive):
		RespondError(c, http.StatusConflict, "CLIENT_ALREADY_ADDED", "Client already added.", nil)
	case errors.Is(err, trainer_clients.ErrClientRelationNotFound):
		RespondError(c, http.StatusNotFound, "CLIENT_RELATION_NOT_FOUND", "Client relationship not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
