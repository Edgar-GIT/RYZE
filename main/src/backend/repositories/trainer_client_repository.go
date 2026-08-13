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
	// ErrClientRelationNotFound indicates the trainer→client relationship does
	// not exist or is not in the state required by the operation (for example,
	// an active-only operation targeting a soft-deleted relationship).
	ErrClientRelationNotFound = errors.New("trainer-client relationship not found")
	// ErrClientRelationAlreadyActive indicates an active relationship already
	// exists for the same trainer and user. The one-active-relation rule is
	// enforced at the database level by the unique index on the active
	// relation identifier.
	ErrClientRelationAlreadyActive = errors.New("active trainer-client relationship already exists")
)

// TrainerClientRepository defines the data-access operations for the
// trainer→client relationship. Every operation receives the trainer id
// explicitly; the repository never obtains it from an HTTP context, so a
// client-supplied trainer id can never influence a query.
type TrainerClientRepository interface {
	Create(ctx context.Context, relation *models.TrainerClient) error
	FindActiveByTrainerAndUser(ctx context.Context, trainerID, userID string) (*models.TrainerClient, error)
	FindIncludingDeletedByTrainerAndUser(ctx context.Context, trainerID, userID string) (*models.TrainerClient, error)
	ListActiveClients(ctx context.Context, trainerID string, page, limit int) ([]models.TrainerClient, int64, error)
	SoftDelete(ctx context.Context, trainerID, userID string) error
	Reactivate(ctx context.Context, trainerID, userID string) error
}

type trainerClientRepository struct {
	db *gorm.DB
}

func NewTrainerClientRepository(db *gorm.DB) TrainerClientRepository {
	return &trainerClientRepository{db: db}
}

// Create persists a new active relationship. The database rejects a second
// active relationship for the same (trainer_id, user_id) pair through the
// unique index on the active relation identifier.
func (r *trainerClientRepository) Create(ctx context.Context, relation *models.TrainerClient) error {
	if err := r.db.WithContext(ctx).Create(relation).Error; err != nil {
		if isDuplicateEntry(err) {
			return ErrClientRelationAlreadyActive
		}
		return fmt.Errorf("failed to create trainer-client relationship: %w", err)
	}
	return nil
}

// FindActiveByTrainerAndUser returns the active relationship between the
// trainer and the user, with the linked user preloaded. Soft-deleted
// relationships are never returned.
func (r *trainerClientRepository) FindActiveByTrainerAndUser(ctx context.Context, trainerID, userID string) (*models.TrainerClient, error) {
	var relation models.TrainerClient
	if err := r.db.WithContext(ctx).
		Preload("User").
		First(&relation, "trainer_id = ? AND user_id = ?", trainerID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClientRelationNotFound
		}
		return nil, fmt.Errorf("failed to find active trainer-client relationship: %w", err)
	}
	return &relation, nil
}

// FindIncludingDeletedByTrainerAndUser looks up the relationship between the
// trainer and the user without excluding soft-deleted rows. It is used by the
// reactivation lifecycle to inspect and restore the exact same relationship
// row.
func (r *trainerClientRepository) FindIncludingDeletedByTrainerAndUser(ctx context.Context, trainerID, userID string) (*models.TrainerClient, error) {
	var relation models.TrainerClient
	if err := r.db.WithContext(ctx).
		Unscoped().
		Preload("User").
		First(&relation, "trainer_id = ? AND user_id = ?", trainerID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClientRelationNotFound
		}
		return nil, fmt.Errorf("failed to find trainer-client relationship: %w", err)
	}
	return &relation, nil
}

// ListActiveClients returns one page of active client relationships owned by
// the given trainer (soft-deleted relationships are excluded by GORM's default
// scope) with the linked user preloaded, ordered by creation time, plus the
// total number of active clients. The trainer id is always an explicit
// parameter; the caller guarantees page >= 1 and limit >= 1.
func (r *trainerClientRepository) ListActiveClients(ctx context.Context, trainerID string, page, limit int) ([]models.TrainerClient, int64, error) {
	var relations []models.TrainerClient
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&models.TrainerClient{}).
		Where("trainer_id = ?", trainerID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count trainer clients: %w", err)
	}

	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("trainer_id = ?", trainerID).
		Order("created_at DESC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&relations).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list trainer clients: %w", err)
	}
	return relations, total, nil
}

// SoftDelete soft-deletes the active relationship between the trainer and the
// user. Only the relationship row is touched: the linked user account and the
// trainer profile are never deleted, their lifecycles stay independent.
func (r *trainerClientRepository) SoftDelete(ctx context.Context, trainerID, userID string) error {
	result := r.db.WithContext(ctx).
		Delete(&models.TrainerClient{}, "trainer_id = ? AND user_id = ?", trainerID, userID)
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete trainer-client relationship: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrClientRelationNotFound
	}
	return nil
}

// Reactivate restores a soft-deleted relationship, clearing deleted_at only:
// the same relationship UUID, trainer link, user link and created_at are
// preserved, so no second relationship is ever created. The `deleted_at IS NOT
// NULL` guard makes reactivation of an already-active relationship impossible
// and the unique index on the active relation identifier enforces the
// one-active-relation rule at the database level.
func (r *trainerClientRepository) Reactivate(ctx context.Context, trainerID, userID string) error {
	result := r.db.WithContext(ctx).
		Unscoped().
		Model(&models.TrainerClient{}).
		Where("trainer_id = ? AND user_id = ? AND deleted_at IS NOT NULL", trainerID, userID).
		Updates(map[string]any{
			"deleted_at": nil,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		if isDuplicateEntry(result.Error) {
			return ErrClientRelationAlreadyActive
		}
		return fmt.Errorf("failed to reactivate trainer-client relationship: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrClientRelationNotFound
	}
	return nil
}
