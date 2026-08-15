package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

var (
	// ErrAssignmentNotFound indicates the program assignment does not exist, is
	// soft-deleted, does not belong to the requested trainer/client pair or the
	// owning trainer-client relationship is not active.
	ErrAssignmentNotFound = errors.New("program assignment not found")
	// ErrAssignmentAlreadyActive indicates an active assignment already exists
	// for the same trainer and client. The one-active-assignment rule is
	// enforced at the database level by the unique index on the active
	// assignment identifier.
	ErrAssignmentAlreadyActive = errors.New("active program assignment already exists")
)

// ProgramAssignmentRepository defines the data-access operations for the
// trainer → client program assignment entity. Every operation receives the
// trainer id explicitly; the repository never obtains it from an HTTP context,
// so a client-supplied trainer id can never influence a query. Every operation
// depends on the trainer-client relationship being active and on the program
// being owned by the trainer. Soft-deleted assignments are excluded from
// regular queries through GORM's default scope.
type ProgramAssignmentRepository interface {
	Create(ctx context.Context, trainerID, userID, programID string, assignment *models.ProgramAssignment) error
	ListByClient(ctx context.Context, trainerID, userID string) ([]models.ProgramAssignment, error)
	FindByIDAndClient(ctx context.Context, trainerID, userID, assignmentID string) (*models.ProgramAssignment, error)
	FindAssignedProgram(ctx context.Context, userID string) (*models.Program, error)
	SoftDelete(ctx context.Context, trainerID, userID, assignmentID string) error
}

type programAssignmentRepository struct {
	db *gorm.DB
}

func NewProgramAssignmentRepository(db *gorm.DB) ProgramAssignmentRepository {
	return &programAssignmentRepository{db: db}
}

// Create assigns one of the trainer's own active programs to one of the
// trainer's active clients. The trainer-client relationship must be active, the
// program must exist, be active and be owned by the trainer, and no active
// assignment may already exist for the same (trainer, client) pair; the unique
// active_assignment index is the final backstop. The trainer id is always set
// by the caller (the authenticated trainer context) and never accepted from
// the client. The assigned program is loaded back onto the row so the caller
// can render the safe program data.
func (r *programAssignmentRepository) Create(ctx context.Context, trainerID, userID, programID string, assignment *models.ProgramAssignment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, "id = ?", userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClientRelationNotFound
			}
			return fmt.Errorf("failed to verify client user: %w", err)
		}

		var relation models.TrainerClient
		if err := tx.First(&relation, "trainer_id = ? AND user_id = ?", trainerID, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrClientRelationNotFound
			}
			return fmt.Errorf("failed to verify client relationship: %w", err)
		}

		var program models.Program
		if err := tx.First(&program, "id = ? AND trainer_id = ?", programID, trainerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to verify program ownership: %w", err)
		}

		var existing models.ProgramAssignment
		if err := tx.Where("trainer_id = ? AND user_id = ?", trainerID, userID).First(&existing).Error; err == nil {
			return ErrAssignmentAlreadyActive
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to check existing assignment: %w", err)
		}

		assignment.TrainerID = trainerID
		assignment.ProgramID = programID
		assignment.UserID = userID
		if err := tx.Create(assignment).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrAssignmentAlreadyActive
			}
			return fmt.Errorf("failed to create program assignment: %w", err)
		}
		if err := tx.Model(assignment).Association("Program").Find(&assignment.Program); err != nil {
			return fmt.Errorf("failed to load the assigned program: %w", err)
		}
		return nil
	})
}

// ListByClient returns every active assignment of one active client of the
// trainer, in chronological order, each with its safe program data preloaded.
// The query is scoped by the active trainer-client relationship, so a client
// that does not exist, is not managed by the trainer or whose relationship was
// soft-deleted is indistinguishable from a client without assignments and
// yields an empty list. A soft-deleted program keeps being returned with its
// data intact: historical assignments are never destroyed by a program
// soft-delete.
func (r *programAssignmentRepository) ListByClient(ctx context.Context, trainerID, userID string) ([]models.ProgramAssignment, error) {
	var assignments []models.ProgramAssignment
	if err := r.db.WithContext(ctx).
		Where("trainer_id = ? AND user_id IN (?)", trainerID, trainerScopedClientID(r.db, trainerID, userID)).
		Preload("Program", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("created_at ASC, id ASC").
		Find(&assignments).Error; err != nil {
		return nil, fmt.Errorf("failed to list program assignments: %w", err)
	}
	return assignments, nil
}

// FindByIDAndClient returns one active assignment scoped by the trainer, the
// client and the assignment id, with its safe program data preloaded. An
// assignment that is missing, soft-deleted, belongs to another client or whose
// trainer-client relationship is not active is indistinguishable and never
// revealed.
func (r *programAssignmentRepository) FindByIDAndClient(ctx context.Context, trainerID, userID, assignmentID string) (*models.ProgramAssignment, error) {
	var assignment models.ProgramAssignment
	if err := r.db.WithContext(ctx).
		Where("id = ? AND trainer_id = ? AND user_id IN (?)", assignmentID, trainerID, trainerScopedClientID(r.db, trainerID, userID)).
		Preload("Program", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAssignmentNotFound
		}
		return nil, fmt.Errorf("failed to find program assignment: %w", err)
	}
	return &assignment, nil
}

// FindAssignedProgram returns the program of the authenticated user's most
// recent active assignment, with its full active structure (weeks, workouts,
// workout exercises and catalog exercises) preloaded in display order. The
// query is scoped solely by the user id resolved from the authentication
// context; the assignment itself is the grant, so no trainer-client
// relationship is checked and no program status filter is applied (publishing
// carries no access semantics). A user can be the client of several trainers
// and hold several active assignments, so the most recently created one wins.
// Soft-deleted assignments, programs, weeks, workouts and workout exercises are
// excluded through GORM's default scope. A user without an active assignment
// maps to ErrAssignmentNotFound; an assignment whose program was soft-deleted
// maps to ErrProgramNotFound.
func (r *programAssignmentRepository) FindAssignedProgram(ctx context.Context, userID string) (*models.Program, error) {
	var assignment models.ProgramAssignment
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("Program", func(db *gorm.DB) *gorm.DB {
			return db.
				Preload("Weeks", func(db *gorm.DB) *gorm.DB { return db.Order("week_number ASC, id ASC") }).
				Preload("Weeks.Workouts", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC, id ASC") }).
				Preload("Weeks.Workouts.Exercises", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC, id ASC") }).
				Preload("Weeks.Workouts.Exercises.Exercise")
		}).
		Order("created_at DESC, id DESC").
		First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAssignmentNotFound
		}
		return nil, fmt.Errorf("failed to find the assigned program: %w", err)
	}

	if assignment.Program.ID == "" {
		return nil, ErrProgramNotFound
	}
	return &assignment.Program, nil
}

// SoftDelete soft-deletes one of the trainer's own client assignments. Only the
// assignment row is touched and it is never removed; the program and the
// trainer-client relationship are never touched. The soft-deleted assignment
// simply disappears from all regular queries.
func (r *programAssignmentRepository) SoftDelete(ctx context.Context, trainerID, userID, assignmentID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND trainer_id = ? AND user_id IN (?)", assignmentID, trainerID, trainerScopedClientID(r.db, trainerID, userID)).
		Delete(&models.ProgramAssignment{})
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete program assignment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAssignmentNotFound
	}
	return nil
}

// trainerScopedClientID builds a subquery selecting the id of the active
// client managed by the trainer. It keeps every assignment query scoped by the
// trainer and by the active trainer-client relationship (both the relationship
// and the client user must be active, not soft-deleted) without trusting any
// client-supplied identity.
func trainerScopedClientID(db *gorm.DB, trainerID, userID string) *gorm.DB {
	return db.Model(&models.TrainerClient{}).Select("user_id").
		Where("trainer_id = ? AND user_id IN (?)", trainerID,
			db.Model(&models.User{}).Select("id").Where("id = ?", userID))
}
