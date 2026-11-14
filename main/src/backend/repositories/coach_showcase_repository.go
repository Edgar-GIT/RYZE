package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// ErrCoachNotFound indicates the trainer is unknown, disabled, a Test Mode
// persona, or has no published program. The cases are indistinguishable: a
// trainer only becomes public by publishing.
var ErrCoachNotFound = errors.New("coach not found")

// CoachProgramRow is one published, trainer-owned generic program with its
// view counter and the owning coach's public name.
type CoachProgramRow struct {
	ID               string
	Name             string
	Description      string
	Type             string
	Status           string
	Level            *string
	DurationWeeks    *int
	FrequencyPerWeek *int
	TrainingType     *string
	CoverImage       *string
	PriceMinorUnits  int64
	Currency         string
	ProductType      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ViewCount        int64
	CoachID          string
	CoachFirstName   string
	CoachLastName    string
}

// PublicCoachRow is the public identity of one coach.
type PublicCoachRow struct {
	ID        string
	FirstName string
	LastName  string
	CreatedAt time.Time
}

// ProgramViewTarget describes a published program for view counting: whether
// it exists and which user owns it through its trainer ("" when the program is
// platform-owned).
type ProgramViewTarget struct {
	OwnerUserID string
}

// CoachShowcaseRepository reads the public coach showcase and maintains the
// program view counters.
type CoachShowcaseRepository interface {
	FindProgramViewTarget(ctx context.Context, programID string) (*ProgramViewTarget, error)
	RecordView(ctx context.Context, programID string, at time.Time) error
	ListFeaturedCoachPrograms(ctx context.Context, limit int) ([]CoachProgramRow, error)
	FindPublicCoach(ctx context.Context, trainerID string) (*PublicCoachRow, error)
	ListCoachPrograms(ctx context.Context, trainerID string, limit int) ([]CoachProgramRow, error)
}

type coachShowcaseRepository struct {
	db *gorm.DB
}

func NewCoachShowcaseRepository(db *gorm.DB) CoachShowcaseRepository {
	return &coachShowcaseRepository{db: db}
}

// publicCoachJoin limits every showcase query to coaches who may appear in
// public: an active trainer row, an active user account and no Test Mode
// persona (test content is never shown on the live marketplace).
const publicCoachJoin = `
JOIN trainers t ON t.id = p.trainer_id AND t.deleted_at IS NULL
JOIN users u ON u.id = t.user_id AND u.deleted_at IS NULL AND u.test_persona IS NULL`

const coachProgramColumns = `
SELECT p.id, p.name, COALESCE(p.description, '') AS description, p.type, p.status, p.level,
       p.duration_weeks, p.frequency_per_week, p.training_type, p.cover_image, p.price_minor_units,
       p.currency, p.product_type, p.created_at, p.updated_at,
       COALESCE(v.view_count, 0) AS view_count,
       t.id AS coach_id, u.first_name AS coach_first_name, u.last_name AS coach_last_name
FROM programs p` + publicCoachJoin + `
LEFT JOIN program_view_stats v ON v.program_id = p.id`

// publishedGenericProgram is the predicate shared by every showcase listing:
// published, not deleted, and a generic program (Premium families have their
// own pages and gates).
const publishedGenericProgram = `
p.deleted_at IS NULL AND p.status = '` + models.ProgramStatusPublished + `' AND p.product_type = '` + models.ProgramProductTypeGeneric + `'`

func (r *coachShowcaseRepository) FindProgramViewTarget(ctx context.Context, programID string) (*ProgramViewTarget, error) {
	var rows []struct {
		OwnerUserID *string
	}
	err := r.db.WithContext(ctx).Raw(`
SELECT t.user_id AS owner_user_id
FROM programs p
LEFT JOIN trainers t ON t.id = p.trainer_id
WHERE p.id = ? AND p.deleted_at IS NULL AND p.status = ?`, programID, models.ProgramStatusPublished).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load program view target: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrProgramNotFound
	}
	target := &ProgramViewTarget{}
	if rows[0].OwnerUserID != nil {
		target.OwnerUserID = *rows[0].OwnerUserID
	}
	return target, nil
}

// RecordView adds one view to the program's counter, creating the counter on
// the first view.
func (r *coachShowcaseRepository) RecordView(ctx context.Context, programID string, at time.Time) error {
	err := r.db.WithContext(ctx).Exec(`
INSERT INTO program_view_stats (program_id, view_count, last_viewed_at)
VALUES (?, 1, ?)
ON CONFLICT (program_id) DO UPDATE
SET view_count = program_view_stats.view_count + 1,
    last_viewed_at = EXCLUDED.last_viewed_at`, programID, at).Error
	if err != nil {
		return fmt.Errorf("failed to record program view: %w", err)
	}
	return nil
}

// ListFeaturedCoachPrograms ranks published trainer programs by views (most
// viewed first); programs with equal views, including new programs with none,
// follow newest first.
func (r *coachShowcaseRepository) ListFeaturedCoachPrograms(ctx context.Context, limit int) ([]CoachProgramRow, error) {
	var rows []CoachProgramRow
	err := r.db.WithContext(ctx).Raw(coachProgramColumns+`
WHERE`+publishedGenericProgram+`
ORDER BY view_count DESC, p.created_at DESC, p.id ASC
LIMIT ?`, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list featured coach programs: %w", err)
	}
	return rows, nil
}

func (r *coachShowcaseRepository) FindPublicCoach(ctx context.Context, trainerID string) (*PublicCoachRow, error) {
	var rows []PublicCoachRow
	err := r.db.WithContext(ctx).Raw(`
SELECT t.id, u.first_name, u.last_name, t.created_at
FROM trainers t
JOIN users u ON u.id = t.user_id AND u.deleted_at IS NULL AND u.test_persona IS NULL
WHERE t.id = ? AND t.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM programs p WHERE p.trainer_id = t.id AND`+publishedGenericProgram+`)`, trainerID).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find public coach: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrCoachNotFound
	}
	return &rows[0], nil
}

// ListCoachPrograms returns the coach's published generic programs, most
// viewed first.
func (r *coachShowcaseRepository) ListCoachPrograms(ctx context.Context, trainerID string, limit int) ([]CoachProgramRow, error) {
	var rows []CoachProgramRow
	err := r.db.WithContext(ctx).Raw(coachProgramColumns+`
WHERE t.id = ? AND`+publishedGenericProgram+`
ORDER BY view_count DESC, p.created_at DESC, p.id ASC
LIMIT ?`, trainerID, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list coach programs: %w", err)
	}
	return rows, nil
}
