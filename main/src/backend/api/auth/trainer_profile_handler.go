package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/trainercontext"
	"ryze/backend/services/trainer_profile"
)

// trainerProfileResponse only exposes safe trainer profile information: the
// trainer identity, the linked user's public profile data and the lifecycle
// timestamps. Password hashes, session versions and deletion markers are never
// exposed.
type trainerProfileResponse struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	User      userResponse `json:"user"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func newTrainerProfileResponse(profile *trainer_profile.Profile) trainerProfileResponse {
	return trainerProfileResponse{
		ID:     profile.TrainerID,
		UserID: profile.UserID,
		User: userResponse{
			ID:        profile.UserID,
			Email:     profile.Email,
			FirstName: profile.FirstName,
			LastName:  profile.LastName,
			CreatedAt: profile.UserCreatedAt,
			UpdatedAt: profile.UserUpdatedAt,
		},
		CreatedAt: profile.TrainerCreatedAt,
		UpdatedAt: profile.TrainerUpdatedAt,
	}
}

// TrainerProfileHandler exposes the authenticated trainer's own profile. It
// never performs authentication or authorization itself: those are enforced by
// the Authenticate, TrainerAuthenticate and RequireTrainerPermission middleware
// mounted on the route. The trainer identity always comes exclusively from the
// trainer context; request parameters, body, headers or any client-supplied
// identity can never influence it.
type TrainerProfileHandler struct {
	service trainer_profile.Service
}

func NewTrainerProfileHandler(svc trainer_profile.Service) *TrainerProfileHandler {
	return &TrainerProfileHandler{service: svc}
}

// GetProfile returns the safe profile of the authenticated trainer. The
// trainer id and the owning user id are resolved exclusively from the trainer
// context; query parameters such as user_id or trainer_id are ignored and can
// never change which profile is returned.
func (h *TrainerProfileHandler) GetProfile(c *gin.Context) {
	identity, err := trainercontext.IdentityFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), identity.UserID, identity.TrainerID)
	if err != nil {
		switch {
		case errors.Is(err, trainer_profile.ErrTrainerNotFound):
			RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Trainer profile retrieved successfully.",
		"data":    newTrainerProfileResponse(profile),
	})
}
