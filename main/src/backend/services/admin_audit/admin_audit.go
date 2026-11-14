// Package admin_audit records and lists the immutable administrative audit
// log. Records describe actions (who, what, on which target, with what
// outcome); request content is never stored.
package admin_audit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// ErrInvalidInput indicates a malformed listing query.
var ErrInvalidInput = errors.New("invalid audit query")

// MaxPageSize caps one listing page.
const MaxPageSize = 100

// Entry is one action to record.
type Entry struct {
	AdminID    string
	Method     string
	Route      string
	TargetID   string
	StatusCode int
	ClientIP   string
	// Action overrides the action derived from Method and Route.
	Action string
}

// Record is the listing view of one audit record.
type Record struct {
	ID         string
	AdminID    string
	Action     string
	Method     string
	Route      string
	TargetID   string
	StatusCode int
	Outcome    string
	ClientIP   string
	CreatedAt  time.Time
}

// ListFilter narrows a listing. Empty values are ignored.
type ListFilter struct {
	AdminID string
	Outcome string
}

// ListResult is one page of records.
type ListResult struct {
	Records []Record
	Total   int64
	Page    int
	Limit   int
}

// Service records and lists audit entries.
type Service interface {
	Record(ctx context.Context, entry Entry) error
	List(ctx context.Context, filter ListFilter, page, limit int) (ListResult, error)
}

type service struct {
	repo repositories.AdminAuditRepository
}

func New(repo repositories.AdminAuditRepository) Service {
	return &service{repo: repo}
}

// Record stores one entry. Entries for unknown identities are rejected: only
// the configured administrators are audited here.
func (s *service) Record(ctx context.Context, entry Entry) error {
	if !config.IsValidAdminIdentity(entry.AdminID) {
		return fmt.Errorf("%w: unknown admin identity", ErrInvalidInput)
	}
	method := strings.ToUpper(strings.TrimSpace(entry.Method))
	route := truncate(strings.TrimSpace(entry.Route), 160)
	if method == "" || route == "" || entry.StatusCode < 100 || entry.StatusCode > 599 {
		return fmt.Errorf("%w: incomplete entry", ErrInvalidInput)
	}
	action := entry.Action
	if action == "" {
		action = ActionFor(method, route)
	}

	log := &models.AdminAuditLog{
		AdminID:    entry.AdminID,
		Action:     truncate(action, 64),
		Method:     truncate(method, 8),
		Route:      route,
		StatusCode: entry.StatusCode,
		Outcome:    OutcomeFor(entry.StatusCode),
	}
	if target := truncate(strings.TrimSpace(entry.TargetID), 64); target != "" {
		log.TargetID = &target
	}
	if ip := net.ParseIP(strings.TrimSpace(entry.ClientIP)); ip != nil {
		value := ip.String()
		log.ClientIP = &value
	}
	return s.repo.Create(ctx, log)
}

func (s *service) List(ctx context.Context, filter ListFilter, page, limit int) (ListResult, error) {
	if page < 1 || limit < 1 || limit > MaxPageSize {
		return ListResult{}, ErrInvalidInput
	}
	if filter.AdminID != "" && !config.IsValidAdminIdentity(filter.AdminID) {
		return ListResult{}, ErrInvalidInput
	}
	switch filter.Outcome {
	case "", models.AdminAuditOutcomeSuccess, models.AdminAuditOutcomeDenied, models.AdminAuditOutcomeFailed:
	default:
		return ListResult{}, ErrInvalidInput
	}

	rows, total, err := s.repo.List(ctx, repositories.AdminAuditFilter{AdminID: filter.AdminID, Outcome: filter.Outcome}, page, limit)
	if err != nil {
		return ListResult{}, err
	}
	records := make([]Record, 0, len(rows))
	for _, row := range rows {
		record := Record{
			ID:         row.ID,
			AdminID:    row.AdminID,
			Action:     row.Action,
			Method:     row.Method,
			Route:      row.Route,
			StatusCode: row.StatusCode,
			Outcome:    row.Outcome,
			CreatedAt:  row.CreatedAt,
		}
		if row.TargetID != nil {
			record.TargetID = *row.TargetID
		}
		if row.ClientIP != nil {
			record.ClientIP = *row.ClientIP
		}
		records = append(records, record)
	}
	return ListResult{Records: records, Total: total, Page: page, Limit: limit}, nil
}

// OutcomeFor maps an HTTP status to an audit outcome: 2xx/3xx succeeded, 401
// and 403 were denied, anything else failed (validation, conflicts, errors).
func OutcomeFor(status int) string {
	switch {
	case status >= 200 && status < 400:
		return models.AdminAuditOutcomeSuccess
	case status == 401 || status == 403:
		return models.AdminAuditOutcomeDenied
	default:
		return models.AdminAuditOutcomeFailed
	}
}

// ShouldAudit reports whether a request on an administrative route is an
// auditable action: every state-changing method, plus evidence downloads
// (reading a trainer's identity documents is sensitive on its own).
func ShouldAudit(method, route string) bool {
	switch strings.ToUpper(method) {
	case "GET", "HEAD", "OPTIONS":
		return strings.HasSuffix(route, "/download")
	default:
		return true
	}
}

// ActionSignIn is the action of a completed (or refused) second-factor sign-in.
const ActionSignIn = "admin.sign_in"

// actions names the administrative routes. Unknown routes fall back to
// "<method> <route>" so nothing is ever left unrecorded.
var actions = map[string]string{
	"POST /api/v1/admin/auth/test-mode": "test_mode.enter",

	"POST /api/v1/admin/users":                "user.create",
	"PATCH /api/v1/admin/users/:id":           "user.update",
	"PATCH /api/v1/admin/users/:id/disable":   "user.disable",
	"POST /api/v1/admin/users/:id/reactivate": "user.reactivate",
	"POST /api/v1/admin/users/:id/password":   "user.password_reset",

	"POST /api/v1/admin/trainers":                "trainer.create",
	"PATCH /api/v1/admin/trainers/:id":           "trainer.update",
	"PATCH /api/v1/admin/trainers/:id/disable":   "trainer.disable",
	"POST /api/v1/admin/trainers/:id/reactivate": "trainer.reactivate",

	"POST /api/v1/admin/trainer-applications/:id/approve":                       "trainer_application.approve",
	"POST /api/v1/admin/trainer-applications/:id/reject":                        "trainer_application.reject",
	"GET /api/v1/admin/trainer-applications/:id/documents/:documentID/download": "trainer_application.document_download",

	"POST /api/v1/admin/feedback/:id/approve": "feedback.approve",
	"POST /api/v1/admin/feedback/:id/reject":  "feedback.reject",

	"POST /api/v1/admin/support/conversations/:conversationID/messages": "support.reply",
	"POST /api/v1/admin/support/conversations/:conversationID/read":     "support.mark_read",
	"POST /api/v1/admin/support/conversations/:conversationID/close":    "support.close",
	"POST /api/v1/admin/support/conversations/:conversationID/reopen":   "support.reopen",

	"POST /api/v1/admin/exercises":              "exercise.create",
	"PATCH /api/v1/admin/exercises/:exerciseID": "exercise.update",

	"POST /api/v1/admin/premium-level-2/requests/:requestID/trainer": "premium_level_2.assign_trainer",

	"PATCH /api/v1/admin/programs/:programID/pricing": "program.pricing_update",
	"POST /api/v1/admin/uploads/program-cover":        "program.cover_upload",

	"PATCH /api/v1/admin/trainers/:id/commission":  "commission.update",
	"DELETE /api/v1/admin/trainers/:id/commission": "commission.delete",

	"POST /api/v1/programs/generic":                            "generic_program.create",
	"PATCH /api/v1/programs/generic/:programID":                "generic_program.update",
	"POST /api/v1/programs/generic/:programID/publish":         "generic_program.publish",
	"DELETE /api/v1/programs/generic/:programID":               "generic_program.delete",
	"POST /api/v1/programs/premium-level-1":                    "premium_level_1_program.create",
	"PATCH /api/v1/programs/premium-level-1/:programID":        "premium_level_1_program.update",
	"POST /api/v1/programs/premium-level-1/:programID/publish": "premium_level_1_program.publish",
	"DELETE /api/v1/programs/premium-level-1/:programID":       "premium_level_1_program.delete",
	"POST /api/v1/programs/premium-level-2":                    "premium_level_2_program.create",
	"PATCH /api/v1/programs/premium-level-2/:programID":        "premium_level_2_program.update",
	"POST /api/v1/programs/premium-level-2/:programID/publish": "premium_level_2_program.publish",
	"DELETE /api/v1/programs/premium-level-2/:programID":       "premium_level_2_program.delete",
}

// ActionFor names the action of a method and route template.
func ActionFor(method, route string) string {
	key := strings.ToUpper(method) + " " + route
	if action, ok := actions[key]; ok {
		return action
	}
	return strings.ToLower(method) + " " + route
}

// TargetParams lists, in priority order, the path parameters that identify
// the record an action touched.
var TargetParams = []string{"id", "programID", "exerciseID", "requestID", "conversationID", "documentID"}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
