package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/public_profiles"
	"ryze/backend/services/token"
)

// PublicProfilesHandler serves public profiles (open to everyone, owner
// decision 2026-11-21) and following. Following requires a session; the
// follower always comes from the authentication context.
type PublicProfilesHandler struct {
	service *public_profiles.Service
	tokens  token.Service
}

func NewPublicProfilesHandler(service *public_profiles.Service, tokens token.Service) *PublicProfilesHandler {
	return &PublicProfilesHandler{service: service, tokens: tokens}
}

type publicPersonResponse struct {
	Username  string  `json:"username"`
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
}

func newPublicPerson(person public_profiles.Person) publicPersonResponse {
	return publicPersonResponse{Username: person.Username, Nickname: person.DisplayName, AvatarURL: AvatarURL(person.AvatarKey)}
}

// viewer returns the signed-in account, or "" for a visitor. It never
// rejects the request.
func (h *PublicProfilesHandler) viewer(c *gin.Context) string {
	if h.tokens == nil {
		return ""
	}
	value, err := c.Cookie(AccessTokenCookieName)
	if err != nil || value == "" {
		return ""
	}
	claims, err := h.tokens.ValidateAccessToken(value)
	if err != nil {
		return ""
	}
	return claims.UserID
}

// Get handles GET /profiles/:username.
func (h *PublicProfilesHandler) Get(c *gin.Context) {
	viewerID := h.viewer(c)
	profile, err := h.service.Get(c.Request.Context(), c.Param("username"), viewerID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	var coachID *string
	if profile.CoachID != "" {
		coachID = &profile.CoachID
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Profile retrieved successfully.",
		"data": gin.H{
			"username":     profile.Username,
			"nickname":     profile.DisplayName,
			"avatar_url":   AvatarURL(profile.AvatarKey),
			"member_since": profile.MemberSince.UTC().Format(time.RFC3339),
			"stats": gin.H{
				"workouts":  profile.Workouts,
				"followers": profile.Followers,
				"following": profile.Following,
				"level":     gin.H{"rank": profile.LevelRank, "key": profile.LevelKey},
				"medals":    profile.Medals,
				"views":     profile.CoachViews,
			},
			"coach_id": coachID,
			"about":    profile.About,
			"viewer": gin.H{
				"signed_in": viewerID != "",
				"is_self":   profile.ViewerIsSelf,
				"following": profile.ViewerFollowing,
			},
		},
	})
}

// Followers handles GET /profiles/:username/followers.
func (h *PublicProfilesHandler) Followers(c *gin.Context) {
	people, err := h.service.Followers(c.Request.Context(), c.Param("username"))
	h.respondPeople(c, people, err)
}

// Following handles GET /profiles/:username/following.
func (h *PublicProfilesHandler) Following(c *gin.Context) {
	people, err := h.service.Following(c.Request.Context(), c.Param("username"))
	h.respondPeople(c, people, err)
}

func (h *PublicProfilesHandler) respondPeople(c *gin.Context, people []public_profiles.Person, err error) {
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]publicPersonResponse, 0, len(people))
	for _, person := range people {
		items = append(items, newPublicPerson(person))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "People retrieved successfully.", "data": gin.H{"people": items}})
}

// Follow handles POST /me/following/:username.
func (h *PublicProfilesHandler) Follow(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	if err := h.service.Follow(c.Request.Context(), userID, c.Param("username")); err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Following.", "data": gin.H{"following": true}})
}

// Unfollow handles DELETE /me/following/:username.
func (h *PublicProfilesHandler) Unfollow(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	if err := h.service.Unfollow(c.Request.Context(), userID, c.Param("username")); err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Not following.", "data": gin.H{"following": false}})
}

func (h *PublicProfilesHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, public_profiles.ErrNotFound):
		RespondError(c, http.StatusNotFound, "PROFILE_NOT_FOUND", "Profile not found.", nil)
	case errors.Is(err, public_profiles.ErrFollowSelf):
		RespondError(c, http.StatusBadRequest, "CANNOT_FOLLOW_SELF", "You cannot follow yourself.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}
