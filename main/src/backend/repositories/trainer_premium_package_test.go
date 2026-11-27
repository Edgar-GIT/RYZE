package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/curated_nutrition"
	"ryze/backend/services/programs"
)

// A trainer's Premium plan is a Premium Level 1 package (v2.132.0): it has an
// objective and a curated meal plan, and it is published only with both.
func TestTrainerPremiumPackageLifecycle(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()

	newTrainer := func(label string) *models.Trainer {
		user := &models.User{Email: fmt.Sprintf("%s-%d@ryze.local", label, time.Now().UnixNano()), PasswordHash: "hash", FirstName: "T", LastName: "R"}
		if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
			t.Fatalf("user: %v", err)
		}
		trainer := &models.Trainer{UserID: user.ID}
		if err := repositories.NewTrainerRepository(tx).Create(ctx, trainer); err != nil {
			t.Fatalf("trainer: %v", err)
		}
		return trainer
	}
	owner := newTrainer("l1-owner")
	other := newTrainer("l1-other")

	svc := programs.NewService(
		repositories.NewProgramRepository(tx),
		config.PricingConfig{MinProgramPriceMinorUnits: 100},
		programs.WithNutrition(repositories.NewCuratedNutritionRepository(tx)),
	)
	nutrition := svc.(programs.NutritionService)

	if _, err := svc.CreateProgram(ctx, owner.ID, programs.CreateProgramInput{Name: "No objective", Type: models.ProgramTypePremium}); !errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("a Premium package needs its objective: %v", err)
	}
	program, err := svc.CreateProgram(ctx, owner.ID, programs.CreateProgramInput{
		Name:      "Lean in 8 weeks",
		Type:      models.ProgramTypePremium,
		Objective: models.ProgramObjectiveGainMuscle,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if program.ProductType != models.ProgramProductTypePremiumLevel1 || program.PriceMinorUnits != 1499 {
		t.Fatalf("expected a Level 1 package at the catalogue price, got %s %d", program.ProductType, program.PriceMinorUnits)
	}

	if _, err := svc.PublishProgram(ctx, owner.ID, program.ID); !errors.Is(err, programs.ErrNutritionRequired) {
		t.Fatalf("publishing without a meal plan must be refused: %v", err)
	}

	input := curated_nutrition.PlanInput{
		DietaryPattern:  "balanced",
		Summary:         "A complete day of meals.",
		PrepGuidance:    "Cook once and portion.",
		HydrationLitres: 2.5,
		HydrationNote:   "Spread water through the day.",
		Meals: []curated_nutrition.MealInput{{
			Label: "Breakfast",
			Kind:  models.NutritionPlanMealKindMeal,
			Items: []curated_nutrition.ItemInput{{FoodName: "Oats", Category: "Grain", Quantity: 80, Unit: "g", Calories: 300, ProteinGrams: 10, CarbohydrateGrams: 50, FatGrams: 6, FiberGrams: 8}},
		}},
	}
	if _, err := nutrition.SaveNutrition(ctx, other.ID, program.ID, input); !errors.Is(err, programs.ErrProgramNotFound) {
		t.Fatalf("another trainer cannot write the meal plan: %v", err)
	}
	if _, err := nutrition.SaveNutrition(ctx, owner.ID, program.ID, input); err != nil {
		t.Fatalf("save meal plan: %v", err)
	}
	stored, err := nutrition.GetNutrition(ctx, owner.ID, program.ID)
	if err != nil || stored == nil || len(stored.Meals) != 1 {
		t.Fatalf("read meal plan: %+v %v", stored, err)
	}
	// Saving again replaces it (no duplicate meals).
	if _, err := nutrition.SaveNutrition(ctx, owner.ID, program.ID, input); err != nil {
		t.Fatalf("save again: %v", err)
	}
	if again, _ := nutrition.GetNutrition(ctx, owner.ID, program.ID); again == nil || len(again.Meals) != 1 {
		t.Fatalf("expected the plan replaced, got %+v", again)
	}

	published, err := svc.PublishProgram(ctx, owner.ID, program.ID)
	if err != nil {
		t.Fatalf("publish with meal plan: %v", err)
	}
	if published.Status != models.ProgramStatusPublished {
		t.Fatalf("expected published, got %s", published.Status)
	}

	// A published package keeps its family.
	free := models.ProgramTypeFree
	if _, err := svc.UpdateProgram(ctx, owner.ID, program.ID, programs.UpdateProgramInput{Type: &free}); !errors.Is(err, programs.ErrInvalidInput) {
		t.Fatalf("a published package cannot become free: %v", err)
	}
}
