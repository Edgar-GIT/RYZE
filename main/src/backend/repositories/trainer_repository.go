package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	// ErrTrainerNotFound indicates the requested trainer does not exist or is
	// not in the state required by the operation (for example, an active-only
	// operation targeting a soft-deleted trainer).
	ErrTrainerNotFound = errors.New("trainer not found")
	// ErrTrainerAlreadyLinked indicates the user already owns an active trainer
	// profile. The one-to-one rule is enforced at the database level by the
	// unique index on the active trainer's user identifier.
	ErrTrainerAlreadyLinked = errors.New("user already has an active trainer")
)

// TrainerRepository defines the data-access operations for trainers.
type TrainerRepository interface {
	Create(ctx context.Context, trainer *models.Trainer) error
	FindByID(ctx context.Context, id string) (*models.Trainer, error)
	FindByIDIncludingDeleted(ctx context.Context, id string) (*models.Trainer, error)
	FindByUserID(ctx context.Context, userID string) (*models.Trainer, error)
	ListActive(ctx context.Context, page, limit int) ([]models.Trainer, int64, error)
	ListDeleted(ctx context.Context, page, limit int) ([]models.Trainer, int64, error)
	SoftDelete(ctx context.Context, id string) error
	Reactivate(ctx context.Context, id string) error
}

type trainerRepository struct {
	db *gorm.DB
}

func NewTrainerRepository(db *gorm.DB) TrainerRepository {
	return &trainerRepository{db: db}
}

func (r *trainerRepository) Create(ctx context.Context, trainer *models.Trainer) error {
	if err := r.db.WithContext(ctx).Create(trainer).Error; err != nil {
		if isDuplicateEntry(err) {
			return ErrTrainerAlreadyLinked
		}
		return fmt.Errorf("failed to create trainer: %w", err)
	}
	return nil
}

// FindByID returns one active trainer with its linked user. Soft-deleted
// trainers are never returned.
func (r *trainerRepository) FindByID(ctx context.Context, id string) (*models.Trainer, error) {
	var trainer models.Trainer
	if err := r.db.WithContext(ctx).Preload("User").First(&trainer, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTrainerNotFound
		}
		return nil, fmt.Errorf("failed to find trainer by id: %w", err)
	}
	return &trainer, nil
}

// FindByIDIncludingDeleted looks up a trainer by id without excluding
// soft-deleted rows. It is used by the admin lifecycle operations
// (reactivation) to inspect trainer profiles that are no longer active.
func (r *trainerRepository) FindByIDIncludingDeleted(ctx context.Context, id string) (*models.Trainer, error) {
	var trainer models.Trainer
	if err := r.db.WithContext(ctx).Unscoped().First(&trainer, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTrainerNotFound
		}
		return nil, fmt.Errorf("failed to find trainer by id: %w", err)
	}
	return &trainer, nil
}

// FindByUserID returns the active trainer profile owned by the given user, or
// ErrTrainerNotFound when the user does not own an active trainer profile.
func (r *trainerRepository) FindByUserID(ctx context.Context, userID string) (*models.Trainer, error) {
	var trainer models.Trainer
	if err := r.db.WithContext(ctx).First(&trainer, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTrainerNotFound
		}
		return nil, fmt.Errorf("failed to find trainer by user id: %w", err)
	}
	return &trainer, nil
}

// ListActive returns one page of active trainers (soft-deleted trainers are
// excluded by GORM's default scope) ordered by creation time, plus the total
// number of active trainers. The caller guarantees page >= 1 and limit >= 1.
func (r *trainerRepository) ListActive(ctx context.Context, page, limit int) ([]models.Trainer, int64, error) {
	var trainers []models.Trainer
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Trainer{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count trainers: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Preload("User").
		Order("created_at DESC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&trainers).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list trainers: %w", err)
	}
	return trainers, total, nil
}

// ListDeleted returns one page of soft-deleted trainers (rows with a populated
// deleted_at) ordered by deletion time, plus the total number of soft-deleted
// trainers. It is used by the admin lifecycle management as a clearly separated
// view from the normal active-trainer listing. The caller guarantees page >= 1
// and limit >= 1.
func (r *trainerRepository) ListDeleted(ctx context.Context, page, limit int) ([]models.Trainer, int64, error) {
	var trainers []models.Trainer
	var total int64

	if err := r.db.WithContext(ctx).
		Unscoped().
		Model(&models.Trainer{}).
		Where("deleted_at IS NOT NULL").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count deleted trainers: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Unscoped().
		Preload("User").
		Where("deleted_at IS NOT NULL").
		Order("deleted_at DESC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&trainers).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list deleted trainers: %w", err)
	}
	return trainers, total, nil
}

// SoftDelete soft-deletes an active trainer by its id. The row is never
// physically removed and the linked user account is never touched: the two
// lifecycles stay independent.
func (r *trainerRepository) SoftDelete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Trainer{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete trainer: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTrainerNotFound
	}
	return nil
}

// Reactivate restores a soft-deleted trainer, clearing deleted_at only: the
// same trainer UUID, user link and created_at are preserved. The `deleted_at
// IS NOT NULL` guard makes reactivation of an already-active trainer
// impossible. Reactivation fails with ErrTrainerAlreadyLinked when the user
// already owns another active trainer profile.
func (r *trainerRepository) Reactivate(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Unscoped().
		Model(&models.Trainer{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Updates(map[string]any{
			"deleted_at": nil,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		if isDuplicateEntry(result.Error) {
			return ErrTrainerAlreadyLinked
		}
		return fmt.Errorf("failed to reactivate trainer: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTrainerNotFound
	}
	return nil
}
