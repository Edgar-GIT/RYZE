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

// TestPremiumJourneyRepository covers the Premium Level 1 pick candidates
// (published curated packages of the objective the client does not own) and
// the Premium Level 2 pre-purchase questionnaire upsert.
func TestPremiumJourneyRepository(t *testing.T) {
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

	user := &models.User{Email: fmt.Sprintf("journey-%d@ryze.local", time.Now().UnixNano()), PasswordHash: "prepared-hash", FirstName: "Journey", LastName: "Client"}
	if err := repositories.NewUserRepository(tx).Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	objective := models.ProgramObjectiveSportSpecific
	curated := models.ProgramNutritionModeCurated
	makePackage := func(name, status string, audience *string) *models.Program {
		program := &models.Program{
			Name: name, Type: models.ProgramTypePremium, Status: status,
			ProductType: models.ProgramProductTypePremiumLevel1, Objective: &objective, NutritionMode: &curated,
			PriceMinorUnits: models.PremiumLevel1PriceMinorUnits, Currency: "EUR", Audience: audience,
		}
		if err := tx.Omit("TrainerID", "Weeks", "CuratedNutrition").Create(program).Error; err != nil {
			t.Fatalf("create package: %v", err)
		}
		return program
	}
	female, male := models.SexFemale, models.SexMale
	owned := makePackage("Owned", models.ProgramStatusPublished, nil)
	free := makePackage("Available", models.ProgramStatusPublished, nil)
	forHer := makePackage("For women", models.ProgramStatusPublished, &female)
	makePackage("For men", models.ProgramStatusPublished, &male)
	makePackage("Draft", models.ProgramStatusDraft, nil)
	if err := tx.Create(&models.Entitlement{UserID: user.ID, ProgramID: owned.ID}).Error; err != nil {
		t.Fatalf("create entitlement: %v", err)
	}

	repo := repositories.NewPremiumJourneyRepository(tx)
	candidates, err := repo.ListMatchCandidates(ctx, user.ID, objective, female)
	if err != nil {
		t.Fatalf("ListMatchCandidates: %v", err)
	}
	ids := map[string]bool{}
	for _, candidate := range candidates {
		ids[candidate.ID] = true
	}
	// Unowned and published, for everyone or for women: never the owned
	// package, the draft or the package for men.
	if len(candidates) != 2 || !ids[free.ID] || !ids[forHer.ID] {
		t.Fatalf("candidates = %v, want the unowned package for everyone and the one for women", candidates)
	}
	total, err := repo.CountPublishedPackages(ctx, objective, male)
	if err != nil || total < 3 {
		t.Fatalf("CountPublishedPackages = %d, %v", total, err)
	}
	if err := repo.CreateMatch(ctx, &models.PremiumLevel1Match{UserID: user.ID, ProgramID: free.ID, Objective: objective, WeightKg: 80, HeightCm: 180, Age: 30}); err != nil {
		t.Fatalf("CreateMatch: %v", err)
	}

	offerStatus := models.ProgramStatusPublished
	offer := &models.Program{Name: "Offer", Type: models.ProgramTypePremium, Status: offerStatus, ProductType: models.ProgramProductTypePremiumLevel2, PriceMinorUnits: models.PremiumLevel2PriceMinorUnits, Currency: "EUR"}
	if err := tx.Omit("TrainerID", "Weeks", "CuratedNutrition").Create(offer).Error; err != nil {
		t.Fatalf("create offer: %v", err)
	}
	if _, err := repo.FindDraft(ctx, user.ID, offer.ID); !errors.Is(err, repositories.ErrPremiumLevel2DraftNotFound) {
		t.Fatalf("FindDraft before saving = %v", err)
	}
	for _, answers := range []string{`{"primary_goal":"strength"}`, `{"primary_goal":"muscle_gain"}`} {
		if err := repo.SaveDraft(ctx, &models.PremiumLevel2IntakeDraft{UserID: user.ID, ProgramID: offer.ID, Answers: json.RawMessage(answers)}); err != nil {
			t.Fatalf("SaveDraft: %v", err)
		}
	}
	draft, err := repo.FindDraft(ctx, user.ID, offer.ID)
	if err != nil {
		t.Fatalf("FindDraft: %v", err)
	}
	var answers map[string]string
	_ = json.Unmarshal(draft.Answers, &answers)
	if answers["primary_goal"] != "muscle_gain" {
		t.Fatalf("draft not replaced: %s", draft.Answers)
	}
	if err := repo.DeleteDraft(ctx, user.ID, offer.ID); err != nil {
		t.Fatalf("DeleteDraft: %v", err)
	}
}
