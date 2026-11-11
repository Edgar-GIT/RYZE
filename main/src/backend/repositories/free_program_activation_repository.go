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
	// ErrFreeActivationNotFound indicates no live Free activation exists for the
	// requested user and program.
	ErrFreeActivationNotFound = errors.New("free program activation not found")
	// ErrFreeActivationAlreadyActive indicates the same Free program is already
	// active for the user.
	ErrFreeActivationAlreadyActive = errors.New("free program activation already active")
)

// FreeProgramActivationRepository defines the data-access operations for Free
// program activations. Every operation receives the user id explicitly and
// never obtains it from an HTTP context. A user may hold at most one active
// activation; activating another Free program replaces the previous one.
type FreeProgramActivationRepository interface {
	Activate(ctx context.Context, userID, programID string, activation *models.FreeProgramActivation) error
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.FreeProgramActivation, error)
	ListActiveByUser(ctx context.Context, userID string) ([]models.FreeProgramActivation, error)
}

type freeProgramActivationRepository struct {
	db *gorm.DB
}

func NewFreeProgramActivationRepository(db *gorm.DB) FreeProgramActivationRepository {
	return &freeProgramActivationRepository{db: db}
}

// Activate creates a Free activation, replacing any previous activation for the
// user. The generated active_free_activation unique index is the final backstop
// against two concurrent activations.
func (r *freeProgramActivationRepository) Activate(ctx context.Context, userID, programID string, activation *models.FreeProgramActivation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.FreeProgramActivation
		err := tx.Where("user_id = ?", userID).First(&existing).Error
		switch {
		case err == nil:
			if existing.ProgramID == programID && (existing.ExpiresAt == nil || existing.ExpiresAt.After(time.Now().UTC())) {
				return ErrFreeActivationAlreadyActive
			}
			if err := tx.Delete(&existing).Error; err != nil {
				return fmt.Errorf("failed to replace free program activation: %w", err)
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			// No existing activation.
		default:
			return fmt.Errorf("failed to check existing free program activation: %w", err)
		}

		activation.UserID = userID
		activation.ProgramID = programID
		if err := tx.Create(activation).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrFreeActivationAlreadyActive
			}
			return fmt.Errorf("failed to create free program activation: %w", err)
		}
		return nil
	})
}

// FindActiveByUserAndProgram returns the live activation for the pair. Expired
// and soft-deleted rows are excluded.
func (r *freeProgramActivationRepository) FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.FreeProgramActivation, error) {
	var activation models.FreeProgramActivation
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND program_id = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)", userID, programID).
		First(&activation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFreeActivationNotFound
		}
		return nil, fmt.Errorf("failed to find free program activation: %w", err)
	}
	return &activation, nil
}

// ListActiveByUser returns the user's live activations with their program
// metadata preloaded. The one-active rule means the slice has at most one row.
func (r *freeProgramActivationRepository) ListActiveByUser(ctx context.Context, userID string) ([]models.FreeProgramActivation, error) {
	var activations []models.FreeProgramActivation
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)", userID).
		Preload("Program").
		Order("activated_at DESC, id DESC").
		Find(&activations).Error; err != nil {
		return nil, fmt.Errorf("failed to list free program activations: %w", err)
	}
	return activations, nil
}
