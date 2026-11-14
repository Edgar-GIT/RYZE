package auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/admin_overview"
)

// AdminOverviewHandler serves the administration dashboards. Authorization
// (commerce for the business overview, system for the platform overview) is
// enforced by the route middleware.
type AdminOverviewHandler struct {
	service admin_overview.Service
}

func NewAdminOverviewHandler(svc admin_overview.Service) *AdminOverviewHandler {
	return &AdminOverviewHandler{service: svc}
}

type overviewDayResponse struct {
	Date       string `json:"date"`
	Count      int64  `json:"count"`
	MinorUnits int64  `json:"minor_units"`
}

type overviewShareResponse struct {
	Label      string `json:"label"`
	Count      int64  `json:"count"`
	MinorUnits int64  `json:"minor_units"`
}

type overviewProgramResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Family     string `json:"family"`
	Count      int64  `json:"count"`
	MinorUnits int64  `json:"minor_units"`
}

func overviewDays(points []admin_overview.DayPoint) []overviewDayResponse {
	result := make([]overviewDayResponse, 0, len(points))
	for _, point := range points {
		result = append(result, overviewDayResponse{Date: point.Date, Count: point.Count, MinorUnits: point.MinorUnits})
	}
	return result
}

func overviewShares(items []admin_overview.Share) []overviewShareResponse {
	result := make([]overviewShareResponse, 0, len(items))
	for _, item := range items {
		result = append(result, overviewShareResponse{Label: item.Label, Count: item.Count, MinorUnits: item.MinorUnits})
	}
	return result
}

func overviewPrograms(items []admin_overview.RankedProgram) []overviewProgramResponse {
	result := make([]overviewProgramResponse, 0, len(items))
	for _, item := range items {
		result = append(result, overviewProgramResponse{ID: item.ID, Name: item.Name, Family: item.Family, Count: item.Count, MinorUnits: item.MinorUnits})
	}
	return result
}

func overviewRange(c *gin.Context) int {
	days, _ := strconv.Atoi(c.Query("days"))
	return admin_overview.NormalizeRange(days)
}

// Business returns the Management Administrator overview (GET /admin/overview/business).
func (h *AdminOverviewHandler) Business(c *gin.Context) {
	overview, err := h.service.Business(c.Request.Context(), overviewRange(c))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"generated_at": overview.GeneratedAt.Format(time.RFC3339),
			"range_days":   overview.RangeDays,
			"currency":     "EUR",
			"revenue": gin.H{
				"all_time_minor_units": overview.RevenueAllTime,
				"all_time_sales":       overview.SalesAllTime,
				"minor_units":          overview.Revenue,
				"previous_minor_units": overview.PreviousRevenue,
				"sales":                overview.Sales,
				"previous_sales":       overview.PreviousSales,
				"platform_minor_units": overview.PlatformShare,
				"trainer_minor_units":  overview.TrainerShare,
				"by_family":            overviewShares(overview.RevenueByFamily),
				"daily":                overviewDays(overview.RevenueDaily),
			},
			"purchases": gin.H{
				"pending": overview.PendingPurchases,
				"failed":  overview.FailedPurchases,
				"test":    overview.TestPurchases,
			},
			"members": gin.H{
				"total":           overview.Members,
				"new":             overview.NewMembers,
				"signups_daily":   overviewDays(overview.SignupsDaily),
				"active_7d":       overview.ActiveMembers7d,
				"active_in_range": overview.ActiveMembersRange,
			},
			"training": gin.H{
				"sessions_completed":  overview.SessionsCompleted,
				"sessions_daily":      overviewDays(overview.SessionsDaily),
				"free_activations":    overview.FreeActivations,
				"active_entitlements": overview.ActiveEntitlements,
			},
			"trainers": gin.H{
				"active":               overview.ActiveTrainers,
				"pending_applications": overview.PendingApplications,
			},
			"catalogue": gin.H{
				"published_by_family": overviewShares(overview.PublishedByFamily),
				"platform_drafts":     overview.PlatformDrafts,
			},
			"queues":      overview.Queues,
			"top_selling": overviewPrograms(overview.TopSelling),
			"top_viewed":  overviewPrograms(overview.TopViewed),
		},
	})
}

// Platform returns the Technical Administrator overview (GET /admin/overview/platform).
func (h *AdminOverviewHandler) Platform(c *gin.Context) {
	overview, err := h.service.Platform(c.Request.Context(), overviewRange(c))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"generated_at": overview.GeneratedAt.Format(time.RFC3339),
			"range_days":   overview.RangeDays,
			"members": gin.H{
				"total":         overview.Members,
				"new":           overview.NewMembers,
				"verified":      overview.VerifiedMembers,
				"disabled":      overview.DisabledUsers,
				"signups_daily": overviewDays(overview.SignupsDaily),
				"active_7d":     overview.ActiveMembers7d,
			},
			"training": gin.H{
				"sessions_completed": overview.SessionsCompleted,
				"sessions_daily":     overviewDays(overview.SessionsDaily),
			},
			"trainers": gin.H{
				"active":               overview.ActiveTrainers,
				"pending_applications": overview.PendingApplications,
			},
			"audit": gin.H{
				"events_24h": overview.AuditEvents24h,
				"denied_24h": overview.AuditDenied24h,
				"failed_24h": overview.AuditFailed24h,
			},
		},
	})
}

// Queues returns the open work counts of the management queues
// (GET /admin/overview/queues), used as navigation badges.
func (h *AdminOverviewHandler) Queues(c *gin.Context) {
	queues, err := h.service.Queues(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": queues})
}
