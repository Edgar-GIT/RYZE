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
	// ErrApplicationNotFound indicates the requested trainer application does
	// not exist or is not in the state required by the operation (for example,
	// a soft-deleted application is never visible to regular operations).
	ErrApplicationNotFound = errors.New("trainer application not found")
	// ErrApplicationStateConflict indicates the application exists but is not
	// in the state required by the operation (for example, approving or
	// rejecting an application that is no longer PENDING).
	ErrApplicationStateConflict = errors.New("trainer application is not in the required state")
	// ErrApplicationAlreadyActive indicates the user already owns an active
	// (PENDING or APPROVED) trainer application. The rule is enforced at the
	// database level by the unique index on the active application's user
	// identifier.
	ErrApplicationAlreadyActive = errors.New("user already has an active trainer application")
)

// TrainerApplicationRepository defines the data-access operations for trainer
// applications.
type TrainerApplicationRepository interface {
	Create(ctx context.Context, application *models.TrainerApplication) error
	FindActiveByUserID(ctx context.Context, userID string) (*models.TrainerApplication, error)
	FindLatestByUserID(ctx context.Context, userID string) (*models.TrainerApplication, error)
	FindByID(ctx context.Context, id string) (*models.TrainerApplication, error)
	List(ctx context.Context, page, limit int, status string) ([]models.TrainerApplication, int64, error)
	// Approve transitions a PENDING application to APPROVED, creates the
	// trainer profile and, when a decision is provided, records it in the same
	// transaction.
	Approve(ctx context.Context, applicationID string, decision *models.TrainerApplicationDecision) (*models.TrainerApplication, error)
	// Reject transitions a PENDING application to REJECTED and, when a decision
	// is provided, records it in the same transaction.
	Reject(ctx context.Context, applicationID string, decision *models.TrainerApplicationDecision) error
}

type trainerApplicationRepository struct {
	db *gorm.DB
}

func NewTrainerApplicationRepository(db *gorm.DB) TrainerApplicationRepository {
	return &trainerApplicationRepository{db: db}
}

func (r *trainerApplicationRepository) Create(ctx context.Context, application *models.TrainerApplication) error {
	if err := r.db.WithContext(ctx).Create(application).Error; err != nil {
		if isDuplicateEntry(err) {
			return ErrApplicationAlreadyActive
		}
		return fmt.Errorf("failed to create trainer application: %w", err)
	}
	return nil
}

// FindActiveByUserID returns the active (PENDING or APPROVED) trainer
// application owned by the given user, or ErrApplicationNotFound when the user
// does not hold one. REJECTED applications are never considered active.
func (r *trainerApplicationRepository) FindActiveByUserID(ctx context.Context, userID string) (*models.TrainerApplication, error) {
	var application models.TrainerApplication
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, []string{models.ApplicationStatusPending, models.ApplicationStatusApproved}).
		First(&application).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to find trainer application by user id: %w", err)
	}
	return &application, nil
}

// FindLatestByUserID returns the most recent trainer application owned by the
// given user, including REJECTED ones, so an applicant can be shown the outcome
// of their last application. Soft-deleted applications are never returned.
func (r *trainerApplicationRepository) FindLatestByUserID(ctx context.Context, userID string) (*models.TrainerApplication, error) {
	var application models.TrainerApplication
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC, id ASC").
		First(&application).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to find latest trainer application by user id: %w", err)
	}
	return &application, nil
}

// FindByID returns one trainer application with its linked user. Soft-deleted
// applications are never returned.
func (r *trainerApplicationRepository) FindByID(ctx context.Context, id string) (*models.TrainerApplication, error) {
	var application models.TrainerApplication
	if err := r.db.WithContext(ctx).Preload("User").First(&application, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationNotFound
		}
		return nil, fmt.Errorf("failed to find trainer application by id: %w", err)
	}
	return &application, nil
}

// List returns one page of trainer applications, optionally filtered by status
// and ordered by creation time, plus the total number of matching
// applications. The caller guarantees page >= 1 and limit >= 1. Soft-deleted
// applications are excluded by GORM's default scope.
func (r *trainerApplicationRepository) List(ctx context.Context, page, limit int, status string) ([]models.TrainerApplication, int64, error) {
	var applications []models.TrainerApplication
	var total int64

	countQuery := r.db.WithContext(ctx).Model(&models.TrainerApplication{})
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count trainer applications: %w", err)
	}

	listQuery := r.db.WithContext(ctx).Preload("User")
	if status != "" {
		listQuery = listQuery.Where("status = ?", status)
	}
	if err := listQuery.
		Order("created_at DESC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&applications).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list trainer applications: %w", err)
	}
	return applications, total, nil
}

// Approve atomically transitions a PENDING application to APPROVED and creates
// the trainer profile for its user in the same database transaction: either
// both changes are committed or neither is, so an approval can never leave an
// APPROVED application without a trainer profile or a trainer profile without
// an APPROVED application. The status transition is a single guarded UPDATE, so
// two concurrent approvals cannot both succeed: the loser observes zero
// affected rows and is reported as a state conflict. Applications that are not
// PENDING (or do not exist) are reported as a state conflict (or not found).
// When a decision is provided it is recorded inside the same transaction, so an
// approval can never exist without its audit record.
func (r *trainerApplicationRepository) Approve(ctx context.Context, applicationID string, decision *models.TrainerApplicationDecision) (*models.TrainerApplication, error) {
	var application models.TrainerApplication
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.TrainerApplication{}).
			Where("id = ? AND status = ?", applicationID, models.ApplicationStatusPending).
			Update("status", models.ApplicationStatusApproved)
		if result.Error != nil {
			return fmt.Errorf("failed to approve trainer application: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			var probe models.TrainerApplication
			if err := tx.First(&probe, "id = ?", applicationID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrApplicationNotFound
				}
				return fmt.Errorf("failed to find trainer application for approval: %w", err)
			}
			return ErrApplicationStateConflict
		}

		if err := tx.First(&application, "id = ?", applicationID).Error; err != nil {
			return fmt.Errorf("failed to load approved trainer application: %w", err)
		}

		trainer := &models.Trainer{UserID: application.UserID}
		if err := tx.Create(trainer).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrTrainerAlreadyLinked
			}
			return fmt.Errorf("failed to create trainer profile: %w", err)
		}

		if decision != nil {
			decision.ApplicationID = applicationID
			if err := tx.Create(decision).Error; err != nil {
				return fmt.Errorf("failed to record approval decision: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	application.Status = models.ApplicationStatusApproved
	return &application, nil
}

// Reject transitions a PENDING application to REJECTED and records the review
// decision in the same transaction. Rejected applications stay in history so
// the user can apply again; the user_id link and created_at are preserved. The
// status transition is a single guarded UPDATE, so two concurrent reviews
// cannot both succeed. Applications that do not exist or are not PENDING are
// reported respectively as not found or as a state conflict.
func (r *trainerApplicationRepository) Reject(ctx context.Context, applicationID string, decision *models.TrainerApplicationDecision) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.TrainerApplication{}).
			Where("id = ? AND status = ?", applicationID, models.ApplicationStatusPending).
			Updates(map[string]any{
				"status":     models.ApplicationStatusRejected,
				"updated_at": time.Now(),
			})
		if result.Error != nil {
			return fmt.Errorf("failed to reject trainer application: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			var probe models.TrainerApplication
			if err := tx.First(&probe, "id = ?", applicationID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrApplicationNotFound
				}
				return fmt.Errorf("failed to find trainer application for rejection: %w", err)
			}
			return ErrApplicationStateConflict
		}

		if decision != nil {
			decision.ApplicationID = applicationID
			if err := tx.Create(decision).Error; err != nil {
				return fmt.Errorf("failed to record rejection decision: %w", err)
			}
		}
		return nil
	})
}
