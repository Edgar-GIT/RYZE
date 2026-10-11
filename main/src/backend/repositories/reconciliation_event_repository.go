package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	// ErrInvalidReconciliationOutcome indicates the outcome is not part of the
	// fixed documented reconciliation vocabulary.
	ErrInvalidReconciliationOutcome = errors.New("invalid reconciliation outcome")
	// ErrInvalidReconciliationInput indicates a reconciliation event record is
	// missing required fields.
	ErrInvalidReconciliationInput = errors.New("invalid reconciliation event")
)

// AdminReconciliationFilter narrows the administrator reconciliation event
// queries. Every value is optional. Open restricts the query to the outcome
// set that requires investigation; when combined with Outcome, the exact
// outcome always wins.
type AdminReconciliationFilter struct {
	Outcome  string
	Open     bool
	Provider string
}

// ReconciliationEventRepository is the data-access surface for the durable
// purchase reconciliation event records written by the webhook layer and read
// by the restricted administrative reconciliation view.
type ReconciliationEventRepository interface {
	// Upsert records one durable reconciliation event per (provider, event_id)
	// delivery. A replay of the same delivery updates the record only when the
	// existing outcome is provisional (transient_failure, provider_uncertain):
	// a resolved verdict is immutable and survives any redelivery, so the
	// history of a settlement is never overwritten by a duplicate probe.
	Upsert(ctx context.Context, event *models.PurchaseReconciliationEvent) error
	// ListAdmin returns one page of reconciliation events matching the filter,
	// most recently recorded first, with the purchase, customer and program
	// preloaded (all unscoped for historical accuracy), plus the total count.
	ListAdmin(ctx context.Context, filter AdminReconciliationFilter, page, limit int) ([]models.PurchaseReconciliationEvent, int64, error)
}

type reconciliationEventRepository struct {
	db *gorm.DB
}

func NewReconciliationEventRepository(db *gorm.DB) ReconciliationEventRepository {
	return &reconciliationEventRepository{db: db}
}

// Upsert persists one durable record of a verified provider delivery's outcome.
// The unique (provider, event_id) key makes a provider retry idempotent at the
// audit layer; the conditional update keeps a resolved verdict immutable while
// letting a later delivery upgrade a provisional outcome to its final one.
func (r *reconciliationEventRepository) Upsert(ctx context.Context, event *models.PurchaseReconciliationEvent) error {
	if event == nil {
		return fmt.Errorf("%w: event is required", ErrInvalidReconciliationInput)
	}
	if event.PurchaseID == "" || event.Provider == "" || event.EventID == "" || event.Outcome == "" {
		return fmt.Errorf("%w: purchase_id, provider, event_id and outcome are required", ErrInvalidReconciliationInput)
	}
	if !models.IsValidReconciliationOutcome(event.Outcome) {
		return fmt.Errorf("%w: %q", ErrInvalidReconciliationOutcome, event.Outcome)
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	provisional := models.ProvisionalReconciliationOutcomes()
	placeholders := make([]string, 0, len(provisional))
	args := make([]any, 0, len(provisional)+6)
	for outcome := range provisional {
		placeholders = append(placeholders, "?")
		args = append(args, outcome)
	}

	sql := `INSERT INTO purchase_reconciliation_events
    (id, purchase_id, provider, event_id, outcome, created_at)
    VALUES (?, ?, ?, ?, ?, ?)
    ON DUPLICATE KEY UPDATE
        outcome = CASE
            WHEN outcome IN (` + strings.Join(placeholders, ", ") + `) THEN VALUES(outcome)
            ELSE outcome
        END`

	args = append([]any{event.ID, event.PurchaseID, event.Provider, event.EventID, event.Outcome, event.CreatedAt}, args...)

	if err := r.db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
		return fmt.Errorf("failed to upsert reconciliation event: %w", err)
	}
	return nil
}

// ListAdmin returns one page of reconciliation events matching the filter,
// most recently recorded first, with the purchase, its customer and its
// program preloaded (both unscoped so soft-deleted identities and retired
// programs keep resolving for historical accuracy).
func (r *reconciliationEventRepository) ListAdmin(ctx context.Context, filter AdminReconciliationFilter, page, limit int) ([]models.PurchaseReconciliationEvent, int64, error) {
	var events []models.PurchaseReconciliationEvent
	var total int64

	db := r.db.WithContext(ctx).Model(&models.PurchaseReconciliationEvent{})
	db = applyReconciliationFilter(db, filter)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count reconciliation events: %w", err)
	}

	query := r.db.WithContext(ctx).
		Model(&models.PurchaseReconciliationEvent{}).
		Preload("Purchase.User", func(q *gorm.DB) *gorm.DB {
			return q.Unscoped()
		}).
		Preload("Purchase.Program", func(q *gorm.DB) *gorm.DB {
			return q.Unscoped()
		}).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset((page - 1) * limit)
	query = applyReconciliationFilter(query, filter)

	if err := query.Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list reconciliation events: %w", err)
	}
	return events, total, nil
}

// applyReconciliationFilter applies the admin reconciliation WHERE clauses.
func applyReconciliationFilter(db *gorm.DB, filter AdminReconciliationFilter) *gorm.DB {
	if filter.Outcome != "" {
		return db.Where("outcome = ?", filter.Outcome)
	}
	if filter.Open {
		open := models.OpenReconciliationOutcomes()
		outcomes := make([]string, 0, len(open))
		for outcome := range open {
			outcomes = append(outcomes, outcome)
		}
		return db.Where("outcome IN ?", outcomes)
	}
	return db
}