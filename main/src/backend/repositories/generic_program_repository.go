package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	// ErrGenericProgramNotFound is returned when a generic (platform-owned)
	// program does not exist, is soft-deleted, or its trainer_id is not NULL.
	ErrGenericProgramNotFound = errors.New("generic program not found")
)

// GenericProgramFilter narrows the generic program search. Empty values are
// ignored; the query matches name and description case-insensitively.
type GenericProgramFilter struct {
	Query            string
	Type             string
	Level            string
	DurationWeeks    int
	FrequencyPerWeek int
}

// GenericProgramRepository defines the data-access operations for the
// platform-owned (generic) program entity. A generic program is identified by
// trainer_id IS NULL: it is owned by no trainer and is managed exclusively by
// an authorized administrator. Every query is scoped to that NULL owner so a
// trainer-owned program can never be reached or mutated through this surface.
// Soft-deleted rows are excluded from regular queries through GORM's default
// scope.
type GenericProgramRepository interface {
	// CreateFull persists a generic program together with its full structure
	// (weeks → workouts → workout exercises → sets) in a single transaction.
	// The trainer id is always left NULL; a client-supplied trainer id is never
	// accepted.
	CreateFull(ctx context.Context, program *models.Program) error
	// UpdateFull reconciles the full structure of an existing generic program
	// with the provided desired state inside a single transaction. Children are
	// matched pairwise by their ordering key (week_number, position,
	// set_number): reused rows keep their ids and are updated in place, missing
	// rows are created, and rows beyond the desired range are soft-deleted.
	UpdateFull(ctx context.Context, programID string, program *models.Program) error
	// FindByID returns one active generic program with its complete nested
	// structure, ordered deterministically.
	FindByID(ctx context.Context, programID string) (*models.Program, error)
	// Search returns one page of active generic programs narrowed by the
	// filter, plus the total count. Results are ordered by creation time
	// (newest first) with the id as a deterministic tie-breaker.
	Search(ctx context.Context, filter GenericProgramFilter, page, limit int) ([]models.Program, int64, error)
	// SoftDelete soft-deletes one active generic program. Only the program row
	// is touched; it is never removed from the database.
	SoftDelete(ctx context.Context, programID string) error
	// Publish transitions a draft generic program to published through a
	// conditional update; missing, soft-deleted or already published programs
	// map to ErrGenericProgramNotFound.
	Publish(ctx context.Context, programID string) error
}

type genericProgramRepository struct {
	db *gorm.DB
}

func NewGenericProgramRepository(db *gorm.DB) GenericProgramRepository {
	return &genericProgramRepository{db: db}
}

// CreateFull persists the generic program and its full structure in one
// transaction. The trainer id is deliberately omitted from the INSERT so the
// platform-owned (trainer_id NULL) invariant is enforced by the database
// itself, never by trusting a client-supplied value. Nested children are
// created with their explicit parent ids so every level lands deterministically.
func (r *genericProgramRepository) CreateFull(ctx context.Context, program *models.Program) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := verifyCatalogExercises(tx, catalogExerciseIDs(program.Weeks)); err != nil {
			return err
		}
		if err := tx.Omit("TrainerID", "Weeks").Create(program).Error; err != nil {
			return fmt.Errorf("failed to create generic program: %w", err)
		}
		if err := createWeeks(tx, program.ID, program.Weeks); err != nil {
			return err
		}
		return nil
	})
}

// UpdateFull reconciles an existing generic program with the desired structure
// in one transaction. Bespoke exercises are matched pairwise by their ordering
// keys: reused children keep their row ids and are updated in place while the
// parent row itself is locked, so concurrent edits are serialized.
func (r *genericProgramRepository) UpdateFull(ctx context.Context, programID string, program *models.Program) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.Program
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND trainer_id IS NULL", programID).
			First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrGenericProgramNotFound
			}
			return fmt.Errorf("failed to verify generic program ownership: %w", err)
		}

		if err := verifyCatalogExercises(tx, catalogExerciseIDs(program.Weeks)); err != nil {
			return err
		}

		if err := tx.Model(&models.Program{}).
			Where("id = ? AND trainer_id IS NULL", programID).
			Updates(map[string]any{
				"name":               program.Name,
				"description":        program.Description,
				"type":               program.Type,
				"status":             program.Status,
				"level":              program.Level,
				"duration_weeks":     program.DurationWeeks,
				"frequency_per_week": program.FrequencyPerWeek,
				"price_minor_units":  program.PriceMinorUnits,
				"currency":           program.Currency,
			}).Error; err != nil {
			return fmt.Errorf("failed to update generic program: %w", err)
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

		return applyWeeks(tx, existingWeeks, existingWorkouts, existingExercises, existingSets, program.Weeks)
	})
}

// FindByID returns one active generic program with its full nested structure
// preloaded in deterministic order (weeks by week_number, workouts and workout
// exercises by position, sets by set_number).
func (r *genericProgramRepository) FindByID(ctx context.Context, programID string) (*models.Program, error) {
	var program models.Program
	err := r.db.WithContext(ctx).
		Where("id = ? AND trainer_id IS NULL", programID).
		Preload("Weeks", orderedWeek()).
		Preload("Weeks.Workouts", orderedByPosition()).
		Preload("Weeks.Workouts.Exercises", orderedByPosition()).
		Preload("Weeks.Workouts.Exercises.Exercise").
		Preload("Weeks.Workouts.Exercises.Sets", orderedBySetNumber()).
		First(&program).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGenericProgramNotFound
		}
		return nil, fmt.Errorf("failed to find generic program: %w", err)
	}
	return &program, nil
}

// Search returns one page of active generic programs narrowed by the filter.
// The name query uses case-insensitive LIKE with escaped wildcards against both
// name and description; empty filter values are ignored. Ordering is
// deterministic: newest first, then id.
func (r *genericProgramRepository) Search(ctx context.Context, filter GenericProgramFilter, page, limit int) ([]models.Program, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Program{}).Where("trainer_id IS NULL")

	if strings.TrimSpace(filter.Query) != "" {
		escaped := escapeSQLLike(strings.TrimSpace(filter.Query))
		query = query.Where("(name LIKE ? OR description LIKE ?)", "%"+escaped+"%", "%"+escaped+"%")
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Level != "" {
		query = query.Where("level = ?", filter.Level)
	}
	if filter.DurationWeeks > 0 {
		query = query.Where("duration_weeks = ?", filter.DurationWeeks)
	}
	if filter.FrequencyPerWeek > 0 {
		query = query.Where("frequency_per_week = ?", filter.FrequencyPerWeek)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count generic programs: %w", err)
	}

	var programs []models.Program
	if err := query.
		Order("created_at DESC, id ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&programs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list generic programs: %w", err)
	}
	return programs, total, nil
}

// SoftDelete soft-deletes one active generic program. Only the program row is
// touched; children become unreachable through the program scope and are never
// removed. An unknown, soft-deleted or trainer-owned program is
// indistinguishable and maps to ErrGenericProgramNotFound.
func (r *genericProgramRepository) SoftDelete(ctx context.Context, programID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND trainer_id IS NULL", programID).
		Delete(&models.Program{})
	if result.Error != nil {
		return fmt.Errorf("failed to soft delete generic program: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrGenericProgramNotFound
	}
	return nil
}

// Publish transitions a draft generic program to published through a
// conditional update scoped to the generic (trainer_id IS NULL) state. The
// WHERE clause enforces status = 'draft' so the operation is idempotent and
// race-safe; a zero RowsAffected result maps to ErrGenericProgramNotFound,
// which covers missing, soft-deleted, foreign and already-published in a single
// indistinguishable domain error that the caller may disambiguate.
func (r *genericProgramRepository) Publish(ctx context.Context, programID string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Program{}).
		Where("id = ? AND trainer_id IS NULL AND status = ?", programID, models.ProgramStatusDraft).
		Update("status", models.ProgramStatusPublished)
	if result.Error != nil {
		return fmt.Errorf("failed to publish generic program: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrGenericProgramNotFound
	}
	return nil
}

// genericTreeIndex gathers every existing child into maps keyed by their parent
// id, in deterministic order, so the update diff can match rows pairwise.
type genericTreeIndex struct {
	workoutsByWeek     map[string][]models.ProgramWorkout
	exercisesByWorkout map[string][]models.WorkoutExercise
	setsByExercise     map[string][]models.WorkoutExerciseSet
}

func applyWeeks(tx *gorm.DB, existingWeeks []models.ProgramWeek, existingWorkouts []models.ProgramWorkout, existingExercises []models.WorkoutExercise, existingSets []models.WorkoutExerciseSet, desired []models.ProgramWeek) error {
	index := &genericTreeIndex{
		workoutsByWeek:     groupByWorkouts(existingWorkouts),
		exercisesByWorkout: groupByExercises(existingExercises),
		setsByExercise:     groupBySets(existingSets),
	}

	for i := range desired {
		var week *models.ProgramWeek
		reuse := true
		if i < len(existingWeeks) {
			week = &existingWeeks[i]
		} else {
			week = &desired[i]
			week.WeekNumber = i + 1
			reuse = false
		}
		week.ProgramID = entityWeekProgramID(existingWeeks, desired[i])
		week.WeekNumber = i + 1

		if reuse {
			if err := tx.Model(&models.ProgramWeek{}).
				Where("id = ?", week.ID).
				Update("week_number", i+1).Error; err != nil {
				return fmt.Errorf("failed to update generic program week: %w", err)
			}
		} else {
			if err := tx.Omit("Workouts").Create(week).Error; err != nil {
				return fmt.Errorf("failed to create generic program week: %w", err)
			}
		}

		existingWorkoutList := index.workoutsByWeek[week.ID]
		if err := applyWorkouts(tx, week.ID, existingWorkoutList, index, desired[i].Workouts); err != nil {
			return err
		}
	}

	for i := len(desired); i < len(existingWeeks); i++ {
		if err := tx.Delete(&models.ProgramWeek{}, "id = ?", existingWeeks[i].ID).Error; err != nil {
			return fmt.Errorf("failed to remove generic program week: %w", err)
		}
	}
	return nil
}

func applyWorkouts(tx *gorm.DB, weekID string, existing []models.ProgramWorkout, index *genericTreeIndex, desired []models.ProgramWorkout) error {
	for i := range desired {
		var workout *models.ProgramWorkout
		reuse := true
		if i < len(existing) {
			workout = &existing[i]
		} else {
			workout = &desired[i]
			workout.Position = i + 1
			reuse = false
		}
		workout.ProgramWeekID = weekID
		workout.Position = i + 1

		if reuse {
			if err := tx.Model(&models.ProgramWorkout{}).
				Where("id = ?", workout.ID).
				Update("position", i+1).Error; err != nil {
				return fmt.Errorf("failed to update generic program workout: %w", err)
			}
		} else {
			if err := tx.Omit("Exercises").Create(workout).Error; err != nil {
				return fmt.Errorf("failed to create generic program workout: %w", err)
			}
		}

		existingExerciseList := index.exercisesByWorkout[workout.ID]
		if err := applyExercises(tx, workout.ID, existingExerciseList, index, desired[i].Exercises); err != nil {
			return err
		}
	}

	for i := len(desired); i < len(existing); i++ {
		if err := tx.Delete(&models.ProgramWorkout{}, "id = ?", existing[i].ID).Error; err != nil {
			return fmt.Errorf("failed to remove generic program workout: %w", err)
		}
	}
	return nil
}

func applyExercises(tx *gorm.DB, workoutID string, existing []models.WorkoutExercise, index *genericTreeIndex, desired []models.WorkoutExercise) error {
	for i := range desired {
		var exercise *models.WorkoutExercise
		reuse := true
		if i < len(existing) {
			exercise = &existing[i]
		} else {
			exercise = &desired[i]
			exercise.Position = i + 1
			reuse = false
		}
		exercise.ProgramWorkoutID = workoutID
		exercise.Position = i + 1

		if reuse {
			if err := tx.Model(&models.WorkoutExercise{}).
				Where("id = ?", exercise.ID).
				Updates(map[string]any{
					"position":     i + 1,
					"exercise_id":  exercise.ExerciseID,
					"instructions": exercise.Instructions,
					"notes":        exercise.Notes,
				}).Error; err != nil {
				return fmt.Errorf("failed to update generic workout exercise: %w", err)
			}
		} else {
			if err := tx.Omit("Sets", "Exercise").Create(exercise).Error; err != nil {
				return fmt.Errorf("failed to create generic workout exercise: %w", err)
			}
		}

		existingSetList := index.setsByExercise[exercise.ID]
		if err := applySets(tx, exercise.ID, existingSetList, desired[i].Sets); err != nil {
			return err
		}
	}

	for i := len(desired); i < len(existing); i++ {
		if err := tx.Delete(&models.WorkoutExercise{}, "id = ?", existing[i].ID).Error; err != nil {
			return fmt.Errorf("failed to remove generic workout exercise: %w", err)
		}
	}
	return nil
}

func applySets(tx *gorm.DB, exerciseID string, existing []models.WorkoutExerciseSet, desired []models.WorkoutExerciseSet) error {
	for i := range desired {
		var set *models.WorkoutExerciseSet
		reuse := true
		if i < len(existing) {
			set = &existing[i]
		} else {
			set = &desired[i]
			set.SetNumber = i + 1
			reuse = false
		}
		set.WorkoutExerciseID = exerciseID
		set.SetNumber = i + 1

		if reuse {
			if err := tx.Model(&models.WorkoutExerciseSet{}).
				Where("id = ?", set.ID).
				Updates(map[string]any{
					"set_number":   i + 1,
					"set_type":     set.SetType,
					"reps":         set.Reps,
					"weight_kg":    set.WeightKg,
					"rir":          set.RIR,
					"rpe":          set.RPE,
					"rest_seconds": set.RestSeconds,
					"tempo":        set.Tempo,
				}).Error; err != nil {
				return fmt.Errorf("failed to update generic workout exercise set: %w", err)
			}
		} else {
			if err := tx.Create(set).Error; err != nil {
				return fmt.Errorf("failed to create generic workout exercise set: %w", err)
			}
		}
	}

	for i := len(desired); i < len(existing); i++ {
		if err := tx.Delete(&models.WorkoutExerciseSet{}, "id = ?", existing[i].ID).Error; err != nil {
			return fmt.Errorf("failed to remove generic workout exercise set: %w", err)
		}
	}
	return nil
}

func createWeeks(tx *gorm.DB, programID string, weeks []models.ProgramWeek) error {
	for i := range weeks {
		week := &weeks[i]
		week.ProgramID = programID
		if err := tx.Omit("Workouts").Create(week).Error; err != nil {
			return fmt.Errorf("failed to create generic program week: %w", err)
		}
		if err := createWorkouts(tx, week.ID, week.Workouts); err != nil {
			return err
		}
	}
	return nil
}

func createWorkouts(tx *gorm.DB, weekID string, workouts []models.ProgramWorkout) error {
	for i := range workouts {
		workout := &workouts[i]
		workout.ProgramWeekID = weekID
		if err := tx.Omit("Exercises").Create(workout).Error; err != nil {
			return fmt.Errorf("failed to create generic program workout: %w", err)
		}
		if err := createExercises(tx, workout.ID, workout.Exercises); err != nil {
			return err
		}
	}
	return nil
}

func createExercises(tx *gorm.DB, workoutID string, exercises []models.WorkoutExercise) error {
	for i := range exercises {
		exercise := &exercises[i]
		exercise.ProgramWorkoutID = workoutID
		if err := tx.Omit("Sets", "Exercise").Create(exercise).Error; err != nil {
			return fmt.Errorf("failed to create generic workout exercise: %w", err)
		}
		if err := createSets(tx, exercise.ID, exercise.Sets); err != nil {
			return err
		}
	}
	return nil
}

func createSets(tx *gorm.DB, exerciseID string, sets []models.WorkoutExerciseSet) error {
	for i := range sets {
		set := &sets[i]
		set.WorkoutExerciseID = exerciseID
		if err := tx.Create(set).Error; err != nil {
			return fmt.Errorf("failed to create generic workout exercise set: %w", err)
		}
	}
	return nil
}

func loadWeeksForProgram(tx *gorm.DB, programID string) ([]models.ProgramWeek, error) {
	var weeks []models.ProgramWeek
	if err := tx.Where("program_id = ?", programID).
		Order("week_number ASC, id ASC").
		Find(&weeks).Error; err != nil {
		return nil, fmt.Errorf("failed to load generic program weeks: %w", err)
	}
	return weeks, nil
}

func loadWorkoutsForWeeks(tx *gorm.DB, weekIDs []string) ([]models.ProgramWorkout, error) {
	var workouts []models.ProgramWorkout
	if len(weekIDs) == 0 {
		return workouts, nil
	}
	if err := tx.Where("program_week_id IN ?", weekIDs).
		Order("position ASC, id ASC").
		Find(&workouts).Error; err != nil {
		return nil, fmt.Errorf("failed to load generic program workouts: %w", err)
	}
	return workouts, nil
}

func loadExercisesForWorkouts(tx *gorm.DB, workoutIDs []string) ([]models.WorkoutExercise, error) {
	var exercises []models.WorkoutExercise
	if len(workoutIDs) == 0 {
		return exercises, nil
	}
	if err := tx.Where("program_workout_id IN ?", workoutIDs).
		Order("position ASC, id ASC").
		Find(&exercises).Error; err != nil {
		return nil, fmt.Errorf("failed to load generic workout exercises: %w", err)
	}
	return exercises, nil
}

func loadSetsForExercises(tx *gorm.DB, exerciseIDs []string) ([]models.WorkoutExerciseSet, error) {
	var sets []models.WorkoutExerciseSet
	if len(exerciseIDs) == 0 {
		return sets, nil
	}
	if err := tx.Where("workout_exercise_id IN ?", exerciseIDs).
		Order("set_number ASC, id ASC").
		Find(&sets).Error; err != nil {
		return nil, fmt.Errorf("failed to load generic workout exercise sets: %w", err)
	}
	return sets, nil
}

func weekIDs(weeks []models.ProgramWeek) []string {
	ids := make([]string, 0, len(weeks))
	for i := range weeks {
		ids = append(ids, weeks[i].ID)
	}
	return ids
}

func workoutIDs(workouts []models.ProgramWorkout) []string {
	ids := make([]string, 0, len(workouts))
	for i := range workouts {
		ids = append(ids, workouts[i].ID)
	}
	return ids
}

func exerciseIDs(exercises []models.WorkoutExercise) []string {
	ids := make([]string, 0, len(exercises))
	for i := range exercises {
		ids = append(ids, exercises[i].ID)
	}
	return ids
}

// catalogExerciseIDs gathers every referenced catalogue exercise id of the
// desired tree, deduplicated, so existence can be verified before any write.
func catalogExerciseIDs(weeks []models.ProgramWeek) []string {
	seen := make(map[string]struct{})
	var ids []string
	for i := range weeks {
		for k := range weeks[i].Workouts {
			for j := range weeks[i].Workouts[k].Exercises {
				id := weeks[i].Workouts[k].Exercises[j].ExerciseID
				if _, exists := seen[id]; exists {
					continue
				}
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// verifyCatalogExercises ensures every referenced catalogue exercise exists and
// is not soft-deleted. Referencing an exercise that does not exist (or was
// removed from the global catalogue) is a hard error so a generic program can
// never dangle references to exercises masters can no longer see.
func verifyCatalogExercises(tx *gorm.DB, exerciseIDs []string) error {
	if len(exerciseIDs) == 0 {
		return nil
	}
	var count int64
	if err := tx.Model(&models.Exercise{}).
		Where("id IN ?", exerciseIDs).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to verify catalogue exercises: %w", err)
	}
	if count != int64(len(exerciseIDs)) {
		return ErrExerciseNotFound
	}
	return nil
}

func groupByWorkouts(workouts []models.ProgramWorkout) map[string][]models.ProgramWorkout {
	grouped := make(map[string][]models.ProgramWorkout)
	for i := range workouts {
		weekID := workouts[i].ProgramWeekID
		grouped[weekID] = append(grouped[weekID], workouts[i])
	}
	return grouped
}

func groupByExercises(exercises []models.WorkoutExercise) map[string][]models.WorkoutExercise {
	grouped := make(map[string][]models.WorkoutExercise)
	for i := range exercises {
		workoutID := exercises[i].ProgramWorkoutID
		grouped[workoutID] = append(grouped[workoutID], exercises[i])
	}
	return grouped
}

func groupBySets(sets []models.WorkoutExerciseSet) map[string][]models.WorkoutExerciseSet {
	grouped := make(map[string][]models.WorkoutExerciseSet)
	for i := range sets {
		exerciseID := sets[i].WorkoutExerciseID
		grouped[exerciseID] = append(grouped[exerciseID], sets[i])
	}
	return grouped
}

// entityWeekProgramID resolves the parent program id of a desired week. The
// desired week already carries its program id (set by the service); the
// fallback keeps the diff immune to a missing value.
func entityWeekProgramID(existingWeeks []models.ProgramWeek, desired models.ProgramWeek) string {
	if desired.ProgramID != "" {
		return desired.ProgramID
	}
	if len(existingWeeks) > 0 {
		return existingWeeks[0].ProgramID
	}
	return ""
}

// orderedWeek returns a GORM preload clause ordering weeks deterministically.
func orderedWeek() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order("week_number ASC, id ASC")
	}
}

// orderedByPosition returns a GORM preload clause ordering workouts and workout
// exercises deterministically.
func orderedByPosition() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC, id ASC")
	}
}

// orderedBySetNumber returns a GORM preload clause ordering sets
// deterministically.
func orderedBySetNumber() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order("set_number ASC, id ASC")
	}
}
