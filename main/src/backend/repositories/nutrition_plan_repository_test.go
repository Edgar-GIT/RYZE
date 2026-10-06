package repositories_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// nutritionRepoHarness is the shared database fixture for the nutrition
// assignment and plan repository tests. Everything runs inside one transaction
// that is rolled back, so the tests never leave a row behind and never depend on
// a migration having been run by hand.
type nutritionRepoHarness struct {
	tx                *gorm.DB
	ctx               context.Context
	userRepo          repositories.UserRepository
	trainerRepo       repositories.TrainerRepository
	programRepo       repositories.ProgramRepository
	questionnaireRepo repositories.NutritionQuestionnaireRepository
	assignmentRepo    repositories.NutritionAssignmentRepository
	planRepo          repositories.NutritionPlanRepository
}

func newNutritionRepoHarness(t *testing.T) *nutritionRepoHarness {
	t.Helper()
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
	t.Cleanup(func() { _ = tx.Rollback().Error })

	return &nutritionRepoHarness{
		tx:                tx,
		ctx:               context.Background(),
		userRepo:          repositories.NewUserRepository(tx),
		trainerRepo:       repositories.NewTrainerRepository(tx),
		programRepo:       repositories.NewProgramRepository(tx),
		questionnaireRepo: repositories.NewNutritionQuestionnaireRepository(tx),
		assignmentRepo:    repositories.NewNutritionAssignmentRepository(tx),
		planRepo:          repositories.NewNutritionPlanRepository(tx),
	}
}

// seedAccount creates a client, a trainer and a published program the client can
// hold an assignment for, plus the intake the assignment is derived from.
func (h *nutritionRepoHarness) seedAccount(t *testing.T, label string) (*models.User, *models.Program, *models.NutritionQuestionnaire) {
	t.Helper()

	trainerUser := &models.User{
		Email:        fmt.Sprintf("nutrition-repo-trainer-%s-%d@ryze.local", label, time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Trainer",
		LastName:     label,
	}
	if err := h.userRepo.Create(h.ctx, trainerUser); err != nil {
		t.Fatalf("create trainer user: %v", err)
	}
	trainer := &models.Trainer{UserID: trainerUser.ID}
	if err := h.trainerRepo.Create(h.ctx, trainer); err != nil {
		t.Fatalf("create trainer: %v", err)
	}

	program := &models.Program{
		TrainerID:   trainer.ID,
		Name:        "Premium Level 1 — " + label,
		Type:        models.ProgramTypePremium,
		ProductType: models.ProgramProductTypePremiumLevel1,
		Status:      models.ProgramStatusPublished,
	}
	if err := h.programRepo.Create(h.ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}

	client := &models.User{
		Email:        fmt.Sprintf("nutrition-repo-client-%s-%d@ryze.local", label, time.Now().UnixNano()),
		PasswordHash: "prepared-hash-outside-repository-scope",
		FirstName:    "Client",
		LastName:     label,
	}
	if err := h.userRepo.Create(h.ctx, client); err != nil {
		t.Fatalf("create client: %v", err)
	}

	questionnaire, err := h.questionnaireRepo.UpsertForUserAndProgram(h.ctx, client.ID, program.ID, []byte(`{"dietary_pattern":"omnivore"}`))
	if err != nil {
		t.Fatalf("create questionnaire: %v", err)
	}

	return client, program, questionnaire
}

// claim moves a fresh assignment into the state generation runs from, at the
// version the assignment currently reports, so a test can go straight to the
// write it wants to exercise.
func (h *nutritionRepoHarness) claim(t *testing.T, userID, programID string, questionnaire *models.NutritionQuestionnaire) {
	t.Helper()
	if _, _, err := h.assignmentRepo.EnsurePendingForUserAndProgram(h.ctx, userID, programID, questionnaire.ID, questionnaire.Version); err != nil {
		t.Fatalf("ensure assignment: %v", err)
	}
	if err := h.assignmentRepo.MarkProcessing(h.ctx, userID, programID, questionnaire.ID, questionnaire.Version, h.currentVersion(t, userID, programID)); err != nil {
		t.Fatalf("claim assignment: %v", err)
	}
}

// currentVersion reports the revision an assignment currently sits at, which is
// the value a claim has to match to be accepted.
func (h *nutritionRepoHarness) currentVersion(t *testing.T, userID, programID string) int {
	t.Helper()
	assignment, err := h.assignmentRepo.FindByUserAndProgram(h.ctx, userID, programID)
	if err != nil {
		t.Fatalf("read assignment: %v", err)
	}
	return assignment.Version
}

// newPlanBody builds the body of a plan version. The identifiers, owner and
// version are deliberately left to Complete, because that is exactly what the
// repository is responsible for deriving.
func newPlanBody(questionnaire *models.NutritionQuestionnaire) *models.NutritionPlan {
	mealNotes := "Cook from scratch where you can."
	substitution := "Swap for 1 slice of wholemeal bread."

	return &models.NutritionPlan{
		QuestionnaireID:      questionnaire.ID,
		QuestionnaireVersion: questionnaire.Version,
		EngineVersion:        2,
		Fingerprint:          "fingerprint-" + questionnaire.ID,
		DietaryPattern:       "omnivore",
		MaintenanceCalories:  2400,
		TargetCalories:       2400,
		DailyProteinGrams:    150,
		DailyCarbsGrams:      260,
		DailyFatGrams:        75,
		DailyFiberGrams:      30,
		ProteinPercent:       25,
		CarbohydratePercent:  45,
		FatPercent:           30,
		MealsPerDay:          3,
		SnacksPerDay:         1,
		HydrationLitres:      2.5,
		HydrationNote:        "Drink steadily through the day.",
		Summary:              "A balanced day built from your recorded answers.",
		PrepGuidance:         "Batch the grains once and portion them cold.",
		Cautions:             json.RawMessage(`["This plan is general guidance, not medical advice."]`),
		GeneratedAt:          time.Now().UTC(),
		Meals: []models.NutritionPlanMeal{
			{
				Position:          1,
				Label:             "Breakfast",
				Kind:              models.NutritionPlanMealKindMeal,
				PercentOfDaily:    25,
				Calories:          600,
				ProteinGrams:      40,
				CarbohydrateGrams: 65,
				FatGrams:          20,
				Notes:             &mealNotes,
				Items: []models.NutritionPlanMealItem{
					{
						Position:          1,
						CatalogCode:       "oats-rolled",
						FoodName:          "Rolled oats",
						Category:          "grains",
						Quantity:          80,
						Unit:              "g",
						Calories:          300,
						ProteinGrams:      11,
						CarbohydrateGrams: 54,
						FatGrams:          6,
						FiberGrams:        8,
					},
					{
						Position:          2,
						CatalogCode:       "egg-boiled",
						FoodName:          "Boiled egg",
						Category:          "protein",
						Quantity:          2,
						Unit:              "count",
						Calories:          156,
						ProteinGrams:      13,
						CarbohydrateGrams: 1,
						FatGrams:          11,
						FiberGrams:        0,
					},
				},
			},
			{
				Position:          2,
				Label:             "Lunch",
				Kind:              models.NutritionPlanMealKindMeal,
				PercentOfDaily:    35,
				Calories:          840,
				ProteinGrams:      50,
				CarbohydrateGrams: 95,
				FatGrams:          28,
				Items: []models.NutritionPlanMealItem{
					{
						Position:          1,
						CatalogCode:       "rice-brown",
						FoodName:          "Brown rice",
						Category:          "grains",
						Quantity:          180,
						Unit:              "g",
						Calories:          200,
						ProteinGrams:      4,
						CarbohydrateGrams: 42,
						FatGrams:          2,
						FiberGrams:        3,
						SubstitutionNote:  &substitution,
					},
				},
			},
		},
		Exclusions: []models.NutritionPlanExclusion{
			{ReasonCode: "allergy", Token: "peanut"},
			{ReasonCode: "diet", Token: "vegetarian"},
		},
	}
}

// TestNutritionAssignmentRepository covers the claim lifecycle the generation
// run depends on: a single idempotent assignment per owner, owner-scoped reads
// and the compare-and-swap claim that makes concurrent runs safe.
func TestNutritionAssignmentRepository(t *testing.T) {
	h := newNutritionRepoHarness(t)
	client, program, questionnaire := h.seedAccount(t, "lifecycle")

	// 1. The first delivery creates the assignment; repeating it returns the same
	// row rather than a second one.
	assignment, created, err := h.assignmentRepo.EnsurePendingForUserAndProgram(
		h.ctx, client.ID, program.ID, questionnaire.ID, questionnaire.Version,
	)
	if err != nil {
		t.Fatalf("ensure assignment: %v", err)
	}
	if !created {
		t.Fatal("first ensure: expected a row to be created")
	}
	if assignment.Status != models.NutritionAssignmentStatusPending {
		t.Fatalf("first ensure: expected pending, got %q", assignment.Status)
	}

	same, createdAgain, err := h.assignmentRepo.EnsurePendingForUserAndProgram(
		h.ctx, client.ID, program.ID, questionnaire.ID, questionnaire.Version,
	)
	if err != nil {
		t.Fatalf("ensure assignment again: %v", err)
	}
	if createdAgain {
		t.Fatal("second ensure: expected no new row")
	}
	if same.ID != assignment.ID {
		t.Fatalf("second ensure: expected assignment %q, got %q", assignment.ID, same.ID)
	}

	// 2. A different owner cannot read the assignment, so the endpoint is never
	// a way to discover which programs another account purchased.
	otherClient, _, _ := h.seedAccount(t, "idor")
	if _, err := h.assignmentRepo.FindByUserAndProgram(h.ctx, otherClient.ID, program.ID); !errors.Is(err, repositories.ErrNutritionAssignmentNotFound) {
		t.Fatalf("cross-owner read: expected ErrNutritionAssignmentNotFound, got %v", err)
	}
	if _, err := h.assignmentRepo.FindByUserAndProgram(h.ctx, client.ID, "program-does-not-exist"); !errors.Is(err, repositories.ErrNutritionAssignmentNotFound) {
		t.Fatalf("unknown program read: expected ErrNutritionAssignmentNotFound, got %v", err)
	}

	// 3. The claim is a compare-and-swap: it succeeds at the version the caller
	// read and is refused once the row has moved on.
	if err := h.assignmentRepo.MarkProcessing(
		h.ctx, client.ID, program.ID, questionnaire.ID, questionnaire.Version, assignment.Version,
	); err != nil {
		t.Fatalf("claim at the read version: %v", err)
	}
	claimed, err := h.assignmentRepo.FindByUserAndProgram(h.ctx, client.ID, program.ID)
	if err != nil {
		t.Fatalf("read claimed assignment: %v", err)
	}
	if claimed.Status != models.NutritionAssignmentStatusProcessing {
		t.Fatalf("claimed: expected processing, got %q", claimed.Status)
	}

	// A second claim while processing is refused: the run is already in flight.
	if err := h.assignmentRepo.MarkProcessing(
		h.ctx, client.ID, program.ID, questionnaire.ID, questionnaire.Version, assignment.Version,
	); !errors.Is(err, repositories.ErrNutritionAssignmentNotRetryable) {
		t.Fatalf("second claim: expected ErrNutritionAssignmentNotRetryable, got %v", err)
	}
	// A stale version is refused for the same reason: the row it read is gone.
	if err := h.assignmentRepo.MarkProcessing(
		h.ctx, client.ID, program.ID, questionnaire.ID, questionnaire.Version, assignment.Version+99,
	); !errors.Is(err, repositories.ErrNutritionAssignmentNotRetryable) {
		t.Fatalf("stale claim: expected ErrNutritionAssignmentNotRetryable, got %v", err)
	}

	// 4. A failure is recorded as an internal diagnostic and leaves the
	// assignment claimable again, which is what makes a retry possible.
	if err := h.assignmentRepo.MarkFailed(h.ctx, client.ID, program.ID, "GENERATOR_NO_ELIGIBLE_FOOD"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	failed, err := h.assignmentRepo.FindByUserAndProgram(h.ctx, client.ID, program.ID)
	if err != nil {
		t.Fatalf("read failed assignment: %v", err)
	}
	if failed.Status != models.NutritionAssignmentStatusFailed {
		t.Fatalf("failed: expected failed, got %q", failed.Status)
	}
	if err := h.assignmentRepo.MarkProcessing(
		h.ctx, client.ID, program.ID, questionnaire.ID, questionnaire.Version, failed.Version,
	); err != nil {
		t.Fatalf("retry after failure: %v", err)
	}

	// 5. A failure can never be recorded against another account's assignment.
	if err := h.assignmentRepo.MarkFailed(h.ctx, otherClient.ID, program.ID, "GENERATOR_NO_ELIGIBLE_FOOD"); !errors.Is(err, repositories.ErrNutritionAssignmentNotFound) {
		t.Fatalf("cross-owner failure: expected ErrNutritionAssignmentNotFound, got %v", err)
	}
}

// TestNutritionPlanRepository covers what the client actually receives: the
// whole normalised plan written and read back, the one-active-plan guarantee,
// supersession rather than in-place rewriting, and owner scoping.
func TestNutritionPlanRepository(t *testing.T) {
	h := newNutritionRepoHarness(t)
	client, program, questionnaire := h.seedAccount(t, "plans")

	// 1. A claimed assignment writes the header, its meals, their foods and the
	// exclusions in one transaction, and the read returns all of them.
	h.claim(t, client.ID, program.ID, questionnaire)
	claimedRevision := h.currentVersion(t, client.ID, program.ID)

	first := newPlanBody(questionnaire)
	if err := h.planRepo.Complete(h.ctx, client.ID, program.ID, first); err != nil {
		t.Fatalf("complete first version: %v", err)
	}

	plan, err := h.planRepo.FindActiveByUserAndProgram(h.ctx, client.ID, program.ID)
	if err != nil {
		t.Fatalf("read active plan: %v", err)
	}
	// A completion always advances the assignment by exactly one revision, and
	// the plan it writes realises that same revision.
	if plan.Version != claimedRevision+1 {
		t.Fatalf("first version: expected %d, got %d", claimedRevision+1, plan.Version)
	}
	if plan.Status != models.NutritionPlanStatusActive {
		t.Fatalf("first version: expected active, got %q", plan.Status)
	}
	if plan.AssignmentID == "" || plan.UserID != client.ID || plan.ProgramID != program.ID {
		t.Fatalf("first version: expected owner-scoped assignment linkage, got plan %+v", plan)
	}
	if plan.QuestionnaireVersion != questionnaire.Version {
		t.Fatalf("first version: expected questionnaire version %d, got %d", questionnaire.Version, plan.QuestionnaireVersion)
	}
	if len(plan.Meals) != 2 {
		t.Fatalf("hydration: expected 2 meals, got %d", len(plan.Meals))
	}
	if len(plan.Meals[0].Items) != 2 || len(plan.Meals[1].Items) != 1 {
		t.Fatalf("hydration: expected meal items 2 and 1, got %d and %d",
			len(plan.Meals[0].Items), len(plan.Meals[1].Items))
	}
	breakfast := plan.Meals[0].Items[0]
	if breakfast.FoodName != "Rolled oats" || breakfast.Quantity != 80 || breakfast.Unit != "g" {
		t.Fatalf("item snapshot: expected Rolled oats 80 g, got %q %v %q",
			breakfast.FoodName, breakfast.Quantity, breakfast.Unit)
	}
	if breakfast.CatalogCode != "oats-rolled" {
		t.Fatalf("item snapshot: expected catalog code oats-rolled, got %q", breakfast.CatalogCode)
	}
	if plan.Meals[0].Items[1].MealID != plan.Meals[0].ID {
		t.Fatal("item linkage: expected the food to belong to its own meal")
	}
	if plan.Meals[1].Items[0].SubstitutionNote == nil {
		t.Fatal("item snapshot: expected the substitution to survive the round trip")
	}
	if len(plan.Exclusions) != 2 {
		t.Fatalf("exclusions: expected 2, got %d", len(plan.Exclusions))
	}
	if plan.Exclusions[0].ReasonCode != "allergy" || plan.Exclusions[0].Token != "peanut" {
		t.Fatalf("exclusions: expected allergy/peanut, got %q/%q",
			plan.Exclusions[0].ReasonCode, plan.Exclusions[0].Token)
	}
	var cautions []string
	if err := json.Unmarshal(plan.Cautions, &cautions); err != nil || len(cautions) != 1 {
		t.Fatalf("cautions: expected a readable list, got %q (%v)", plan.Cautions, err)
	}

	// The assignment itself is completed at the same version as the plan it
	// produced, so a read and a write can never disagree about what was
	// delivered.
	assignment, err := h.assignmentRepo.FindByUserAndProgram(h.ctx, client.ID, program.ID)
	if err != nil {
		t.Fatalf("read assignment: %v", err)
	}
	if assignment.Status != models.NutritionAssignmentStatusCompleted || assignment.Version != plan.Version {
		t.Fatalf("assignment: expected completed at version %d, got %q at %d",
			plan.Version, assignment.Status, assignment.Version)
	}

	// 2. Another owner gets the same answer as "no plan", so the read is not an
	// ownership oracle.
	otherClient, _, _ := h.seedAccount(t, "plan-idor")
	if _, err := h.planRepo.FindActiveByUserAndProgram(h.ctx, otherClient.ID, program.ID); !errors.Is(err, repositories.ErrNutritionPlanNotFound) {
		t.Fatalf("cross-owner read: expected ErrNutritionPlanNotFound, got %v", err)
	}
	if _, err := h.planRepo.FindActiveByUserAndProgram(h.ctx, client.ID, "program-does-not-exist"); !errors.Is(err, repositories.ErrNutritionPlanNotFound) {
		t.Fatalf("unknown program read: expected ErrNutritionPlanNotFound, got %v", err)
	}

	// 3. An explicit regeneration inserts the next version and supersedes the
	// previous one; the plan the client was following is retained, never
	// rewritten.
	h.claim(t, client.ID, program.ID, questionnaire)
	second := newPlanBody(questionnaire)
	second.Fingerprint = "fingerprint-second-" + questionnaire.ID
	if err := h.planRepo.Complete(h.ctx, client.ID, program.ID, second); err != nil {
		t.Fatalf("complete second version: %v", err)
	}

	versions, err := h.planRepo.ListVersions(h.ctx, first.AssignmentID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("versions: expected 2, got %d", len(versions))
	}
	if versions[0].Version != plan.Version+1 || versions[1].Version != plan.Version {
		t.Fatalf("versions: expected newest first %d then %d, got %d then %d",
			plan.Version+1, plan.Version, versions[0].Version, versions[1].Version)
	}
	if versions[1].Status != models.NutritionPlanStatusSuperseded {
		t.Fatalf("previous version: expected superseded, got %q", versions[1].Status)
	}
	if versions[1].Fingerprint != "fingerprint-"+questionnaire.ID {
		t.Fatalf("previous version: expected its own body to be retained, got %q", versions[1].Fingerprint)
	}
	if versions[0].Status != models.NutritionPlanStatusActive {
		t.Fatalf("new version: expected active, got %q", versions[0].Status)
	}
	if versions[1].ID == versions[0].ID {
		t.Fatal("supersession: expected a new row rather than an update in place")
	}

	current, err := h.planRepo.FindActiveByAssignment(h.ctx, first.AssignmentID)
	if err != nil {
		t.Fatalf("read active by assignment: %v", err)
	}
	if current.Version != plan.Version+1 {
		t.Fatalf("active version: expected %d, got %d", plan.Version+1, current.Version)
	}
	if len(current.Meals) != 2 || len(current.Meals[0].Items) != 2 {
		t.Fatalf("active version: expected the new meals and foods, got %d meals", len(current.Meals))
	}

	// 4. The database — not the application — refuses a second active plan for
	// the same assignment. The version is deliberately unused so only the
	// one-active-plan index can reject the row.
	duplicate := versions[0]
	duplicate.ID = ""
	duplicate.Version = versions[0].Version + 1
	duplicate.Status = models.NutritionPlanStatusActive
	if err := h.tx.Create(&duplicate).Error; err == nil {
		t.Fatal("two active plans: expected the unique active_plan index to reject the row")
	}

	// 5. A completion is refused when it does not match the intake the claim
	// pinned, so a plan can never be written for a revision it never read.
	h.claim(t, client.ID, program.ID, questionnaire)
	stale := newPlanBody(questionnaire)
	stale.QuestionnaireVersion = questionnaire.Version + 7
	if err := h.planRepo.Complete(h.ctx, client.ID, program.ID, stale); !errors.Is(err, repositories.ErrNutritionAssignmentNotRetryable) {
		t.Fatalf("stale revision: expected ErrNutritionAssignmentNotRetryable, got %v", err)
	}
	versions, err = h.planRepo.ListVersions(h.ctx, first.AssignmentID)
	if err != nil {
		t.Fatalf("list versions after refusal: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("stale revision: expected no plan to be written, got %d versions", len(versions))
	}

	// 6. A completion is refused when the assignment is not currently claimed
	// for generation, and when it belongs to somebody else.
	if err := h.assignmentRepo.MarkFailed(h.ctx, client.ID, program.ID, "GENERATOR_NO_ELIGIBLE_FOOD"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	unclaimed := newPlanBody(questionnaire)
	if err := h.planRepo.Complete(h.ctx, client.ID, program.ID, unclaimed); !errors.Is(err, repositories.ErrNutritionAssignmentNotRetryable) {
		t.Fatalf("unclaimed completion: expected ErrNutritionAssignmentNotRetryable, got %v", err)
	}
	if err := h.planRepo.Complete(h.ctx, otherClient.ID, program.ID, newPlanBody(questionnaire)); !errors.Is(err, repositories.ErrNutritionAssignmentNotFound) {
		t.Fatalf("cross-owner completion: expected ErrNutritionAssignmentNotFound, got %v", err)
	}
}
