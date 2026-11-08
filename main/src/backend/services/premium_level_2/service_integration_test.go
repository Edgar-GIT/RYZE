package premium_level_2_test

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
	"ryze/backend/services/premium_level_2"
)

func TestServiceRequiresOwnershipAndActivePurchase(t *testing.T) {
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
	programRepo := repositories.NewProgramRepository(tx)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	requestRepo := repositories.NewPremiumLevel2Repository(tx)
	svc := premium_level_2.NewService(requestRepo, programRepo, entitlementRepo)

	seedUser := func(label string) *models.User {
		user := &models.User{Email: fmt.Sprintf("premium-l2-svc-%s-%d@ryze.local", label, time.Now().UnixNano()), PasswordHash: "hash", FirstName: "Service", LastName: label}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}
	owner := seedUser("owner")
	attacker := seedUser("attacker")
	offer := &models.Program{Name: "L2 service offer", Type: models.ProgramTypePremium, ProductType: models.ProgramProductTypePremiumLevel2, Status: models.ProgramStatusPublished, PriceMinorUnits: 20000, Currency: "EUR"}
	if err := tx.Omit("TrainerID").Create(offer).Error; err != nil {
		t.Fatalf("create offer: %v", err)
	}
	entitlement := &models.Entitlement{}
	if err := entitlementRepo.Create(ctx, owner.ID, offer.ID, entitlement); err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	request, err := requestRepo.CreateRequest(ctx, owner.ID, offer.ID)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	if _, _, _, err := svc.GetClientRequest(ctx, attacker.ID, request.ID); !errors.Is(err, premium_level_2.ErrRequestNotFound) {
		t.Fatalf("expected foreign owner to be denied, got %v", err)
	}
	if _, _, err := svc.GetDeliveredNutrition(ctx, owner.ID, request.ID); !errors.Is(err, premium_level_2.ErrNutritionMissing) {
		t.Fatalf("expected no nutrition before delivery, got %v", err)
	}
	invalid, _ := json.Marshal(map[string]any{"primary_goal": "unknown"})
	if _, _, err := svc.SubmitIntake(ctx, owner.ID, request.ID, invalid); err == nil {
		t.Fatal("expected invalid intake to be rejected")
	}
	received, intake, err := svc.SubmitIntake(ctx, owner.ID, request.ID, encoded(t, validAnswers()))
	if err != nil {
		t.Fatalf("submit valid intake: %v", err)
	}
	if received.IntakeVersion != 1 || intake.Version != 1 {
		t.Fatalf("expected intake version 1, got request=%d intake=%d", received.IntakeVersion, intake.Version)
	}
	requests, err := svc.ListClientRequests(ctx, owner.ID)
	if err != nil || len(requests) != 1 {
		t.Fatalf("expected one entitled request, got %d err=%v", len(requests), err)
	}
	if err := entitlementRepo.SoftDelete(ctx, owner.ID, entitlement.ID); err != nil {
		t.Fatalf("revoke entitlement: %v", err)
	}
	if _, _, _, err := svc.GetClientRequest(ctx, owner.ID, request.ID); !errors.Is(err, premium_level_2.ErrRequestNotFound) {
		t.Fatalf("expected revoked purchase to be hidden, got %v", err)
	}
	requests, err = svc.ListClientRequests(ctx, owner.ID)
	if err != nil || len(requests) != 0 {
		t.Fatalf("expected revoked request to be hidden, got %d err=%v", len(requests), err)
	}
}
