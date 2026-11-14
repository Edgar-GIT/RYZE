package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ryze/backend/middleware/authcontext"
	"ryze/backend/services/achievements"
)

// AchievementsHandler serves the member's EXP, lifter level and record medals.
type AchievementsHandler struct {
	service achievements.Service
}

func NewAchievementsHandler(service achievements.Service) *AchievementsHandler {
	return &AchievementsHandler{service: service}
}

func achievementRules() gin.H {
	return gin.H{
		"workout_xp":       achievements.WorkoutXP,
		"set_xp":           achievements.SetXP,
		"max_sets_counted": achievements.MaxSetsCounted,
		"record_xp":        achievements.RecordXP,
		"sessions_per_day": achievements.SessionsPerDay,
	}
}

// Summary handles GET /api/v1/me/achievements.
func (h *AchievementsHandler) Summary(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	summary, err := h.service.Summary(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	medals := summary.Medals
	if medals == nil {
		medals = []achievements.Medal{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"xp":               summary.XP,
			"level":            summary.Level,
			"next_level":       summary.NextLevel,
			"levels":           achievements.Levels,
			"rules":            achievementRules(),
			"workouts_counted": summary.WorkoutsCounted,
			"medal_count":      summary.MedalCount,
			"medals":           medals,
		},
	})
}

// Session handles GET /api/v1/me/achievements/sessions/:sessionID: what one
// completed session earned, with the level before and after it.
func (h *AchievementsHandler) Session(c *gin.Context) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return
	}
	award, err := h.service.Session(c.Request.Context(), userID, c.Param("sessionID"))
	if err != nil {
		switch {
		case errors.Is(err, achievements.ErrSessionNotFound):
			RespondError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "Workout session not found.", nil)
		case errors.Is(err, achievements.ErrSessionNotCompleted):
			RespondError(c, http.StatusConflict, "SESSION_NOT_COMPLETED", "The workout session is not completed.", nil)
		default:
			RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		}
		return
	}
	before, _ := achievements.LevelFor(award.XPBefore)
	after, next := achievements.LevelFor(award.XPAfter)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"award":        award,
			"level_before": before,
			"level_after":  after,
			"next_level":   next,
			"rules":        achievementRules(),
		},
	})
}
