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

// ErrVersionState is returned when a version cannot be made or published in
// the programme's current state (for example an old, superseded version).
var ErrVersionState = errors.New("programme version cannot be made or published in this state")

// ProgramVersionRepository keeps what a buyer bought unchanged (v2.133.0, owner
// decision 2026-11-27): a sold package is never edited in place. Editing makes
// a new version (a full copy, as a draft); publishing it supersedes the old
// version, which leaves the market but stays readable for its buyers.
type ProgramVersionRepository interface {
	// IsSold reports whether the programme has a completed purchase.
	IsSold(ctx context.Context, programID string) (bool, error)
	// IsSoldByTrainer is IsSold limited to the trainer's own programmes.
	IsSoldByTrainer(ctx context.Context, trainerID, programID string) (bool, error)
	// OpenDraftVersion returns the unpublished new version of a programme, or
	// ErrProgramNotFound when there is none.
	OpenDraftVersion(ctx context.Context, trainerID, programID string) (*models.Program, error)
	// CreateVersion copies the programme (details, weeks, days, exercises,
	// sets and meal plan) into a new draft, or returns the open draft version.
	CreateVersion(ctx context.Context, trainerID, programID string) (*models.Program, error)
	// PublishVersion publishes a draft version and supersedes the version it
	// replaces, in one transaction; the views carry over.
	PublishVersion(ctx context.Context, trainerID, programID string, at time.Time) error
}

type programVersionRepository struct {
	db *gorm.DB
}

func NewProgramVersionRepository(db *gorm.DB) ProgramVersionRepository {
	return &programVersionRepository{db: db}
}

func (r *programVersionRepository) IsSold(ctx context.Context, programID string) (bool, error) {
	return hasCompletedPurchase(r.db.WithContext(ctx), programID)
}

func (r *programVersionRepository) IsSoldByTrainer(ctx context.Context, trainerID, programID string) (bool, error) {
	var owned int64
	if err := r.db.WithContext(ctx).Model(&models.Program{}).
		Where("id = ? AND trainer_id = ?", programID, trainerID).Count(&owned).Error; err != nil {
		return false, fmt.Errorf("failed to check programme ownership: %w", err)
	}
	if owned == 0 {
		return false, nil
	}
	return hasCompletedPurchase(r.db.WithContext(ctx), programID)
}

func openDraftVersion(tx *gorm.DB, trainerID, programID string) (*models.Program, error) {
	var draft models.Program
	err := tx.Where("previous_version_id = ? AND trainer_id = ? AND status = ? AND superseded_at IS NULL",
		programID, trainerID, models.ProgramStatusDraft).
		Order("created_at DESC").
		Take(&draft).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProgramNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the draft version: %w", err)
	}
	return &draft, nil
}

func (r *programVersionRepository) OpenDraftVersion(ctx context.Context, trainerID, programID string) (*models.Program, error) {
	return openDraftVersion(r.db.WithContext(ctx), trainerID, programID)
}

func (r *programVersionRepository) CreateVersion(ctx context.Context, trainerID, programID string) (*models.Program, error) {
	var created *models.Program
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var source models.Program
		if err := trainerProgramScope(tx.Clauses(clause.Locking{Strength: "UPDATE"}), trainerID, programID).
			Preload("Weeks", orderedByWeekNumber()).
			Preload("Weeks.Workouts", orderedByPosition()).
			Preload("Weeks.Workouts.Exercises", orderedByPosition()).
			Preload("Weeks.Workouts.Exercises.Sets", orderedBySetNumber()).
			First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to load the programme to copy: %w", err)
		}
		if source.SupersededAt != nil {
			return ErrVersionState
		}
		if existing, err := openDraftVersion(tx, trainerID, programID); err == nil {
			created = existing
			return nil
		} else if !errors.Is(err, ErrProgramNotFound) {
			return err
		}

		root := source.ID
		if source.VersionRootID != nil {
			root = *source.VersionRootID
		}
		copyProgram := source
		copyProgram.ID = ""
		copyProgram.Status = models.ProgramStatusDraft
		copyProgram.PreviousVersionID = &source.ID
		copyProgram.VersionRootID = &root
		copyProgram.ArchivedAt = nil
		copyProgram.SupersededAt = nil
		copyProgram.CreatedAt = time.Time{}
		copyProgram.UpdatedAt = time.Time{}
		copyProgram.DeletedAt = gorm.DeletedAt{}
		copyProgram.Weeks = nil
		copyProgram.CuratedNutrition = nil
		if err := tx.Omit(clause.Associations).Create(&copyProgram).Error; err != nil {
			return fmt.Errorf("failed to create the new version: %w", err)
		}

		for _, week := range source.Weeks {
			newWeek := models.ProgramWeek{ProgramID: copyProgram.ID, WeekNumber: week.WeekNumber}
			if err := tx.Omit(clause.Associations).Create(&newWeek).Error; err != nil {
				return fmt.Errorf("failed to copy a week: %w", err)
			}
			for _, workout := range week.Workouts {
				newWorkout := models.ProgramWorkout{
					ProgramWeekID: newWeek.ID, Position: workout.Position, Name: workout.Name,
					Description: workout.Description, Focus: workout.Focus,
				}
				if err := tx.Omit(clause.Associations).Create(&newWorkout).Error; err != nil {
					return fmt.Errorf("failed to copy a training day: %w", err)
				}
				for _, exercise := range workout.Exercises {
					newExercise := models.WorkoutExercise{
						ProgramWorkoutID: newWorkout.ID, ExerciseID: exercise.ExerciseID, Position: exercise.Position,
						Instructions: exercise.Instructions, Notes: exercise.Notes,
					}
					if err := tx.Omit(clause.Associations).Create(&newExercise).Error; err != nil {
						return fmt.Errorf("failed to copy an exercise: %w", err)
					}
					for _, set := range exercise.Sets {
						newSet := models.WorkoutExerciseSet{
							WorkoutExerciseID: newExercise.ID, SetNumber: set.SetNumber, Reps: set.Reps,
							WeightKg: set.WeightKg, RIR: set.RIR, RPE: set.RPE, RestSeconds: set.RestSeconds,
							Tempo: set.Tempo, SetType: set.SetType,
						}
						if err := tx.Omit(clause.Associations).Create(&newSet).Error; err != nil {
							return fmt.Errorf("failed to copy a set: %w", err)
						}
					}
				}
			}
		}

		var plan models.CuratedNutritionPlan
		err := tx.Where("program_id = ?", source.ID).
			Preload("Meals", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
			Preload("Meals.Items", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
			Take(&plan).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to load the meal plan to copy: %w", err)
		}
		if err == nil {
			plan.ID = ""
			plan.Version = 1
			plan.CreatedAt = time.Time{}
			plan.UpdatedAt = time.Time{}
			plan.DeletedAt = gorm.DeletedAt{}
			for i := range plan.Meals {
				plan.Meals[i].ID = ""
				plan.Meals[i].CreatedAt = time.Time{}
				for j := range plan.Meals[i].Items {
					plan.Meals[i].Items[j].ID = ""
					plan.Meals[i].Items[j].CreatedAt = time.Time{}
				}
			}
			if err := createCuratedNutrition(tx, copyProgram.ID, &plan); err != nil {
				return err
			}
		}
		created = &copyProgram
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *programVersionRepository) PublishVersion(ctx context.Context, trainerID, programID string, at time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var version models.Program
		if err := trainerProgramScope(tx.Clauses(clause.Locking{Strength: "UPDATE"}), trainerID, programID).
			First(&version).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to load the version: %w", err)
		}
		if version.PreviousVersionID == nil || version.Status != models.ProgramStatusDraft || version.SupersededAt != nil {
			return ErrVersionState
		}
		if err := tx.Model(&models.Program{}).Where("id = ?", version.ID).
			Update("status", models.ProgramStatusPublished).Error; err != nil {
			return fmt.Errorf("failed to publish the version: %w", err)
		}
		// The old version leaves the market (archived) and is marked superseded:
		// hidden from the trainer's lists, kept for the people who bought it.
		if err := tx.Model(&models.Program{}).
			Where("id = ? AND trainer_id = ?", *version.PreviousVersionID, trainerID).
			Updates(map[string]any{
				"superseded_at": at,
				"archived_at":   gorm.Expr("COALESCE(archived_at, ?)", at),
			}).Error; err != nil {
			return fmt.Errorf("failed to supersede the previous version: %w", err)
		}
		if err := tx.Exec(`
INSERT INTO program_view_stats (program_id, view_count, last_viewed_at)
SELECT ?, view_count, last_viewed_at FROM program_view_stats WHERE program_id = ?
ON CONFLICT (program_id) DO UPDATE
SET view_count = program_view_stats.view_count + EXCLUDED.view_count`, version.ID, *version.PreviousVersionID).Error; err != nil {
			return fmt.Errorf("failed to carry the views over: %w", err)
		}
		return nil
	})
}
