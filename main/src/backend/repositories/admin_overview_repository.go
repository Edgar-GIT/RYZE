package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// DayCount is one day of a daily series (UTC calendar days).
type DayCount struct {
	Day        time.Time
	Count      int64
	MinorUnits int64
}

// LabelCount is one labelled aggregate.
type LabelCount struct {
	Label      string
	Count      int64
	MinorUnits int64
}

// RevenueTotals aggregates completed, non-test purchases.
type RevenueTotals struct {
	Count          int64
	MinorUnits     int64
	PlatformAmount int64
	TrainerAmount  int64
}

// TopProgram is one programme ranked by sales or views.
type TopProgram struct {
	ID         string
	Name       string
	Family     string
	Count      int64
	MinorUnits int64
}

// AdminOverviewRepository reads the aggregates of the administration
// dashboards. Every query is read-only and returns counts and sums only: no
// personal data, payment references or message content.
type AdminOverviewRepository interface {
	Revenue(ctx context.Context, from, to time.Time) (RevenueTotals, error)
	RevenueAllTime(ctx context.Context) (RevenueTotals, error)
	RevenueByFamily(ctx context.Context, from time.Time) ([]LabelCount, error)
	RevenueDaily(ctx context.Context, from time.Time) ([]DayCount, error)
	PurchaseCounts(ctx context.Context, from time.Time) (pending, failed, test int64, err error)
	TopSellingPrograms(ctx context.Context, from time.Time, limit int) ([]TopProgram, error)
	TopViewedPrograms(ctx context.Context, limit int) ([]TopProgram, error)
	MemberCounts(ctx context.Context, from time.Time) (total, newInPeriod, verified int64, err error)
	SignupsDaily(ctx context.Context, from time.Time) ([]DayCount, error)
	ActiveMembers(ctx context.Context, since time.Time) (int64, error)
	SessionsCompletedDaily(ctx context.Context, from time.Time) ([]DayCount, error)
	FreeActivations(ctx context.Context, from time.Time) (int64, error)
	ActiveEntitlements(ctx context.Context) (int64, error)
	TrainerCounts(ctx context.Context) (active, pendingApplications int64, err error)
	CatalogueCounts(ctx context.Context) (published []LabelCount, drafts int64, err error)
	QueueCounts(ctx context.Context) (map[string]int64, error)
	DisabledUsers(ctx context.Context) (int64, error)
	AuditCounts(ctx context.Context, since time.Time) (total, denied, failed int64, err error)
}

type adminOverviewRepository struct {
	db *gorm.DB
}

func NewAdminOverviewRepository(db *gorm.DB) AdminOverviewRepository {
	return &adminOverviewRepository{db: db}
}

// realPurchases is the base condition for revenue: completed, real (non-test)
// purchases.
const realPurchases = "pu.deleted_at IS NULL AND pu.status = 'completed' AND pu.test = false"

// realMembers excludes Test Mode personas from every member figure.
const realMembers = "u.deleted_at IS NULL AND (u.test_persona IS NULL OR u.test_persona = '')"

// programFamily labels a programme by catalogue category; trainer plans are
// grouped together.
const programFamily = "CASE WHEN p.trainer_id IS NOT NULL THEN 'trainer' ELSE p.product_type END"

func (r *adminOverviewRepository) scanTotals(ctx context.Context, query string, args ...any) (RevenueTotals, error) {
	var totals RevenueTotals
	row := r.db.WithContext(ctx).Raw(query, args...).Row()
	if err := row.Scan(&totals.Count, &totals.MinorUnits, &totals.PlatformAmount, &totals.TrainerAmount); err != nil {
		return RevenueTotals{}, fmt.Errorf("failed to aggregate revenue: %w", err)
	}
	return totals, nil
}

const revenueSelect = `SELECT COUNT(*), COALESCE(SUM(pu.price_minor_units), 0), COALESCE(SUM(pu.platform_amount), 0), COALESCE(SUM(pu.trainer_amount), 0) FROM purchases pu WHERE `

func (r *adminOverviewRepository) Revenue(ctx context.Context, from, to time.Time) (RevenueTotals, error) {
	return r.scanTotals(ctx, revenueSelect+realPurchases+" AND pu.created_at >= ? AND pu.created_at < ?", from, to)
}

func (r *adminOverviewRepository) RevenueAllTime(ctx context.Context) (RevenueTotals, error) {
	return r.scanTotals(ctx, revenueSelect+realPurchases)
}

func (r *adminOverviewRepository) RevenueByFamily(ctx context.Context, from time.Time) ([]LabelCount, error) {
	var rows []LabelCount
	err := r.db.WithContext(ctx).Raw(`SELECT `+programFamily+` AS label, COUNT(*) AS count, COALESCE(SUM(pu.price_minor_units), 0) AS minor_units
		FROM purchases pu JOIN programs p ON p.id = pu.program_id
		WHERE `+realPurchases+` AND pu.created_at >= ?
		GROUP BY 1 ORDER BY 3 DESC`, from).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate revenue by family: %w", err)
	}
	return rows, nil
}

func (r *adminOverviewRepository) RevenueDaily(ctx context.Context, from time.Time) ([]DayCount, error) {
	var rows []DayCount
	err := r.db.WithContext(ctx).Raw(`SELECT (pu.created_at AT TIME ZONE 'UTC')::date AS day, COUNT(*) AS count, COALESCE(SUM(pu.price_minor_units), 0) AS minor_units
		FROM purchases pu WHERE `+realPurchases+` AND pu.created_at >= ?
		GROUP BY 1 ORDER BY 1`, from).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate daily revenue: %w", err)
	}
	return rows, nil
}

func (r *adminOverviewRepository) PurchaseCounts(ctx context.Context, from time.Time) (int64, int64, int64, error) {
	var pending, failed, test int64
	row := r.db.WithContext(ctx).Raw(`SELECT
		COUNT(*) FILTER (WHERE status = 'pending' AND test = false),
		COUNT(*) FILTER (WHERE status = 'failed' AND test = false AND created_at >= ?),
		COUNT(*) FILTER (WHERE status = 'completed' AND test = true AND created_at >= ?)
		FROM purchases WHERE deleted_at IS NULL`, from, from).Row()
	if err := row.Scan(&pending, &failed, &test); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count purchases: %w", err)
	}
	return pending, failed, test, nil
}

func (r *adminOverviewRepository) TopSellingPrograms(ctx context.Context, from time.Time, limit int) ([]TopProgram, error) {
	var rows []TopProgram
	err := r.db.WithContext(ctx).Raw(`SELECT p.id, p.name, `+programFamily+` AS family, COUNT(*) AS count, COALESCE(SUM(pu.price_minor_units), 0) AS minor_units
		FROM purchases pu JOIN programs p ON p.id = pu.program_id
		WHERE `+realPurchases+` AND pu.created_at >= ?
		GROUP BY p.id, p.name, 3 ORDER BY 5 DESC, 4 DESC LIMIT ?`, from, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to rank programs by sales: %w", err)
	}
	return rows, nil
}

func (r *adminOverviewRepository) TopViewedPrograms(ctx context.Context, limit int) ([]TopProgram, error) {
	var rows []TopProgram
	err := r.db.WithContext(ctx).Raw(`SELECT p.id, p.name, `+programFamily+` AS family, s.view_count AS count, 0 AS minor_units
		FROM program_view_stats s JOIN programs p ON p.id = s.program_id
		WHERE p.deleted_at IS NULL
		ORDER BY s.view_count DESC, p.created_at DESC LIMIT ?`, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to rank programs by views: %w", err)
	}
	return rows, nil
}

func (r *adminOverviewRepository) MemberCounts(ctx context.Context, from time.Time) (int64, int64, int64, error) {
	var total, fresh, verified int64
	row := r.db.WithContext(ctx).Raw(`SELECT COUNT(*), COUNT(*) FILTER (WHERE u.created_at >= ?), COUNT(*) FILTER (WHERE u.email_verified_at IS NOT NULL)
		FROM users u WHERE `+realMembers, from).Row()
	if err := row.Scan(&total, &fresh, &verified); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count members: %w", err)
	}
	return total, fresh, verified, nil
}

func (r *adminOverviewRepository) SignupsDaily(ctx context.Context, from time.Time) ([]DayCount, error) {
	var rows []DayCount
	err := r.db.WithContext(ctx).Raw(`SELECT (u.created_at AT TIME ZONE 'UTC')::date AS day, COUNT(*) AS count, 0 AS minor_units
		FROM users u WHERE `+realMembers+` AND u.created_at >= ? GROUP BY 1 ORDER BY 1`, from).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate signups: %w", err)
	}
	return rows, nil
}

func (r *adminOverviewRepository) ActiveMembers(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(DISTINCT ws.user_id) FROM workout_sessions ws
		JOIN users u ON u.id = ws.user_id
		WHERE ws.deleted_at IS NULL AND ws.started_at >= ? AND `+realMembers, since).Row().Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count active members: %w", err)
	}
	return count, nil
}

func (r *adminOverviewRepository) SessionsCompletedDaily(ctx context.Context, from time.Time) ([]DayCount, error) {
	var rows []DayCount
	err := r.db.WithContext(ctx).Raw(`SELECT (ws.completed_at AT TIME ZONE 'UTC')::date AS day, COUNT(*) AS count, 0 AS minor_units
		FROM workout_sessions ws JOIN users u ON u.id = ws.user_id
		WHERE ws.deleted_at IS NULL AND ws.status = 'completed' AND ws.completed_at >= ? AND `+realMembers+`
		GROUP BY 1 ORDER BY 1`, from).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate sessions: %w", err)
	}
	return rows, nil
}

func (r *adminOverviewRepository) FreeActivations(ctx context.Context, from time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM free_program_activations WHERE deleted_at IS NULL AND activated_at >= ?`, from).Row().Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count free activations: %w", err)
	}
	return count, nil
}

func (r *adminOverviewRepository) ActiveEntitlements(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM entitlements WHERE deleted_at IS NULL AND (expires_at IS NULL OR expires_at > now())`).Row().Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count entitlements: %w", err)
	}
	return count, nil
}

func (r *adminOverviewRepository) TrainerCounts(ctx context.Context) (int64, int64, error) {
	var active, pending int64
	row := r.db.WithContext(ctx).Raw(`SELECT
		(SELECT COUNT(*) FROM trainers WHERE deleted_at IS NULL),
		(SELECT COUNT(*) FROM trainer_applications WHERE deleted_at IS NULL AND status = 'PENDING')`).Row()
	if err := row.Scan(&active, &pending); err != nil {
		return 0, 0, fmt.Errorf("failed to count trainers: %w", err)
	}
	return active, pending, nil
}

func (r *adminOverviewRepository) CatalogueCounts(ctx context.Context) ([]LabelCount, int64, error) {
	var published []LabelCount
	if err := r.db.WithContext(ctx).Raw(`SELECT ` + programFamily + ` AS label, COUNT(*) AS count, 0 AS minor_units
		FROM programs p WHERE p.deleted_at IS NULL AND p.status = 'published' GROUP BY 1 ORDER BY 1`).Scan(&published).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count catalogue: %w", err)
	}
	var drafts int64
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM programs WHERE deleted_at IS NULL AND status = 'draft' AND trainer_id IS NULL`).Row().Scan(&drafts); err != nil {
		return nil, 0, fmt.Errorf("failed to count drafts: %w", err)
	}
	return published, drafts, nil
}

func (r *adminOverviewRepository) QueueCounts(ctx context.Context) (map[string]int64, error) {
	var awaitingIntake, awaitingTrainer, inProgress, ready, supportOpen, feedbackPending int64
	row := r.db.WithContext(ctx).Raw(`SELECT
		(SELECT COUNT(*) FROM premium_level_2_requests WHERE deleted_at IS NULL AND status = 'awaiting_intake'),
		(SELECT COUNT(*) FROM premium_level_2_requests WHERE deleted_at IS NULL AND status = 'awaiting_trainer_assignment'),
		(SELECT COUNT(*) FROM premium_level_2_requests WHERE deleted_at IS NULL AND status = 'in_progress'),
		(SELECT COUNT(*) FROM premium_level_2_requests WHERE deleted_at IS NULL AND status = 'ready_for_delivery'),
		(SELECT COUNT(*) FROM support_conversations WHERE status = 'open'),
		(SELECT COUNT(*) FROM member_feedback WHERE deleted_at IS NULL AND status = 'pending')`).Row()
	if err := row.Scan(&awaitingIntake, &awaitingTrainer, &inProgress, &ready, &supportOpen, &feedbackPending); err != nil {
		return nil, fmt.Errorf("failed to count queues: %w", err)
	}
	return map[string]int64{
		"premium_level_2_awaiting_intake":  awaitingIntake,
		"premium_level_2_awaiting_trainer": awaitingTrainer,
		"premium_level_2_in_progress":      inProgress,
		"premium_level_2_ready":            ready,
		"support_open":                     supportOpen,
		"feedback_pending":                 feedbackPending,
	}, nil
}

func (r *adminOverviewRepository) DisabledUsers(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM users WHERE deleted_at IS NOT NULL`).Row().Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count disabled users: %w", err)
	}
	return count, nil
}

func (r *adminOverviewRepository) AuditCounts(ctx context.Context, since time.Time) (int64, int64, int64, error) {
	var total, denied, failed int64
	row := r.db.WithContext(ctx).Raw(`SELECT COUNT(*), COUNT(*) FILTER (WHERE outcome = 'denied'), COUNT(*) FILTER (WHERE outcome = 'failed')
		FROM admin_audit_logs WHERE created_at >= ?`, since).Row()
	if err := row.Scan(&total, &denied, &failed); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count audit events: %w", err)
	}
	return total, denied, failed, nil
}
