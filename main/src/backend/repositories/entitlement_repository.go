package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	// ErrEntitlementNotFound indicates the entitlement does not exist, is
	// soft-deleted or does not belong to the requested user.
	ErrEntitlementNotFound = errors.New("entitlement not found")
	// ErrEntitlementAlreadyExists indicates an active entitlement already exists
	// for the same (user, program) pair. The one-active-entitlement rule is
	// enforced at the database level by the unique index on the active
	// entitlement generated column.
	ErrEntitlementAlreadyExists = errors.New("active entitlement already exists")
)

// EntitlementRepository defines the data-access operations for the
// entitlement entity. Every operation receives the user id explicitly; the
// repository never obtains it from an HTTP context, so a client-supplied user
// id can never influence a query. Soft-deleted entitlements are excluded from
// regular queries through GORM's default scope.
type EntitlementRepository interface {
	Create(ctx context.Context, userID, programID string, entitlement *models.Entitlement) error
	ListByUser(ctx context.Context, userID string) ([]models.Entitlement, error)
	FindByIDAndUser(ctx context.Context, userID, entitlementID string) (*models.Entitlement, error)
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Entitlement, error)
	RestoreByUserAndProgram(ctx context.Context, userID, programID string) error
	SoftDelete(ctx context.Context, userID, entitlementID string) error
}

type entitlementRepository struct {
	db *gorm.DB
}

func NewEntitlementRepository(db *gorm.DB) EntitlementRepository {
	return &entitlementRepository{db: db}
}

// Create persists a new entitlement for the given user and program. The caller
// must verify that the user and program exist. The unique active_entitlement
// generated column is the final backstop against duplicate active entitlements.
func (r *entitlementRepository) Create(ctx context.Context, userID, programID string, entitlement *models.Entitlement) error {
	entitlement.UserID = userID
	entitlement.ProgramID = programID
	if err := inSavepoint(r.db.WithContext(ctx), func(tx *gorm.DB) error { return tx.Create(entitlement).Error }); err != nil {
		if isDuplicateEntry(err) {
			return ErrEntitlementAlreadyExists
		}
		return fmt.Errorf("failed to create entitlement: %w", err)
	}
	return nil
}

// ListByUser returns every active entitlement for the given user, in
// chronological order, with the associated published program data preloaded.
// Soft-deleted entitlements are excluded through GORM's default scope.
func (r *entitlementRepository) ListByUser(ctx context.Context, userID string) ([]models.Entitlement, error) {
	var entitlements []models.Entitlement
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)", userID).
		Preload("Program").
		Order("created_at ASC, id ASC").
		Find(&entitlements).Error; err != nil {
		return nil, fmt.Errorf("failed to list entitlements: %w", err)
	}
	return entitlements, nil
}

// FindByIDAndUser returns one active entitlement scoped by both the user and
// the entitlement id. A missing, soft-deleted or cross-user entitlement is
// indistinguishable from a missing one and never revealed.
func (r *entitlementRepository) FindByIDAndUser(ctx context.Context, userID, entitlementID string) (*models.Entitlement, error) {
	var entitlement models.Entitlement
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)", entitlementID, userID).
		Preload("Program").
		First(&entitlement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEntitlementNotFound
		}
		return nil, fmt.Errorf("failed to find entitlement: %w", err)
	}
	return &entitlement, nil
}

// FindActiveByUserAndProgram returns the active (non-deleted) entitlement for
// the given user and program pair, or ErrEntitlementNotFound when none exists.
func (r *entitlementRepository) FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Entitlement, error) {
	var entitlement models.Entitlement
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND program_id = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)", userID, programID).
		First(&entitlement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEntitlementNotFound
		}
		return nil, fmt.Errorf("failed to find active entitlement: %w", err)
	}
	return &entitlement, nil
}

// RestoreByUserAndProgram reclaims a soft-deleted or expired entitlement for
// the given (user, program) pair, effectively reactivating it. If none exists,
// ErrEntitlementNotFound is returned. The generated one-active constraint keeps
// at most one live row, so the restore path must be used instead of inserting a
// second entitlement.
func (r *entitlementRepository) RestoreByUserAndProgram(ctx context.Context, userID, programID string) error {
	result := r.db.WithContext(ctx).
		Unscoped().
		Where("user_id = ? AND program_id = ? AND (deleted_at IS NOT NULL OR expires_at <= CURRENT_TIMESTAMP)", userID, programID).
		Model(&models.Entitlement{}).
		Updates(map[string]any{
			"deleted_at": nil,
			"expires_at": nil,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to restore entitlement: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrEntitlementNotFound
	}
	return nil
}

// SoftDelete soft-deletes one of the user's entitlements. Only the entitlement
// row is touched and it is never removed. The program is never touched. The
// soft-deleted entitlement disappears from all regular queries.
func (r *entitlementRepository) SoftDelete(ctx context.Context, userID, entitlementID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", entitlementID, userID).
		Delete(&models.Entitlement{})
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete entitlement: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrEntitlementNotFound
	}
	return nil
}
