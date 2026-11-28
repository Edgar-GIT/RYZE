package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var ErrCuratedNutritionNotFound = errors.New("curated nutrition plan not found")

// CuratedNutritionRepository reads the package-owned nutrition plan delivered
// with a curated Premium Level 1 program. The plan is not owner-scoped: access
// is decided by the caller's entitlement, and only then is the package plan
// loaded.
type CuratedNutritionRepository interface {
	FindByProgramID(ctx context.Context, programID string) (*models.CuratedNutritionPlan, error)
	// ReplaceForTrainerProgram replaces the meal plan of a trainer's Premium
	// package (v2.132.0); the package must belong to the trainer.
	ReplaceForTrainerProgram(ctx context.Context, trainerID, programID string, plan *models.CuratedNutritionPlan) error
}

type curatedNutritionRepository struct {
	db *gorm.DB
}

func NewCuratedNutritionRepository(db *gorm.DB) CuratedNutritionRepository {
	return &curatedNutritionRepository{db: db}
}

func (r *curatedNutritionRepository) FindByProgramID(ctx context.Context, programID string) (*models.CuratedNutritionPlan, error) {
	var plan models.CuratedNutritionPlan
	// The view already leaves out deleted plans and has no deleted_at column;
	// Unscoped stops GORM adding that soft-delete filter (which made every
	// read fail with "column deleted_at does not exist").
	if err := r.db.WithContext(ctx).Unscoped().
		Table("v_curated_nutrition_plan").
		Where("program_id = ?", programID).
		Take(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCuratedNutritionNotFound
		}
		return nil, fmt.Errorf("failed to find curated nutrition plan: %w", err)
	}

	var meals []models.CuratedNutritionPlanMeal
	if err := r.db.WithContext(ctx).
		Table("v_curated_nutrition_plan_meal").
		Where("plan_id = ?", plan.ID).
		Order("position ASC").
		Find(&meals).Error; err != nil {
		return nil, fmt.Errorf("failed to find curated nutrition meals: %w", err)
	}

	var items []models.CuratedNutritionPlanMealItem
	if err := r.db.WithContext(ctx).
		Table("v_curated_nutrition_plan_meal_item").
		Where("plan_id = ?", plan.ID).
		Order("position ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to find curated nutrition foods: %w", err)
	}

	itemsByMeal := make(map[string][]models.CuratedNutritionPlanMealItem, len(meals))
	for _, item := range items {
		itemsByMeal[item.MealID] = append(itemsByMeal[item.MealID], item)
	}
	for i := range meals {
		meals[i].Items = itemsByMeal[meals[i].ID]
	}
	plan.Meals = meals
	return &plan, nil
}

func (r *curatedNutritionRepository) ReplaceForTrainerProgram(ctx context.Context, trainerID, programID string, plan *models.CuratedNutritionPlan) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owned models.Program
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND trainer_id = ? AND product_type = ?", programID, trainerID, models.ProgramProductTypePremiumLevel1).
			Take(&owned).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to verify the package: %w", err)
		}
		// A sold package is never changed in place (v2.133.0): its buyers keep
		// the version they bought.
		if sold, err := hasCompletedPurchase(tx, programID); err != nil {
			return err
		} else if sold {
			return ErrProgramContentLocked
		}
		return reconcileCuratedNutrition(tx, programID, plan)
	})
}
