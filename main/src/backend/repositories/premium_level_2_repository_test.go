package repositories_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

func TestPremiumLevel2RepositoryLifecycle(t *testing.T) {
	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	tx := db.Begin()
	defer tx.Rollback()

	ctx := context.Background()
	userRepo := repositories.NewUserRepository(tx)
	trainerRepo := repositories.NewTrainerRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	genericRepo := repositories.NewGenericProgramRepository(tx)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	requestRepo := repositories.NewPremiumLevel2Repository(tx)

	seedUser := func(label string) *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("premium-l2-%s-%d@ryze.local", label, time.Now().UnixNano()),
			PasswordHash: "prepared-hash-outside-repository-scope",
			FirstName:    "Test",
			LastName:     label,
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}
	client := seedUser("client")
	trainerUser := seedUser("trainer")
	trainer := &models.Trainer{UserID: trainerUser.ID}
	if err := trainerRepo.Create(ctx, trainer); err != nil {
		t.Fatalf("create trainer: %v", err)
	}
	offer := &models.Program{
		Name:            "Premium Level 2 - Individual Coaching",
		Description:     "A trainer-authored programme and nutrition plan.",
		Type:            models.ProgramTypePremium,
		ProductType:     models.ProgramProductTypePremiumLevel2,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 14999,
		Currency:        string(models.ProgramCurrencyEUR),
	}
	if err := genericRepo.CreateFull(ctx, offer); err != nil {
		t.Fatalf("create offer: %v", err)
	}

	request, err := requestRepo.CreateRequest(ctx, client.ID, offer.ID)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if request.Status != models.PremiumLevel2StatusAwaitingIntake {
		t.Fatalf("expected awaiting intake, got %q", request.Status)
	}
	duplicate, err := requestRepo.CreateRequest(ctx, client.ID, offer.ID)
	if err != nil {
		t.Fatalf("idempotent create request: %v", err)
	}
	if duplicate.ID != request.ID {
		t.Fatalf("expected duplicate to return the existing request")
	}

	answers := json.RawMessage(`{"primary_goal":"fat_loss"}`)
	request, intake, err := requestRepo.CreateIntake(ctx, request.ID, answers)
	if err != nil {
		t.Fatalf("create intake: %v", err)
	}
	if intake.Version != 1 || request.IntakeVersion != 1 || request.Status != models.PremiumLevel2StatusAwaitingTrainerAssignment {
		t.Fatalf("unexpected intake state: version=%d status=%q", request.IntakeVersion, request.Status)
	}

	request, err = requestRepo.AssignTrainer(ctx, request.ID, trainer.ID, "admin-1")
	if err != nil {
		t.Fatalf("assign trainer: %v", err)
	}
	if request.Status != models.PremiumLevel2StatusInProgress || request.TrainerID == nil || *request.TrainerID != trainer.ID {
		t.Fatalf("unexpected assignment state: %+v", request)
	}

	training := &models.Program{
		TrainerID: trainer.ID,
		Name:      "Client individual programme",
		Type:      models.ProgramTypePremium,
		Status:    models.ProgramStatusDraft,
	}
	if err := programRepo.Create(ctx, training); err != nil {
		t.Fatalf("create training program: %v", err)
	}
	week := &models.ProgramWeek{ProgramID: training.ID, WeekNumber: 1}
	if err := tx.Create(week).Error; err != nil {
		t.Fatalf("create week: %v", err)
	}
	workout := &models.ProgramWorkout{ProgramWeekID: week.ID, Position: 1}
	if err := tx.Create(workout).Error; err != nil {
		t.Fatalf("create workout: %v", err)
	}
	exercise := &models.Exercise{Name: "Test squat", Status: models.ExerciseStatusPublished}
	if err := tx.Create(exercise).Error; err != nil {
		t.Fatalf("create exercise: %v", err)
	}
	workoutExercise := &models.WorkoutExercise{ProgramWorkoutID: workout.ID, ExerciseID: exercise.ID, Position: 1}
	if err := tx.Create(workoutExercise).Error; err != nil {
		t.Fatalf("create workout exercise: %v", err)
	}

	request, err = requestRepo.SetTrainingProgram(ctx, request.ID, trainer.ID, training.ID)
	if err != nil {
		t.Fatalf("attach training program: %v", err)
	}
	if request.TrainingProgramID == nil || *request.TrainingProgramID != training.ID {
		t.Fatal("expected training program attachment")
	}

	plan := &models.PremiumLevel2NutritionPlan{
		Summary:         "Balanced plan for the training week.",
		PrepGuidance:    "Batch cook on Sunday.",
		HydrationLitres: 2.5,
		HydrationNote:   "Drink across the day.",
		Cautions:        json.RawMessage(`[]`),
		Meals: []models.PremiumLevel2NutritionPlanMeal{
			{Label: "Breakfast", Position: 1, Items: []models.PremiumLevel2NutritionPlanMealItem{{FoodName: "Oats", Category: "grain", Quantity: 80, Unit: "g", Calories: 300, ProteinGrams: 10, CarbohydrateGrams: 50, FatGrams: 6, FiberGrams: 8, Position: 1}}},
			{Label: "Lunch", Position: 2, Items: []models.PremiumLevel2NutritionPlanMealItem{{FoodName: "Chicken rice", Category: "meal", Quantity: 1, Unit: "serving", Calories: 600, ProteinGrams: 45, CarbohydrateGrams: 70, FatGrams: 12, FiberGrams: 5, Position: 1}}},
			{Label: "Dinner", Position: 3, Items: []models.PremiumLevel2NutritionPlanMealItem{{FoodName: "Salmon potatoes", Category: "meal", Quantity: 1, Unit: "serving", Calories: 650, ProteinGrams: 40, CarbohydrateGrams: 60, FatGrams: 25, FiberGrams: 7, Position: 1}}},
		},
	}
	saved, err := requestRepo.SaveNutritionPlan(ctx, request.ID, trainer.ID, plan)
	if err != nil {
		t.Fatalf("save nutrition: %v", err)
	}
	if len(saved.Meals) != 3 || saved.Status != "draft" {
		t.Fatalf("unexpected saved plan: meals=%d status=%q", len(saved.Meals), saved.Status)
	}
	request, err = requestRepo.FindRequestByID(ctx, request.ID)
	if err != nil {
		t.Fatalf("find ready request: %v", err)
	}
	if request.Status != models.PremiumLevel2StatusReadyForDelivery {
		t.Fatalf("expected ready status, got %q", request.Status)
	}

	delivered, err := requestRepo.Deliver(ctx, request.ID, trainer.ID)
	if err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if delivered.Status != models.PremiumLevel2StatusDelivered || delivered.DeliveredAt == nil {
		t.Fatalf("unexpected delivery state: %+v", delivered)
	}
	entitlement, err := entitlementRepo.FindActiveByUserAndProgram(ctx, client.ID, training.ID)
	if err != nil || entitlement == nil {
		t.Fatalf("expected delivered entitlement, got %v err=%v", entitlement, err)
	}
	deliveredPlan, err := requestRepo.FindNutritionPlan(ctx, request.ID)
	if err != nil {
		t.Fatalf("find delivered nutrition: %v", err)
	}
	if deliveredPlan.Status != "delivered" || len(deliveredPlan.Meals) != 3 {
		t.Fatalf("unexpected delivered plan: status=%q meals=%d", deliveredPlan.Status, len(deliveredPlan.Meals))
	}
	if _, err := requestRepo.Deliver(ctx, request.ID, trainer.ID); err != nil {
		t.Fatalf("idempotent delivery: %v", err)
	}

	linkReader, ok := programRepo.(interface {
		PremiumLevel2Link(context.Context, string) (bool, bool, error)
	})
	if !ok {
		t.Fatal("program repository does not expose the premium level 2 link reader")
	}
	linked, deliveredLink, err := linkReader.PremiumLevel2Link(ctx, training.ID)
	if err != nil {
		t.Fatalf("premium level 2 link: %v", err)
	}
	if !linked || !deliveredLink {
		t.Fatalf("expected delivered link, got linked=%v delivered=%v", linked, deliveredLink)
	}

	events, err := requestRepo.ListEvents(ctx, request.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) < 5 {
		t.Fatalf("expected lifecycle events, got %d", len(events))
	}
}

func TestPremiumLevel2RepositoryRejectsInvalidTransitions(t *testing.T) {
	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	tx := db.Begin()
	defer tx.Rollback()
	ctx := context.Background()
	userRepo := repositories.NewUserRepository(tx)
	genericRepo := repositories.NewGenericProgramRepository(tx)
	requestRepo := repositories.NewPremiumLevel2Repository(tx)
	user := &models.User{Email: fmt.Sprintf("premium-l2-state-%d@ryze.local", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "State", LastName: "Tester"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	offer := &models.Program{Name: "State offer", Type: models.ProgramTypePremium, ProductType: models.ProgramProductTypePremiumLevel2, Status: models.ProgramStatusPublished, Currency: string(models.ProgramCurrencyEUR)}
	if err := genericRepo.CreateFull(ctx, offer); err != nil {
		t.Fatalf("create offer: %v", err)
	}
	request, err := requestRepo.CreateRequest(ctx, user.ID, offer.ID)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if _, _, err := requestRepo.CreateIntake(ctx, request.ID, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("create intake: %v", err)
	}
	if _, err := requestRepo.Deliver(ctx, request.ID, "missing-trainer"); !errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound) {
		t.Fatalf("expected not found for unassigned request, got %v", err)
	}
	if _, err := requestRepo.SetTrainingProgram(ctx, request.ID, "missing-trainer", "missing-program"); !errors.Is(err, repositories.ErrPremiumLevel2RequestNotFound) {
		t.Fatalf("expected not found for foreign trainer, got %v", err)
	}
}
