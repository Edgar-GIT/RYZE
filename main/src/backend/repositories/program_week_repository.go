package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	// ErrWeekNotFound indicates the program week does not exist, is
	// soft-deleted, does not belong to the requested program or the program is
	// not owned by the trainer performing the operation.
	ErrWeekNotFound = errors.New("program week not found")
	// ErrWeekReorderConflict indicates the reorder list does not match the set
	// of active weeks of the program.
	ErrWeekReorderConflict = errors.New("week reorder list mismatch")
)

// ProgramWeekRepository defines the data-access operations for the program
// weeks entity. Every operation receives the trainer id explicitly and scopes
// the query through the owning program; the repository never obtains the
// trainer id from an HTTP context, so a client-supplied trainer id can never
// influence a query. Soft-deleted weeks and workouts are excluded from regular
// queries through GORM's default scope.
type ProgramWeekRepository interface {
	Create(ctx context.Context, trainerID, programID string, week *models.ProgramWeek) error
	ListByProgram(ctx context.Context, trainerID, programID string) ([]models.ProgramWeek, error)
	FindByIDAndProgram(ctx context.Context, trainerID, programID, weekID string) (*models.ProgramWeek, error)
	Reorder(ctx context.Context, trainerID, programID string, orderedIDs []string) error
	SoftDelete(ctx context.Context, trainerID, programID, weekID string) error
}

type programWeekRepository struct {
	db *gorm.DB
}

func NewProgramWeekRepository(db *gorm.DB) ProgramWeekRepository {
	return &programWeekRepository{db: db}
}

// Create appends a new week to the end of a program owned by the trainer. The
// program row is locked inside the transaction so concurrent week creation is
// serialized and the next week_number is always computed against a stable state;
// the unique active_week index is the final backstop.
func (r *programWeekRepository) Create(ctx context.Context, trainerID, programID string, week *models.ProgramWeek) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var program models.Program
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND trainer_id = ?", programID, trainerID).
			First(&program).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to verify program ownership: %w", err)
		}

		var maxWeek int
		if err := tx.Model(&models.ProgramWeek{}).
			Where("program_id = ?", programID).
			Select("COALESCE(MAX(week_number), 0)").
			Scan(&maxWeek).Error; err != nil {
			return fmt.Errorf("failed to resolve next week number: %w", err)
		}

		week.ProgramID = programID
		week.WeekNumber = maxWeek + 1
		if err := tx.Create(week).Error; err != nil {
			return fmt.Errorf("failed to create program week: %w", err)
		}
		return nil
	})
}

// ListByProgram returns every active week of the trainer's program, ordered by
// week number, with each week's active workouts preloaded in position order.
// A program that is missing, soft-deleted or owned by another trainer simply
// yields an empty list: it is indistinguishable from a program without weeks.
func (r *programWeekRepository) ListByProgram(ctx context.Context, trainerID, programID string) ([]models.ProgramWeek, error) {
	var weeks []models.ProgramWeek
	if err := r.db.WithContext(ctx).
		Where("program_id = ? AND program_id IN (?)", programID,
			trainerScopedProgramID(r.db, trainerID, programID)).
		Order("week_number ASC, id ASC").
		Preload("Workouts", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC, id ASC")
		}).
		Find(&weeks).Error; err != nil {
		return nil, fmt.Errorf("failed to list program weeks: %w", err)
	}
	return weeks, nil
}

// FindByIDAndProgram returns one active week of the trainer's program scoped by
// the trainer, the program and the week id, with its active workouts preloaded
// in position order. A week that is missing, soft-deleted, under another
// program or under a foreign program is indistinguishable and never revealed.
func (r *programWeekRepository) FindByIDAndProgram(ctx context.Context, trainerID, programID, weekID string) (*models.ProgramWeek, error) {
	var week models.ProgramWeek
	if err := r.db.WithContext(ctx).
		Where("id = ? AND program_id = ? AND program_id IN (?)", weekID, programID,
			trainerScopedProgramID(r.db, trainerID, programID)).
		Preload("Workouts", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC, id ASC")
		}).
		First(&week).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWeekNotFound
		}
		return nil, fmt.Errorf("failed to find program week: %w", err)
	}
	return &week, nil
}

// Reorder replaces the week_number of every active week of the trainer's program
// with the position of its id inside orderedIDs. The list must contain every
// active week exactly once; anything else is rejected before any write. The
// reassignment runs in two phases inside one transaction: every active slot is
// first released to a distinct value far outside the final range, then each week
// is assigned its final number, so the unique active_week index can never reject
// an intermediate state and the week_number > 0 check is never violated.
func (r *programWeekRepository) Reorder(ctx context.Context, trainerID, programID string, orderedIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var program models.Program
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND trainer_id = ?", programID, trainerID).
			First(&program).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to verify program ownership: %w", err)
		}

		var weeks []models.ProgramWeek
		if err := tx.Where("program_id = ?", programID).
			Order("week_number ASC, id ASC").
			Find(&weeks).Error; err != nil {
			return fmt.Errorf("failed to load program weeks for reorder: %w", err)
		}
		if err := validateWeekOrder(orderedIDs, weeks); err != nil {
			return err
		}

		if err := tx.Model(&models.ProgramWeek{}).
			Where("program_id = ?", programID).
			Update("week_number", gorm.Expr("week_number + ?", releaseSlotOffset)).Error; err != nil {
			return fmt.Errorf("failed to release program week slots: %w", err)
		}
		for i, id := range orderedIDs {
			if err := tx.Model(&models.ProgramWeek{}).
				Where("id = ? AND program_id = ?", id, programID).
				Update("week_number", i+1).Error; err != nil {
				return fmt.Errorf("failed to assign program week slot: %w", err)
			}
		}
		return nil
	})
}

// SoftDelete soft-deletes one of the trainer's own program weeks. Only the week
// row is touched; the program and the week's workouts are never deleted. The
// workouts become unreachable because every workout query is scoped through an
// active week.
func (r *programWeekRepository) SoftDelete(ctx context.Context, trainerID, programID, weekID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND program_id = ? AND program_id IN (?)", weekID, programID,
			trainerScopedProgramID(r.db, trainerID, programID)).
		Delete(&models.ProgramWeek{})
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete program week: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrWeekNotFound
	}
	return nil
}

// validateWeekOrder rejects reorder lists that are not an exact permutation of
// the active weeks.
func validateWeekOrder(orderedIDs []string, weeks []models.ProgramWeek) error {
	if len(orderedIDs) != len(weeks) {
		return fmt.Errorf("%w: the week list must contain every active week exactly once", ErrWeekReorderConflict)
	}
	seen := make(map[string]struct{}, len(orderedIDs))
	for _, id := range orderedIDs {
		if _, exists := seen[id]; exists {
			return fmt.Errorf("%w: duplicate week id %q", ErrWeekReorderConflict, id)
		}
		seen[id] = struct{}{}
	}
	for _, week := range weeks {
		if _, exists := seen[week.ID]; !exists {
			return fmt.Errorf("%w: active week %q is missing from the order list", ErrWeekReorderConflict, week.ID)
		}
	}
	return nil
}

// releaseSlotOffset is added to every week_number during the first reorder
// phase. It moves every slot far outside the final 1..n range so the unique
// active_week index can never observe a collision, while keeping every value
// positive for the week_number > 0 constraint. A week can never grow beyond
// this number of active weeks.
const releaseSlotOffset = 1000000

// trainerScopedProgramID builds a subquery selecting the id of the active
// program owned by the trainer. It keeps every week query scoped by the trainer
// without trusting any client-supplied identity.
func trainerScopedProgramID(db *gorm.DB, trainerID, programID string) *gorm.DB {
	return db.Model(&models.Program{}).Select("id").Where("id = ? AND trainer_id = ?", programID, trainerID)
}
