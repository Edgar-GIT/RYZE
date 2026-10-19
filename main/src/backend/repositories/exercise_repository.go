package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"ryze/backend/models"
)

// ErrExerciseNotFound indicates the exercise does not exist or is soft-deleted.
// The catalog is global: there is no ownership scoping on this entity.
var ErrExerciseNotFound = errors.New("exercise not found")

// ErrExerciseNameTaken indicates another active catalogue entry already uses
// the requested name. The comparison follows the database collation, so names
// that differ only in case collide too.
var ErrExerciseNameTaken = errors.New("exercise name already taken")

// isDuplicateEntryError reports whether an error is a MySQL duplicate-key
// violation, which for the exercise catalogue can only be the active name or
// the active source linkage.
func isDuplicateEntryError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// ExerciseSearchFilter groups the optional criteria the exercise library can be
// browsed by. Empty fields are ignored. The vocabulary of the fields is
// validated by the service layer, never here.
type ExerciseSearchFilter struct {
	Query      string
	Muscle     string
	Equipment  string
	Difficulty string
	Category   string
	BodyPart   string
}

// AdminExerciseFilter groups the optional criteria the administration
// catalogue view can be browsed by. Unlike the public filter it also returns
// draft and archived entries; soft-deleted rows are never returned.
type AdminExerciseFilter struct {
	Query    string
	Status   string
	BodyPart string
	Source   string
}

// ExerciseUpdate carries the mutable columns of one catalogue entry. A nil
// field is left untouched, so callers can update a single field without
// rewriting the whole row. Status is validated by the service layer before it
// reaches the repository.
type ExerciseUpdate struct {
	Description           *string
	Instructions          *string
	InstructionSteps      *string
	TargetMuscles         *string
	PrimaryMuscleGroup    *string
	SecondaryMuscleGroups *string
	Equipment             *string
	Difficulty            *string
	MovementCategory      *string
	BodyPart              *string
	ImageURL              *string
	AnimationURL          *string
	SourceMediaID         *string
	MediaAttribution      *string
	Status                *string
	// Source and SourceID are written by the dataset importer when it links an
	// existing entry to its source record. The administration surface never
	// sets them.
	Source   *string
	SourceID *string
}

// ExerciseAlternativeLink is one active alternative link with the linked
// exercise name already resolved from the catalog, so callers never need a
// second lookup.
type ExerciseAlternativeLink struct {
	ID                    string
	ExerciseID            string
	AlternativeExerciseID string
	AlternativeName       string
	CreatedAt             time.Time
}

// ExerciseRepository defines the data-access operations for the global
// exercise catalog. The catalog is platform-owned; the public read surface and
// the administration/import surface share this repository so both always
// operate on the same integrity rules. Soft-deleted exercises are excluded
// through GORM's default scope; the public read operations additionally
// exclude everything that is not published.
type ExerciseRepository interface {
	FindByID(ctx context.Context, exerciseID string) (*models.Exercise, error)
	List(ctx context.Context, page, limit int) ([]models.Exercise, int64, error)
	Search(ctx context.Context, query string, page, limit int) ([]models.Exercise, int64, error)
	Library(ctx context.Context, filter ExerciseSearchFilter, page, limit int) ([]models.Exercise, int64, error)
	ListAlternatives(ctx context.Context, exerciseID string) ([]ExerciseAlternativeLink, error)
	// ListActive returns every non-deleted catalogue entry regardless of
	// publication status. It exists for administrative tooling and the dataset
	// importer; it is never used by a public read path.
	ListActive(ctx context.Context) ([]models.Exercise, error)
	// FindBySource returns the entry linked to an import source record, or
	// ErrExerciseNotFound. Publication status is irrelevant: the importer must
	// enrich archived and draft entries without republishing them.
	FindBySource(ctx context.Context, source, sourceID string) (*models.Exercise, error)
	// FindAnyByID returns one non-deleted entry regardless of publication
	// status for administrative reads.
	FindAnyByID(ctx context.Context, exerciseID string) (*models.Exercise, error)
	// FindByName returns the non-deleted entry with the given name, or
	// ErrExerciseNotFound.
	FindByName(ctx context.Context, name string) (*models.Exercise, error)
	// Create inserts one catalogue entry.
	Create(ctx context.Context, exercise *models.Exercise) error
	// Update applies the non-nil fields of the update to one entry. It reports
	// ErrExerciseNotFound when the entry does not exist or is soft-deleted.
	Update(ctx context.Context, exerciseID string, update ExerciseUpdate) error
	// AdminList returns one page of catalogue entries including draft and
	// archived ones, plus the total number of matches.
	AdminList(ctx context.Context, filter AdminExerciseFilter, page, limit int) ([]models.Exercise, int64, error)
}

type exerciseRepository struct {
	db *gorm.DB
}

func NewExerciseRepository(db *gorm.DB) ExerciseRepository {
	return &exerciseRepository{db: db}
}

// Library returns one page of active exercises ordered alphabetically by name,
// restricted by the optional filter criteria, plus the total number of matches.
// The Muscle criterion matches either the primary muscle group or any of the
// comma-separated secondary groups; Equipment and Query are case-insensitive
// substring matches over free-text columns. LIKE wildcards twice escaped so a
// term can never widen into a full scan. The caller guarantees page >= 1,
// limit >= 1 and that the filter uses validated vocabulary.
func (r *exerciseRepository) Library(ctx context.Context, filter ExerciseSearchFilter, page, limit int) ([]models.Exercise, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Exercise{})
	query = applyExerciseFilters(query, filter)

	var exercises []models.Exercise
	var total int64

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count exercises: %w", err)
	}

	if err := query.
		Order("name ASC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&exercises).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list exercises: %w", err)
	}
	return exercises, total, nil
}

// ListAlternatives returns the active directed alternative links leaving the
// given exercise, ordered by the linked exercise name. Alternative links that
// point to a soft-deleted or unpublished exercise are never returned.
func (r *exerciseRepository) ListAlternatives(ctx context.Context, exerciseID string) ([]ExerciseAlternativeLink, error) {
	var links []ExerciseAlternativeLink
	if err := r.db.WithContext(ctx).
		Model(&models.ExerciseAlternative{}).
		Select("exercise_alternatives.id, exercise_alternatives.exercise_id, exercise_alternatives.alternative_exercise_id, exercises.name AS alternative_name, exercise_alternatives.created_at").
		Joins("JOIN exercises ON exercises.id = exercise_alternatives.alternative_exercise_id").
		Where("exercise_alternatives.exercise_id = ? AND exercises.deleted_at IS NULL AND exercises.status = ?", exerciseID, models.ExerciseStatusPublished).
		Order("exercises.name ASC, exercise_alternatives.id ASC").
		Scan(&links).Error; err != nil {
		return nil, fmt.Errorf("failed to list exercise alternatives: %w", err)
	}
	return links, nil
}

// FindByID returns one published exercise. Soft-deleted and unpublished
// exercises are never returned and an unknown id is indistinguishable from a
// missing one on this public read path.
func (r *exerciseRepository) FindByID(ctx context.Context, exerciseID string) (*models.Exercise, error) {
	var exercise models.Exercise
	if err := r.db.WithContext(ctx).
		First(&exercise, "id = ? AND status = ?", exerciseID, models.ExerciseStatusPublished).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExerciseNotFound
		}
		return nil, fmt.Errorf("failed to find exercise: %w", err)
	}
	return &exercise, nil
}

// List returns one page of active exercises ordered alphabetically by name,
// plus the total number of active exercises. The caller guarantees
// page >= 1 and limit >= 1.
func (r *exerciseRepository) List(ctx context.Context, page, limit int) ([]models.Exercise, int64, error) {
	return r.Library(ctx, ExerciseSearchFilter{}, page, limit)
}

// Search returns one page of active exercises whose name contains the query
// (case-insensitive), plus the total number of matches. The caller guarantees
// page >= 1, limit >= 1 and a non-empty trimmed query.
func (r *exerciseRepository) Search(ctx context.Context, query string, page, limit int) ([]models.Exercise, int64, error) {
	return r.Library(ctx, ExerciseSearchFilter{Query: query}, page, limit)
}

// applyExerciseFilters narrows a base exercises query with every non-empty
// filter field and restricts it to published entries. LIKE wildcards in every
// user-supplied term are escaped.
func applyExerciseFilters(query *gorm.DB, filter ExerciseSearchFilter) *gorm.DB {
	query = query.Where("status = ?", models.ExerciseStatusPublished)
	if filter.Query != "" {
		pattern := "%" + escapeLikePattern(filter.Query) + "%"
		query = query.Where("name LIKE ?", pattern)
	}
	if filter.Muscle != "" {
		musclePattern := "%" + escapeLikePattern(filter.Muscle) + "%"
		query = query.Where(
			"(primary_muscle_group = ? OR secondary_muscle_groups LIKE ? OR target_muscles LIKE ?)",
			filter.Muscle, musclePattern, musclePattern,
		)
	}
	if filter.Equipment != "" {
		pattern := "%" + escapeLikePattern(filter.Equipment) + "%"
		query = query.Where("equipment LIKE ?", pattern)
	}
	if filter.Difficulty != "" {
		query = query.Where("difficulty = ?", filter.Difficulty)
	}
	if filter.Category != "" {
		query = query.Where("movement_category = ?", filter.Category)
	}
	if filter.BodyPart != "" {
		query = query.Where("body_part = ?", filter.BodyPart)
	}
	return query
}

// escapeLikePattern neutralizes LIKE wildcards so that user input is matched
// literally. GORM already binds the value as a parameter, so this only guards
// against the wildcards themselves, not SQL injection.
func escapeLikePattern(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// ListActive returns every non-deleted catalogue entry regardless of
// publication status, ordered alphabetically by name. It is only used by the
// importer, which must be able to match entries that are currently drafts or
// archived.
func (r *exerciseRepository) ListActive(ctx context.Context) ([]models.Exercise, error) {
	var exercises []models.Exercise
	if err := r.db.WithContext(ctx).
		Order("name ASC, id ASC").
		Find(&exercises).Error; err != nil {
		return nil, fmt.Errorf("failed to list exercises: %w", err)
	}
	return exercises, nil
}

// FindBySource returns the entry linked to an import source record. Soft-deleted
// entries are ignored; an unknown pair is reported as ErrExerciseNotFound.
func (r *exerciseRepository) FindBySource(ctx context.Context, source, sourceID string) (*models.Exercise, error) {
	var exercise models.Exercise
	if err := r.db.WithContext(ctx).
		First(&exercise, "source = ? AND source_id = ?", source, sourceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExerciseNotFound
		}
		return nil, fmt.Errorf("failed to find exercise by source: %w", err)
	}
	return &exercise, nil
}

// FindAnyByID returns one non-deleted entry regardless of publication status.
// It is the administrative lookup; the public read path uses FindByID, which
// only returns published entries.
func (r *exerciseRepository) FindAnyByID(ctx context.Context, exerciseID string) (*models.Exercise, error) {
	var exercise models.Exercise
	if err := r.db.WithContext(ctx).
		First(&exercise, "id = ?", exerciseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExerciseNotFound
		}
		return nil, fmt.Errorf("failed to find exercise: %w", err)
	}
	return &exercise, nil
}

// FindByName returns the non-deleted entry with the given name, using the
// database collation for the comparison. An unknown name is reported as
// ErrExerciseNotFound.
func (r *exerciseRepository) FindByName(ctx context.Context, name string) (*models.Exercise, error) {
	var exercise models.Exercise
	if err := r.db.WithContext(ctx).
		First(&exercise, "name = ?", name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExerciseNotFound
		}
		return nil, fmt.Errorf("failed to find exercise by name: %w", err)
	}
	return &exercise, nil
}

// Create inserts one catalogue entry. A name that is already used by another
// active entry is reported as ErrExerciseNameTaken.
func (r *exerciseRepository) Create(ctx context.Context, exercise *models.Exercise) error {
	if err := r.db.WithContext(ctx).Create(exercise).Error; err != nil {
		if isDuplicateEntryError(err) {
			return ErrExerciseNameTaken
		}
		return fmt.Errorf("failed to create exercise: %w", err)
	}
	return nil
}

// Update applies the non-nil fields of the update to one entry. It reports
// ErrExerciseNotFound when no non-deleted entry matches.
func (r *exerciseRepository) Update(ctx context.Context, exerciseID string, update ExerciseUpdate) error {
	columns := updateColumns(update)
	if len(columns) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Model(&models.Exercise{}).
		Where("id = ?", exerciseID).
		Updates(columns)
	if result.Error != nil {
		return fmt.Errorf("failed to update exercise: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrExerciseNotFound
	}
	return nil
}

// AdminList returns one page of catalogue entries including draft and archived
// ones, ordered alphabetically by name, plus the total number of matches.
// Soft-deleted entries are never returned. The caller guarantees page >= 1,
// limit >= 1 and validated filter values.
func (r *exerciseRepository) AdminList(ctx context.Context, filter AdminExerciseFilter, page, limit int) ([]models.Exercise, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Exercise{})
	if filter.Query != "" {
		pattern := "%" + escapeLikePattern(filter.Query) + "%"
		query = query.Where("name LIKE ?", pattern)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.BodyPart != "" {
		query = query.Where("body_part = ?", filter.BodyPart)
	}
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}

	var exercises []models.Exercise
	var total int64

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count exercises: %w", err)
	}

	if err := query.
		Order("name ASC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&exercises).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list exercises: %w", err)
	}
	return exercises, total, nil
}

// updateColumns converts the update struct into the column map handed to GORM.
// Only the columns listed here can ever be written through this repository,
// which keeps identity and deletion state immutable. Source linkage is only
// writable for the importer's enrichment path; the administration surface never
// populates those fields.
func updateColumns(update ExerciseUpdate) map[string]any {
	columns := map[string]any{}
	setIfPresent := func(column string, value *string) {
		if value != nil {
			columns[column] = *value
		}
	}
	setIfPresent("description", update.Description)
	setIfPresent("instructions", update.Instructions)
	// The instruction steps column is JSON: an explicitly cleared list is
	// stored as NULL, never as an empty document the JSON validity check would
	// reject.
	if update.InstructionSteps != nil {
		if *update.InstructionSteps == "" {
			columns["instruction_steps"] = nil
		} else {
			columns["instruction_steps"] = *update.InstructionSteps
		}
	}
	setIfPresent("target_muscles", update.TargetMuscles)
	setIfPresent("primary_muscle_group", update.PrimaryMuscleGroup)
	setIfPresent("secondary_muscle_groups", update.SecondaryMuscleGroups)
	setIfPresent("equipment", update.Equipment)
	setIfPresent("difficulty", update.Difficulty)
	setIfPresent("movement_category", update.MovementCategory)
	// The body-part classification is nullable: an explicitly cleared value is
	// stored as NULL, never as the empty string the vocabulary check would
	// reject.
	if update.BodyPart != nil {
		if *update.BodyPart == "" {
			columns["body_part"] = nil
		} else {
			columns["body_part"] = *update.BodyPart
		}
	}
	setIfPresent("image_url", update.ImageURL)
	setIfPresent("animation_url", update.AnimationURL)
	setIfPresent("source_media_id", update.SourceMediaID)
	setIfPresent("media_attribution", update.MediaAttribution)
	setIfPresent("status", update.Status)
	setIfPresent("source", update.Source)
	setIfPresent("source_id", update.SourceID)
	return columns
}
