package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// TestPremiumJourneyRepository covers the Premium Level 1 pick candidates
// (published curated packages of the objective the client does not own).
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
	tall := repositories.MatchProfile{Objective: objective, Sex: female, HeightCm: 181, WeightKg: 70, Age: 30}
	// v2.133.0: a package only for 150–170 cm, and a new version of the owned
	// package; neither is offered to this client.
	shortOnly := makePackage("Short only", models.ProgramStatusPublished, nil)
	if err := tx.Model(shortOnly).Updates(map[string]any{"min_height_cm": 150, "max_height_cm": 170}).Error; err != nil {
		t.Fatalf("ranges: %v", err)
	}
	ownedV2 := makePackage("Owned, version 2", models.ProgramStatusPublished, nil)
	if err := tx.Model(ownedV2).Updates(map[string]any{"version_root_id": owned.ID, "previous_version_id": owned.ID}).Error; err != nil {
		t.Fatalf("version: %v", err)
	}
	candidates, err := repo.ListMatchCandidates(ctx, user.ID, tall)
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
	short := tall
	short.HeightCm = 160
	withShort, err := repo.ListMatchCandidates(ctx, user.ID, short)
	if err != nil || len(withShort) != 3 {
		t.Fatalf("a 160 cm client also gets the 150–170 cm package: %d, %v", len(withShort), err)
	}
	total, err := repo.CountPublishedPackages(ctx, repositories.MatchProfile{Objective: objective, Sex: male, HeightCm: 175, WeightKg: 80, Age: 30})
	if err != nil || total < 3 {
		t.Fatalf("CountPublishedPackages = %d, %v", total, err)
	}
	if err := repo.CreateMatch(ctx, &models.PremiumLevel1Match{UserID: user.ID, ProgramID: free.ID, Objective: objective, WeightKg: 80, HeightCm: 180, Age: 30}); err != nil {
		t.Fatalf("CreateMatch: %v", err)
	}
}
