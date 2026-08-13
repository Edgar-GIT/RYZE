package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// ErrProgramNotFound indicates the program does not exist, is soft-deleted or
// is not owned by the trainer performing the operation.
var ErrProgramNotFound = errors.New("program not found")

// ProgramRepository defines the data-access operations for the trainer-owned
// program entity. Every operation receives the trainer id explicitly; the
// repository never obtains it from an HTTP context, so a client-supplied
// trainer id can never influence a query. Soft-deleted programs are excluded
// from regular queries through GORM's default scope.
type ProgramRepository interface {
	Create(ctx context.Context, program *models.Program) error
	FindByIDAndTrainer(ctx context.Context, trainerID, programID string) (*models.Program, error)
	ListByTrainer(ctx context.Context, trainerID string, page, limit int) ([]models.Program, int64, error)
	Update(ctx context.Context, trainerID, programID string, updates map[string]any) error
	SoftDelete(ctx context.Context, trainerID, programID string) error
}

type programRepository struct {
	db *gorm.DB
}

func NewProgramRepository(db *gorm.DB) ProgramRepository {
	return &programRepository{db: db}
}

// Create persists a new program. The trainer id is always set by the service
// from the authenticated trainer context; a NULL trainer id is reserved for
// future platform-owned programs and can never be produced by the trainer API.
func (r *programRepository) Create(ctx context.Context, program *models.Program) error {
	if err := r.db.WithContext(ctx).Create(program).Error; err != nil {
		return fmt.Errorf("failed to create program: %w", err)
	}
	return nil
}

// FindByIDAndTrainer returns one active program scoped by both the owner
// trainer and the program id. Soft-deleted programs are never returned and a
// program owned by another trainer is indistinguishable from a missing one.
func (r *programRepository) FindByIDAndTrainer(ctx context.Context, trainerID, programID string) (*models.Program, error) {
	var program models.Program
	if err := r.db.WithContext(ctx).
		First(&program, "id = ? AND trainer_id = ?", programID, trainerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to find program: %w", err)
	}
	return &program, nil
}

// ListByTrainer returns one page of the trainer's active programs (soft-deleted
// programs are excluded by GORM's default scope) ordered by creation time, plus
// the total number of active programs. The trainer id is always an explicit
// parameter; the caller guarantees page >= 1 and limit >= 1.
func (r *programRepository) ListByTrainer(ctx context.Context, trainerID string, page, limit int) ([]models.Program, int64, error) {
	var programs []models.Program
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&models.Program{}).
		Where("trainer_id = ?", trainerID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count programs: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Where("trainer_id = ?", trainerID).
		Order("created_at DESC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&programs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list programs: %w", err)
	}
	return programs, total, nil
}

// Update applies the whitelisted field updates to a program scoped by both the
// owner trainer and the program id. Ownership and existence are verified by the
// caller before updating; a zero RowsAffected simply means the values were
// unchanged, never that the program is missing. trainer_id can never be
// updated through this path.
func (r *programRepository) Update(ctx context.Context, trainerID, programID string, updates map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&models.Program{}).
		Where("id = ? AND trainer_id = ?", programID, trainerID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update program: %w", result.Error)
	}
	return nil
}

// SoftDelete soft-deletes one of the trainer's own programs. Only the program
// row is touched and it is never removed: the soft-deleted program simply
// disappears from all regular queries.
func (r *programRepository) SoftDelete(ctx context.Context, trainerID, programID string) error {
	result := r.db.WithContext(ctx).
		Delete(&models.Program{}, "id = ? AND trainer_id = ?", programID, trainerID)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete program: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrProgramNotFound
	}
	return nil
}
