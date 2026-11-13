package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/admin_audit"
)

// AdminAuditHandler serves the read-only administrative audit log.
type AdminAuditHandler struct {
	service admin_audit.Service
}

func NewAdminAuditHandler(service admin_audit.Service) *AdminAuditHandler {
	return &AdminAuditHandler{service: service}
}

type adminAuditEntryResponse struct {
	ID         string    `json:"id"`
	AdminID    string    `json:"admin_id"`
	Action     string    `json:"action"`
	Method     string    `json:"method"`
	Route      string    `json:"route"`
	TargetID   *string   `json:"target_id"`
	StatusCode int       `json:"status_code"`
	Outcome    string    `json:"outcome"`
	ClientIP   *string   `json:"client_ip"`
	CreatedAt  time.Time `json:"created_at"`
}

// List returns one page of audit records, newest first. Optional filters:
// admin (ADMIN_1 or ADMIN_2) and outcome (success, denied, failed).
func (h *AdminAuditHandler) List(c *gin.Context) {
	page, err := queryInt(c, "page", 1)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}
	limit, err := queryInt(c, "limit", 50)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
		return
	}

	result, err := h.service.List(c.Request.Context(), admin_audit.ListFilter{
		AdminID: strings.TrimSpace(c.Query("admin")),
		Outcome: strings.TrimSpace(c.Query("outcome")),
	}, page, limit)
	if err != nil {
		if errors.Is(err, admin_audit.ErrInvalidInput) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed.", nil)
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}

	entries := make([]adminAuditEntryResponse, 0, len(result.Records))
	for _, record := range result.Records {
		entries = append(entries, adminAuditEntryResponse{
			ID:         record.ID,
			AdminID:    record.AdminID,
			Action:     record.Action,
			Method:     record.Method,
			Route:      record.Route,
			TargetID:   optionalString(record.TargetID),
			StatusCode: record.StatusCode,
			Outcome:    record.Outcome,
			ClientIP:   optionalString(record.ClientIP),
			CreatedAt:  record.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Audit log retrieved successfully.",
		"data": gin.H{
			"entries": entries,
			"pagination": gin.H{
				"page":        result.Page,
				"limit":       result.Limit,
				"total":       result.Total,
				"total_pages": totalPages(result.Total, result.Limit),
			},
		},
	})
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
