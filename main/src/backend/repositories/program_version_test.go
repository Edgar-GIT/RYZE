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

// A buyer keeps the version they bought (v2.133.0): a sold package cannot
// change in place; editing makes a new version, and publishing it takes the
// old one off the market while its buyers keep it.
func TestProgramVersionsKeepWhatBuyersBought(t *testing.T) {
	db := openInvoiceDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()

	newUser := func(label string) *models.User {
		user := &models.User{Email: fmt.Sprintf("%s-%d@ryze.local", label, time.Now().UnixNano()), PasswordHash: "hash", FirstName: "V", LastName: "R"}
		if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
			t.Fatalf("user: %v", err)
		}
		return user
	}
	trainer := &models.Trainer{UserID: newUser("version-trainer").ID}
	if err := repositories.NewTrainerRepository(tx).Create(ctx, trainer); err != nil {
		t.Fatalf("trainer: %v", err)
	}
	buyer := newUser("version-buyer")

	versions := repositories.NewProgramVersionRepository(tx)
	svc := programs.NewService(repositories.NewProgramRepository(tx), config.PricingConfig{MinProgramPriceMinorUnits: 100},
		programs.WithNutrition(repositories.NewCuratedNutritionRepository(tx)), programs.WithVersions(versions))
	nutrition := svc.(programs.NutritionService)
	versioning := svc.(programs.VersionService)

	minHeight, maxHeight := 160, 190
	original, err := svc.CreateProgram(ctx, trainer.ID, programs.CreateProgramInput{
		Name: "Version probe", Type: models.ProgramTypePremium, Objective: models.ProgramObjectiveGainMuscle,
		DurationWeeks: 4, FrequencyPerWeek: 3,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := tx.Model(&models.Program{}).Where("id = ?", original.ID).
		Updates(map[string]any{"min_height_cm": minHeight, "max_height_cm": maxHeight}).Error; err != nil {
		t.Fatalf("ranges: %v", err)
	}
	if err := tx.Create(&models.ProgramWeek{ProgramID: original.ID, WeekNumber: 1}).Error; err != nil {
		t.Fatalf("week: %v", err)
	}
	plan := curated_nutrition.PlanInput{
		DietaryPattern: "balanced", Summary: "A complete day.", PrepGuidance: "Cook once.", HydrationLitres: 2.5, HydrationNote: "Water.",
		Meals: []curated_nutrition.MealInput{{
			Label: "Breakfast", Kind: models.NutritionPlanMealKindMeal, Alternatives: "Rice instead of oats",
			Items: []curated_nutrition.ItemInput{{FoodName: "Oats", Category: "Grain", Quantity: 80, Unit: "g", Calories: 300, ProteinGrams: 10, CarbohydrateGrams: 50, FatGrams: 6, FiberGrams: 8}},
		}},
	}
	if _, err := nutrition.SaveNutrition(ctx, trainer.ID, original.ID, plan); err != nil {
		t.Fatalf("meal plan: %v", err)
	}
	if _, err := svc.PublishProgram(ctx, trainer.ID, original.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Sold.
	if err := tx.Create(&models.Purchase{UserID: buyer.ID, ProgramID: original.ID, PriceMinorUnits: 1499, Currency: "EUR", Status: models.PurchaseStatusCompleted}).Error; err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if err := tx.Create(&models.Entitlement{UserID: buyer.ID, ProgramID: original.ID}).Error; err != nil {
		t.Fatalf("entitlement: %v", err)
	}
	if sold, err := versions.IsSoldByTrainer(ctx, trainer.ID, original.ID); err != nil || !sold {
		t.Fatalf("expected sold: %v %v", sold, err)
	}
	if sold, _ := versions.IsSoldByTrainer(ctx, "another-trainer", original.ID); sold {
		t.Fatalf("another trainer must not learn that the package was sold")
	}
	if _, err := nutrition.SaveNutrition(ctx, trainer.ID, original.ID, plan); !errors.Is(err, programs.ErrProgramSold) {
		t.Fatalf("a sold package's meal plan cannot change in place: %v", err)
	}
	if err := repositories.NewTrainerStructureRepository(tx).ReplaceTrainerStructure(ctx, trainer.ID, original.ID, nil); !errors.Is(err, repositories.ErrProgramContentLocked) {
		t.Fatalf("a sold package's structure cannot change in place: %v", err)
	}

	// A new version: a full copy, as a draft; asking again returns the same.
	version, err := versioning.CreateVersion(ctx, trainer.ID, original.ID)
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	if version.ID == original.ID || version.Status != models.ProgramStatusDraft || version.PreviousVersionID == nil || *version.PreviousVersionID != original.ID {
		t.Fatalf("unexpected version %+v", version)
	}
	if version.Criteria.MinHeightCm == nil || *version.Criteria.MinHeightCm != minHeight {
		t.Fatalf("the ranges are copied, got %+v", version.Criteria)
	}
	again, err := versioning.CreateVersion(ctx, trainer.ID, original.ID)
	if err != nil || again.ID != version.ID {
		t.Fatalf("the open version is reused: %v %v", again, err)
	}
	copied, err := nutrition.GetNutrition(ctx, trainer.ID, version.ID)
	if err != nil || copied == nil || len(copied.Meals) != 1 || copied.Meals[0].Alternatives != "Rice instead of oats" {
		t.Fatalf("the meal plan is copied: %+v %v", copied, err)
	}
	var weeks int64
	tx.Model(&models.ProgramWeek{}).Where("program_id = ?", version.ID).Count(&weeks)
	if weeks != 1 {
		t.Fatalf("the weeks are copied, got %d", weeks)
	}
	info, err := versioning.VersionInfo(ctx, trainer.ID, original.ID)
	if err != nil || !info.Sold || info.DraftVersionID != version.ID {
		t.Fatalf("version info: %+v %v", info, err)
	}

	// The new version can change; publishing it supersedes the old one.
	edited := plan
	edited.Summary = "A better day."
	if _, err := nutrition.SaveNutrition(ctx, trainer.ID, version.ID, edited); err != nil {
		t.Fatalf("edit the new version: %v", err)
	}
	if _, err := svc.PublishProgram(ctx, trainer.ID, version.ID); err != nil {
		t.Fatalf("publish the version: %v", err)
	}
	var old models.Program
	if err := tx.First(&old, "id = ?", original.ID).Error; err != nil {
		t.Fatalf("old: %v", err)
	}
	if old.SupersededAt == nil || old.ArchivedAt == nil || old.Status != models.ProgramStatusPublished {
		t.Fatalf("the old version leaves the market but stays published for its buyers: %+v", old)
	}
	// The buyer's meal plan is the one they bought.
	stored, err := repositories.NewCuratedNutritionRepository(tx).FindByProgramID(ctx, original.ID)
	if err != nil || stored.Summary != "A complete day." {
		t.Fatalf("the buyer keeps the version they bought: %+v %v", stored, err)
	}

	listed, _, err := repositories.NewProgramRepository(tx).ListByTrainer(ctx, trainer.ID, 1, 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, program := range listed {
		if program.ID == original.ID {
			t.Fatalf("the old version is not in the trainer's list")
		}
	}
	archive := repositories.NewProgramArchiveRepository(tx)
	if count, _ := archive.CountArchived(ctx, trainer.ID); count != 0 {
		t.Fatalf("an old version does not count as archived, got %d", count)
	}
	if _, err := archive.Restore(ctx, repositories.ProgramOwner{TrainerID: trainer.ID}, original.ID); !errors.Is(err, repositories.ErrArchiveState) {
		t.Fatalf("an old version never returns to the market: %v", err)
	}
	owner := repositories.NewEntitlementRepository(tx).(interface {
		OwnsOtherVersion(ctx context.Context, userID, programID string) (bool, error)
	})
	if owns, err := owner.OwnsOtherVersion(ctx, buyer.ID, version.ID); err != nil || !owns {
		t.Fatalf("the buyer owns the package through version 1: %v %v", owns, err)
	}
}
