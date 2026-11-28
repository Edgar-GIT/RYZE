package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

// MaxArchivedPrograms is how many programmes one owner (a trainer, or the
// platform) may keep archived at once (owner request, 2026-11-27).
const MaxArchivedPrograms = 15

var (
	// ErrArchiveLimit: the owner already has MaxArchivedPrograms archived.
	ErrArchiveLimit = errors.New("archive limit reached")
	// ErrArchiveState: only a published, active programme is archived, and
	// only an archived one is restored.
	ErrArchiveState = errors.New("programme cannot change archive state")
)

// ProgramOwner identifies whose programme it is: a trainer, or the platform
// (an empty TrainerID) within one catalogue family.
type ProgramOwner struct {
	TrainerID   string
	ProductType string
}

func (o ProgramOwner) scope(tx *gorm.DB) *gorm.DB {
	if o.TrainerID != "" {
		return tx.Where("trainer_id = ?", o.TrainerID)
	}
	return tx.Where("trainer_id IS NULL AND product_type = ?", o.ProductType)
}

// ProgramArchiveRepository archives and restores programmes (migration 000072).
type ProgramArchiveRepository interface {
	Archive(ctx context.Context, owner ProgramOwner, programID string, at time.Time) (*models.Program, error)
	Restore(ctx context.Context, owner ProgramOwner, programID string) (*models.Program, error)
	// CountArchived counts the owner's archived programmes (the platform's
	// across every family).
	CountArchived(ctx context.Context, trainerID string) (int64, error)
}

type programArchiveRepository struct {
	db *gorm.DB
}

func NewProgramArchiveRepository(db *gorm.DB) ProgramArchiveRepository {
	return &programArchiveRepository{db: db}
}

func countArchived(tx *gorm.DB, trainerID string) (int64, error) {
	// A superseded old version is off the market too, but it is not one of the
	// owner's archived programmes (v2.133.0).
	query := tx.Model(&models.Program{}).Where("archived_at IS NOT NULL AND superseded_at IS NULL")
	if trainerID != "" {
		query = query.Where("trainer_id = ?", trainerID)
	} else {
		query = query.Where("trainer_id IS NULL")
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count archived programmes: %w", err)
	}
	return count, nil
}

func (r *programArchiveRepository) CountArchived(ctx context.Context, trainerID string) (int64, error) {
	return countArchived(r.db.WithContext(ctx), trainerID)
}

func (r *programArchiveRepository) Archive(ctx context.Context, owner ProgramOwner, programID string, at time.Time) (*models.Program, error) {
	var program models.Program
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// One owner archives one programme at a time, so the limit holds even
		// with two archive requests at once.
		if owner.TrainerID != "" {
			if err := tx.Exec(`SELECT 1 FROM trainers WHERE id = ? FOR UPDATE`, owner.TrainerID).Error; err != nil {
				return fmt.Errorf("failed to lock the owner: %w", err)
			}
		} else if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext('ryze-platform-archive'))`).Error; err != nil {
			return fmt.Errorf("failed to lock the platform archive: %w", err)
		}
		if err := owner.scope(tx.Clauses(clause.Locking{Strength: "UPDATE"})).First(&program, "id = ?", programID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to load programme: %w", err)
		}
		if program.Status != models.ProgramStatusPublished || program.ArchivedAt != nil || program.SupersededAt != nil {
			return ErrArchiveState
		}
		count, err := countArchived(tx, owner.TrainerID)
		if err != nil {
			return err
		}
		if count >= MaxArchivedPrograms {
			return ErrArchiveLimit
		}
		program.ArchivedAt = &at
		return tx.Model(&models.Program{}).Where("id = ?", programID).Update("archived_at", at).Error
	})
	if err != nil {
		return nil, err
	}
	return &program, nil
}

func (r *programArchiveRepository) Restore(ctx context.Context, owner ProgramOwner, programID string) (*models.Program, error) {
	var program models.Program
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := owner.scope(tx.Clauses(clause.Locking{Strength: "UPDATE"})).First(&program, "id = ?", programID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to load programme: %w", err)
		}
		// An old version replaced by a newer one never returns to the market.
		if program.ArchivedAt == nil || program.SupersededAt != nil {
			return ErrArchiveState
		}
		program.ArchivedAt = nil
		return tx.Model(&models.Program{}).Where("id = ?", programID).Update("archived_at", nil).Error
	})
	if err != nil {
		return nil, err
	}
	return &program, nil
}
