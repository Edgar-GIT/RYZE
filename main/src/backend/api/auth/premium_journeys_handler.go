package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/premium_match"
	"ryze/backend/services/program_details"
)

// PremiumJourneysHandler serves the client journeys of the premium services
// (owner request, 2026-11-14): the Premium Level 1 pick. The Premium Level 2
// questionnaire is part of the purchased service, not a checkout prerequisite.
// The client is always the authenticated user.
type PremiumJourneysHandler struct {
	matches premium_match.Service
}

func NewPremiumJourneysHandler(matches premium_match.Service) *PremiumJourneysHandler {
	return &PremiumJourneysHandler{matches: matches}
}

type premiumMatchRequest struct {
	Objective string  `json:"objective"`
	Sex       string  `json:"sex"`
	WeightKg  float64 `json:"weight_kg"`
	HeightCm  float64 `json:"height_cm"`
	Age       int     `json:"age"`
}

// Match picks a Premium Level 1 package (POST /me/premium-level-1/match).
func (h *PremiumJourneysHandler) Match(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	var req premiumMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	result, err := h.matches.Match(c.Request.Context(), userID, premium_match.Input{
		Objective: req.Objective,
		Sex:       req.Sex,
		WeightKg:  req.WeightKg,
		HeightCm:  req.HeightCm,
		Age:       req.Age,
	})
	if err != nil {
		switch {
		case errors.Is(err, premium_match.ErrInvalidInput):
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		case errors.Is(err, premium_match.ErrNoPackages):
			RespondError(c, http.StatusNotFound, "NO_MATCHING_PLAN", "No plan is available for this objective yet.", nil)
		case errors.Is(err, premium_match.ErrAllOwned):
			RespondError(c, http.StatusConflict, "ALL_PLANS_OWNED", "You already own every plan for this objective.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}
	program := result.Program
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"match": gin.H{
				"id":         result.Match.ID,
				"objective":  result.Match.Objective,
				"created_at": result.Match.CreatedAt,
			},
			"program": gin.H{
				"id":                 program.ID,
				"name":               program.Name,
				"description":        program.Description,
				"type":               program.Type,
				"product_type":       program.ProductType,
				"objective":          program.Objective,
				"nutrition_mode":     program.NutritionMode,
				"level":              program.Level,
				"duration_weeks":     program.DurationWeeks,
				"frequency_per_week": program.FrequencyPerWeek,
				"training_type":      program.TrainingType,
				"price_minor_units":  program.PriceMinorUnits,
				"currency":           program.Currency,
				"cover_image_url":    program_details.CoverURL(program.CoverImage),
			},
		},
	})
}
