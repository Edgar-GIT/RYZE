package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

// ErrNutritionPlanNotFound indicates no plan exists for the requested scope. It
// covers "no plan yet", "no plan at all" and "not your plan" alike, so a read can
// never be used to discover which programs or accounts exist.
var ErrNutritionPlanNotFound = errors.New("nutrition plan not found")

// NutritionPlanRepository is the data-access surface for generated nutrition
// plans.
//
// Every operation receives the owner identity explicitly; the repository never
// reads it from a request context, so a client-supplied user id can never
// influence a query.
type NutritionPlanRepository interface {
	// Complete inserts a plan version for an assignment that has been claimed for
	// generation, supersedes the previous active version and marks the assignment
	// completed — all in one transaction.
	//
	// It is the only path that writes a plan, and it re-reads the assignment under
	// a row lock first. That lock is what makes concurrent generation safe: the
	// loser of a race cannot claim the assignment, and a completion that arrives
	// after the intake moved on is rejected instead of claiming to be current for a
	// revision it never read.
	Complete(ctx context.Context, userID, programID string, plan *models.NutritionPlan) error
	// FindActiveByAssignment returns the active plan of an assignment with its
	// meals, foods and exclusions populated.
	FindActiveByAssignment(ctx context.Context, assignmentID string) (*models.NutritionPlan, error)
	// FindActiveByUserAndProgram returns the active plan for an owner and program.
	// It is the read the access surface uses, and it filters on the owner in the
	// same query that selects the active plan.
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionPlan, error)
	// ListVersions returns every plan version of an assignment, newest first. It
	// exists for auditing and administration, not for client delivery.
	ListVersions(ctx context.Context, assignmentID string) ([]models.NutritionPlan, error)
}

type nutritionPlanRepository struct {
	db *gorm.DB
}

func NewNutritionPlanRepository(db *gorm.DB) NutritionPlanRepository {
	return &nutritionPlanRepository{db: db}
}

// Complete writes a plan version and finishes the assignment in one transaction.
//
// The transaction covers four effects that must never be observed separately: the
// new version's rows, the superseding of the previous active version, the
// assignment's completion, and the assignment's version bump. If any of them
// failed and were applied alone, the aggregate would be inconsistent — a completed
// assignment with no readable plan, or two active plans competing for delivery.
func (r *nutritionPlanRepository) Complete(ctx context.Context, userID, programID string, plan *models.NutritionPlan) error {
	if plan == nil {
		return fmt.Errorf("nutrition plan is required")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// The assignment is locked for the duration so a second generator cannot
		// read a stale status and race this write.
		var assignment models.NutritionAssignment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND program_id = ?", userID, programID).
			First(&assignment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNutritionAssignmentNotFound
			}
			return fmt.Errorf("failed to lock nutrition assignment: %w", err)
		}

		if assignment.Status != models.NutritionAssignmentStatusProcessing {
			return ErrNutritionAssignmentNotRetryable
		}
		if assignment.QuestionnaireVersion != plan.QuestionnaireVersion {
			return fmt.Errorf("%w: assignment targets questionnaire version %d",
				ErrNutritionAssignmentNotRetryable, assignment.QuestionnaireVersion)
		}

		nextVersion := assignment.Version + 1
		plan.AssignmentID = assignment.ID
		plan.UserID = assignment.UserID
		plan.ProgramID = assignment.ProgramID
		plan.QuestionnaireID = assignment.QuestionnaireID
		plan.Version = nextVersion
		plan.Status = models.NutritionPlanStatusActive
		if plan.GeneratedAt.IsZero() {
			plan.GeneratedAt = time.Now().UTC()
		}

		// The previous version is retained, never deleted: it is the record of
		// what the client was previously shown.
		if err := tx.Model(&models.NutritionPlan{}).
			Where("assignment_id = ? AND status = ?", assignment.ID, models.NutritionPlanStatusActive).
			Updates(map[string]any{"status": models.NutritionPlanStatusSuperseded}).Error; err != nil {
			return fmt.Errorf("failed to supersede previous nutrition plan: %w", err)
		}

		cautions, err := json.Marshal(plan.Cautions)
		if err != nil {
			return fmt.Errorf("failed to encode nutrition cautions: %w", err)
		}
		plan.Cautions = cautions

		if err := tx.Create(plan).Error; err != nil {
			if isDuplicateEntry(err) {
				// The unique guarantees — one active plan per assignment and one
				// plan per version — are the last line of defence against a
				// concurrent run reaching this point.
				return ErrNutritionAssignmentNotRetryable
			}
			return fmt.Errorf("failed to create nutrition plan: %w", err)
		}

		// Meals, foods and exclusions are plain slices rather than GORM
		// associations, so they are written explicitly. They are inserted after the
		// header because they carry foreign keys into it, and a failure anywhere in
		// this set rolls the whole version back rather than delivering a plan whose
		// meals are missing.
		if err := insertPlanChildren(tx, plan); err != nil {
			return err
		}

		if err := tx.Model(&models.NutritionAssignment{}).
			Where("id = ?", assignment.ID).
			Updates(map[string]any{
				"status":         models.NutritionAssignmentStatusCompleted,
				"failure_reason": nil,
				"generated_at":   plan.GeneratedAt,
				"version":        nextVersion,
			}).Error; err != nil {
			return fmt.Errorf("failed to complete nutrition assignment: %w", err)
		}

		return nil
	})
}

// insertPlanChildren writes the meals, foods and exclusions of a plan version.
//
// The foods of every meal go out in one statement: a plan can hold up to fourteen
// occasions, and inserting them meal by meal would turn a single generation into a
// long chain of round trips for no benefit.
func insertPlanChildren(tx *gorm.DB, plan *models.NutritionPlan) error {
	meals := make([]models.NutritionPlanMeal, 0, len(plan.Meals))
	items := make([]models.NutritionPlanMealItem, 0)
	for _, meal := range plan.Meals {
		items = append(items, meal.Items...)
		meals = append(meals, meal)
	}

	if len(meals) > 0 {
		if err := tx.Create(&meals).Error; err != nil {
			return fmt.Errorf("failed to create nutrition plan meals: %w", err)
		}
	}
	if len(items) > 0 {
		if err := tx.Create(&items).Error; err != nil {
			return fmt.Errorf("failed to create nutrition plan meal items: %w", err)
		}
	}
	if len(plan.Exclusions) > 0 {
		if err := tx.Create(&plan.Exclusions).Error; err != nil {
			return fmt.Errorf("failed to create nutrition plan exclusions: %w", err)
		}
	}
	return nil
}

// FindActiveByAssignment returns the active plan of an assignment, fully
// populated.
//
// Reads go through the v_nutrition_plan* views, which already exclude
// soft-deleted rows, so Unscoped is required rather than a soft-delete filter:
// the views do not project deleted_at, and re-applying GORM's implicit predicate
// would reference a column that is not there.
func (r *nutritionPlanRepository) FindActiveByAssignment(ctx context.Context, assignmentID string) (*models.NutritionPlan, error) {
	var plan models.NutritionPlan
	if err := r.db.WithContext(ctx).
		Unscoped().
		Table("v_nutrition_plan").
		Where("assignment_id = ? AND status = ?", assignmentID, models.NutritionPlanStatusActive).
		Take(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNutritionPlanNotFound
		}
		return nil, fmt.Errorf("failed to find nutrition plan: %w", err)
	}
	if err := r.hydrate(ctx, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// FindActiveByUserAndProgram returns the active plan for an owner and program.
//
// The owner filter is applied in the selecting query rather than as a later
// check, so there is no code path in which a plan is loaded and then compared.
func (r *nutritionPlanRepository) FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionPlan, error) {
	var plan models.NutritionPlan
	if err := r.db.WithContext(ctx).
		Unscoped().
		Table("v_nutrition_plan").
		Where("user_id = ? AND program_id = ? AND status = ?", userID, programID, models.NutritionPlanStatusActive).
		Take(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNutritionPlanNotFound
		}
		return nil, fmt.Errorf("failed to find nutrition plan: %w", err)
	}
	if err := r.hydrate(ctx, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// ListVersions returns every version of an assignment, newest first.
func (r *nutritionPlanRepository) ListVersions(ctx context.Context, assignmentID string) ([]models.NutritionPlan, error) {
	var plans []models.NutritionPlan
	if err := r.db.WithContext(ctx).
		Unscoped().
		Table("v_nutrition_plan").
		Where("assignment_id = ?", assignmentID).
		Order("version DESC").
		Find(&plans).Error; err != nil {
		return nil, fmt.Errorf("failed to list nutrition plan versions: %w", err)
	}
	return plans, nil
}

// hydrate loads the meals, foods and exclusions of a plan.
//
// The foods of every meal are fetched in a single query and grouped in memory
// rather than queried per meal. A plan can hold up to fourteen occasions, and a
// per-meal lookup would turn every plan read into a dozen round trips for no
// benefit.
func (r *nutritionPlanRepository) hydrate(ctx context.Context, plan *models.NutritionPlan) error {
	var meals []models.NutritionPlanMeal
	if err := r.db.WithContext(ctx).
		Unscoped().
		Table("v_nutrition_plan_meal").
		Where("plan_id = ?", plan.ID).
		Order("position ASC").
		Find(&meals).Error; err != nil {
		return fmt.Errorf("failed to load nutrition plan meals: %w", err)
	}

	var items []models.NutritionPlanMealItem
	if err := r.db.WithContext(ctx).
		Unscoped().
		Table("v_nutrition_plan_meal_item").
		Where("plan_id = ?", plan.ID).
		Order("position ASC").
		Find(&items).Error; err != nil {
		return fmt.Errorf("failed to load nutrition plan meal items: %w", err)
	}

	byMeal := map[string][]models.NutritionPlanMealItem{}
	for _, item := range items {
		byMeal[item.MealID] = append(byMeal[item.MealID], item)
	}
	for i := range meals {
		meals[i].Items = byMeal[meals[i].ID]
	}

	var exclusions []models.NutritionPlanExclusion
	if err := r.db.WithContext(ctx).
		Unscoped().
		Table("v_nutrition_plan_exclusion").
		Where("plan_id = ?", plan.ID).
		Order("reason_code ASC, token ASC").
		Find(&exclusions).Error; err != nil {
		return fmt.Errorf("failed to load nutrition plan exclusions: %w", err)
	}

	plan.Meals = meals
	plan.Exclusions = exclusions
	return nil
}
