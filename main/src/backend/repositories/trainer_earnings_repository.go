package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// TrainerEarningsRow totals one currency of a trainer's completed sales.
type TrainerEarningsRow struct {
	Currency   string
	Sales      int64
	Gross      int64
	Net        int64
	Commission int64
}

// TrainerEarningsRepository sums the commercial snapshot of the purchases of
// a trainer's plans (owner request, 2026-11-21).
type TrainerEarningsRepository interface {
	Totals(ctx context.Context, trainerID string) ([]TrainerEarningsRow, error)
}

type trainerEarningsRepository struct {
	db *gorm.DB
}

func NewTrainerEarningsRepository(db *gorm.DB) TrainerEarningsRepository {
	return &trainerEarningsRepository{db: db}
}

// Totals counts only completed, real (non-Test Mode) purchases of programs the
// trainer owns, and the Premium Level 2 services they delivered. The amounts are the split captured at purchase time, so a
// later commission change never rewrites past earnings.
func (r *trainerEarningsRepository) Totals(ctx context.Context, trainerID string) ([]TrainerEarningsRow, error) {
	var rows []TrainerEarningsRow
	err := r.db.WithContext(ctx).Raw(`
SELECT pu.currency AS currency,
       COUNT(*) AS sales,
       COALESCE(SUM(pu.price_minor_units), 0) AS gross,
       COALESCE(SUM(pu.trainer_amount), 0) AS net,
       COALESCE(SUM(pu.platform_amount), 0) AS commission
FROM purchases pu
JOIN programs p ON p.id = pu.program_id
WHERE (
        p.trainer_id = ?
        -- Premium Level 2 sales the trainer delivered (v2.131.0): the split is
        -- recorded on the purchase at delivery.
        OR EXISTS (
            SELECT 1 FROM premium_level_2_requests r
            WHERE r.user_id = pu.user_id AND r.program_id = pu.program_id
              AND r.trainer_id = ? AND r.status = 'delivered' AND r.deleted_at IS NULL
        )
      )
  AND pu.status = 'completed'
  AND pu.test = FALSE
  AND pu.deleted_at IS NULL
GROUP BY pu.currency
ORDER BY gross DESC, pu.currency ASC`, trainerID, trainerID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load trainer earnings: %w", err)
	}
	return rows, nil
}
