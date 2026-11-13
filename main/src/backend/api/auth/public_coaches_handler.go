package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/coach_showcase"
	"ryze/backend/services/token"
)

// PublicCoachesHandler serves the public coach showcase and the program view
// counter. Every endpoint is public; the optional session cookie is read only
// to keep a coach's own visits out of the counter.
type PublicCoachesHandler struct {
	service coach_showcase.Service
	tokens  token.Service
}

func NewPublicCoachesHandler(service coach_showcase.Service, tokens token.Service) *PublicCoachesHandler {
	return &PublicCoachesHandler{service: service, tokens: tokens}
}

type publicCoachResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// coachProgramResponse keeps the public program card fields (the same names
// as publicProgramResponse) and adds the view counter and the coach.
type coachProgramResponse struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	Type             string              `json:"type"`
	Status           string              `json:"status"`
	Level            *string             `json:"level"`
	DurationWeeks    *int                `json:"duration_weeks"`
	FrequencyPerWeek *int                `json:"frequency_per_week"`
	TrainingType     *string             `json:"training_type"`
	PriceMinorUnits  int64               `json:"price_minor_units"`
	Currency         string              `json:"currency"`
	ProductType      string              `json:"product_type"`
	Objective        *string             `json:"objective"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	Views            int64               `json:"views"`
	Coach            publicCoachResponse `json:"coach"`
}

func newCoachProgramResponse(program coach_showcase.Program) coachProgramResponse {
	return coachProgramResponse{
		ID:               program.ID,
		Name:             program.Name,
		Description:      program.Description,
		Type:             program.Type,
		Status:           "published",
		Level:            program.Level,
		DurationWeeks:    program.DurationWeeks,
		FrequencyPerWeek: program.FrequencyPerWeek,
		TrainingType:     program.TrainingType,
		PriceMinorUnits:  program.PriceMinorUnits,
		Currency:         program.Currency,
		ProductType:      program.ProductType,
		CreatedAt:        program.CreatedAt,
		UpdatedAt:        program.CreatedAt,
		Views:            program.Views,
		Coach:            publicCoachResponse{ID: program.Coach.ID, Name: program.Coach.Name},
	}
}

// RecordView counts one view of a published program page. It always answers
// 204 for a published program, whether or not the view was counted, so the
// response reveals nothing about deduplication or ownership.
func (h *PublicCoachesHandler) RecordView(c *gin.Context) {
	viewer := coach_showcase.Viewer{Key: "ip:" + digest(c.ClientIP())}
	if userID := h.optionalUserID(c); userID != "" {
		viewer = coach_showcase.Viewer{Key: "user:" + userID, UserID: userID}
	}

	if _, err := h.service.RecordView(c.Request.Context(), c.Param("programID"), viewer); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// FeaturedPrograms returns the featured coach programs, most viewed first.
func (h *PublicCoachesHandler) FeaturedPrograms(c *gin.Context) {
	limit, err := queryInt(c, "limit", coach_showcase.DefaultFeaturedLimit)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	programs, err := h.service.FeaturedPrograms(c.Request.Context(), limit)
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]coachProgramResponse, 0, len(programs))
	for _, program := range programs {
		items = append(items, newCoachProgramResponse(program))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Featured coach programs retrieved successfully.",
		"data":    gin.H{"programs": items},
	})
}

// GetCoach returns a public coach profile with the coach's published
// programs. Only coaches with at least one published program are public.
func (h *PublicCoachesHandler) GetCoach(c *gin.Context) {
	profile, err := h.service.GetProfile(c.Request.Context(), c.Param("trainerID"))
	if err != nil {
		h.respondError(c, err)
		return
	}
	items := make([]coachProgramResponse, 0, len(profile.Programs))
	for _, program := range profile.Programs {
		items = append(items, newCoachProgramResponse(program))
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Coach retrieved successfully.",
		"data": gin.H{
			"coach": gin.H{
				"id":          profile.Coach.ID,
				"name":        profile.Coach.Name,
				"coach_since": profile.CoachSince,
				"total_views": profile.TotalViews,
			},
			"programs": items,
		},
	})
}

// optionalUserID returns the signed-in user, or "" when the request carries no
// valid session. It never rejects the request.
func (h *PublicCoachesHandler) optionalUserID(c *gin.Context) string {
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

func (h *PublicCoachesHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, coach_showcase.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
	case errors.Is(err, coach_showcase.ErrNotFound):
		RespondError(c, http.StatusNotFound, "NOT_FOUND", "Not found.", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}
