package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

// TrainerStructureRepository reads and replaces the whole structure (weeks,
// training days, exercises and sets) of one trainer-owned plan, so the trainer
// builder can save a plan in one step like the platform builder. Every query
// is scoped by both the program id and the owning trainer id.
type TrainerStructureRepository interface {
	FindTrainerProgramTree(ctx context.Context, trainerID, programID string) (*models.Program, error)
	ReplaceTrainerStructure(ctx context.Context, trainerID, programID string, weeks []models.ProgramWeek) error
}

type trainerStructureRepository struct {
	db *gorm.DB
}

func NewTrainerStructureRepository(db *gorm.DB) TrainerStructureRepository {
	return &trainerStructureRepository{db: db}
}

func trainerProgramScope(db *gorm.DB, trainerID, programID string) *gorm.DB {
	return db.Where("id = ?", programID).Where("trainer_id = ?", trainerID)
}

// FindTrainerProgramTree loads one active trainer-owned plan with its full
// structure in deterministic order.
func (r *trainerStructureRepository) FindTrainerProgramTree(ctx context.Context, trainerID, programID string) (*models.Program, error) {
	var program models.Program
	err := trainerProgramScope(r.db.WithContext(ctx), trainerID, programID).
		Preload("Weeks", orderedWeek()).
		Preload("Weeks.Workouts", orderedByPosition()).
		Preload("Weeks.Workouts.Exercises", orderedByPosition()).
		Preload("Weeks.Workouts.Exercises.Exercise").
		Preload("Weeks.Workouts.Exercises.Sets", orderedBySetNumber()).
		First(&program).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to find trainer program: %w", err)
	}
	return &program, nil
}

// ReplaceTrainerStructure reconciles the structure of one trainer-owned plan
// with the desired tree in one transaction. The plan row is locked first, so
// concurrent saves are serialized; reused rows keep their ids (workout history
// keeps pointing at the same training days) and surplus rows are soft-deleted.
func (r *trainerStructureRepository) ReplaceTrainerStructure(ctx context.Context, trainerID, programID string, weeks []models.ProgramWeek) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.Program
		if err := trainerProgramScope(tx.Clauses(clause.Locking{Strength: "UPDATE"}), trainerID, programID).
			First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to verify trainer program ownership: %w", err)
		}
		if err := verifyCatalogExercises(tx, catalogExerciseIDs(weeks)); err != nil {
			return err
		}
		for i := range weeks {
			weeks[i].ProgramID = programID
		}

		existingWeeks, err := loadWeeksForProgram(tx, programID)
		if err != nil {
			return err
		}
		existingWorkouts, err := loadWorkoutsForWeeks(tx, weekIDs(existingWeeks))
		if err != nil {
			return err
		}
		existingExercises, err := loadExercisesForWorkouts(tx, workoutIDs(existingWorkouts))
		if err != nil {
			return err
		}
		existingSets, err := loadSetsForExercises(tx, exerciseIDs(existingExercises))
		if err != nil {
			return err
		}
		if err := applyWeeks(tx, existingWeeks, existingWorkouts, existingExercises, existingSets, weeks); err != nil {
			return err
		}
		if err := tx.Model(&models.Program{}).Where("id = ?", programID).
			Update("content_version", gorm.Expr("content_version + 1")).Error; err != nil {
			return fmt.Errorf("failed to update trainer program version: %w", err)
		}
		return nil
	})
}
