package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

var ErrCuratedNutritionNotFound = errors.New("curated nutrition plan not found")

// CuratedNutritionRepository reads the package-owned nutrition plan delivered
// with a curated Premium Level 1 program. The plan is not owner-scoped: access
// is decided by the caller's entitlement, and only then is the package plan
// loaded.
type CuratedNutritionRepository interface {
	FindByProgramID(ctx context.Context, programID string) (*models.CuratedNutritionPlan, error)
}

type curatedNutritionRepository struct {
	db *gorm.DB
}

func NewCuratedNutritionRepository(db *gorm.DB) CuratedNutritionRepository {
	return &curatedNutritionRepository{db: db}
}

func (r *curatedNutritionRepository) FindByProgramID(ctx context.Context, programID string) (*models.CuratedNutritionPlan, error) {
	var plan models.CuratedNutritionPlan
	if err := r.db.WithContext(ctx).
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
