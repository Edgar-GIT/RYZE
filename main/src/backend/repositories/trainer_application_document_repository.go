package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	// ErrApplicationDocumentNotFound indicates the requested evidence document
	// does not exist or is soft-deleted.
	ErrApplicationDocumentNotFound = errors.New("trainer application document not found")
	// ErrApplicationDocumentAlreadyRemoved indicates the document exists but
	// is already soft-deleted.
	ErrApplicationDocumentAlreadyRemoved = errors.New("trainer application document is already removed")
)

// TrainerApplicationDocumentRepository defines the data-access operations for
// trainer-application evidence documents. The binary content is owned by the
// evidence store; this repository only manages the validated metadata rows.
type TrainerApplicationDocumentRepository interface {
	Create(ctx context.Context, document *models.TrainerApplicationDocument) error
	ListByApplication(ctx context.Context, applicationID string) ([]models.TrainerApplicationDocument, error)
	// FindByID returns one active document or ErrApplicationDocumentNotFound.
	FindByID(ctx context.Context, documentID string) (*models.TrainerApplicationDocument, error)
	// SoftDelete marks one active document as removed. The stored file is kept
	// so a withdrawn upload never destroys the review trail. The operation is
	// guarded by deleted_at IS NULL, so an already-removed document can never
	// be removed again.
	SoftDelete(ctx context.Context, documentID string) error
}

type trainerApplicationDocumentRepository struct {
	db *gorm.DB
}

func NewTrainerApplicationDocumentRepository(db *gorm.DB) TrainerApplicationDocumentRepository {
	return &trainerApplicationDocumentRepository{db: db}
}

func (r *trainerApplicationDocumentRepository) Create(ctx context.Context, document *models.TrainerApplicationDocument) error {
	if err := r.db.WithContext(ctx).Create(document).Error; err != nil {
		return fmt.Errorf("failed to create trainer application document: %w", err)
	}
	return nil
}

func (r *trainerApplicationDocumentRepository) ListByApplication(ctx context.Context, applicationID string) ([]models.TrainerApplicationDocument, error) {
	var documents []models.TrainerApplicationDocument
	if err := r.db.WithContext(ctx).
		Where("application_id = ?", applicationID).
		Order("created_at ASC, id ASC").
		Find(&documents).Error; err != nil {
		return nil, fmt.Errorf("failed to list trainer application documents: %w", err)
	}
	return documents, nil
}

func (r *trainerApplicationDocumentRepository) FindByID(ctx context.Context, documentID string) (*models.TrainerApplicationDocument, error) {
	var document models.TrainerApplicationDocument
	if err := r.db.WithContext(ctx).First(&document, "id = ?", documentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrApplicationDocumentNotFound
		}
		return nil, fmt.Errorf("failed to find trainer application document: %w", err)
	}
	return &document, nil
}

func (r *trainerApplicationDocumentRepository) SoftDelete(ctx context.Context, documentID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", documentID).
		Delete(&models.TrainerApplicationDocument{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove trainer application document: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		// The guarded delete only matches an active row. Distinguish "already
		// removed" from "never existed" with an unscoped probe, so the caller
		// can tell an idempotent repeat from a stale reference without ever
		// exposing a soft-deleted row through the active-scoped reads.
		var probe models.TrainerApplicationDocument
		if err := r.db.WithContext(ctx).Unscoped().First(&probe, "id = ?", documentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrApplicationDocumentNotFound
			}
			return fmt.Errorf("failed to find trainer application document: %w", err)
		}
		return ErrApplicationDocumentAlreadyRemoved
	}
	return nil
}
