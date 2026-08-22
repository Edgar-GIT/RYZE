package auth_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/commission_rules"
	"ryze/backend/services/payments"
	"ryze/backend/services/purchases"
	"ryze/backend/services/token"
)

func newE2EPurchaseRouter(t *testing.T) (*gin.Engine, purchases.Service, repositories.EntitlementRepository, repositories.PurchaseRepository, *gorm.DB) {
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

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("retrieve database handle: %v", err)
	}
	tx := db.Begin()
	t.Cleanup(func() {
		_ = tx.Rollback()
		_ = sqlDB.Close()
	})

	userRepo := repositories.NewUserRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	purchaseRepo := repositories.NewPurchaseRepository(tx)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	commissionRuleRepo := repositories.NewCommissionRuleRepository(tx)
	trainerRepo := repositories.NewTrainerRepository(tx)

	commissionCfg := config.CommissionConfig{DefaultPlatformCommissionBPS: 2000}
	commissionSvc := commission_rules.NewService(commissionRuleRepo, trainerRepo, commissionCfg)

	svc := purchases.NewService(
		programRepo,
		purchaseRepo,
		entitlementRepo,
		&commissionAdapter{svc: commissionSvc},
		payments.NewFakeProvider(),
		func(_ context.Context, _ payments.PaymentMethod) (payments.Provider, error) {
			return payments.NewFakeProvider(), nil
		},
	)

	handler := auth.NewPurchaseHandler(svc)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(middleware.Authenticate(tokenSvc, userRepo))
	me.POST("/programs/:programID/purchase", handler.CreatePurchase)
	me.POST("/purchases/:purchaseID/payment", handler.InitiatePayment)

	return router, svc, entitlementRepo, purchaseRepo, tx
}

func TestE2EFullPurchaseFlow(t *testing.T) {
	router, svc, entitlementRepo, purchaseRepo, tx := newE2EPurchaseRouter(t)
	userRepo := repositories.NewUserRepository(tx)
	trainerRepo := repositories.NewTrainerRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)

	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, trainerUser)

	program := &models.Program{
		TrainerID:       trainer.ID,
		Name:            "E2E Premium Program",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 15000,
		Currency:        "EUR",
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	// Step 1: Create purchase intent
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, purchaseRoute+program.ID+"/purchase", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("Step 1 - expected 201, got %d (body: %s)", rec.Code, raw)
	}

	purchaseID, _ := data["id"].(string)
	if purchaseID == "" {
		t.Fatal("Step 1 - expected purchase id")
	}
	if s, _ := data["status"].(string); s != models.PurchaseStatusPending {
		t.Fatalf("Step 1 - expected status %q, got %q", models.PurchaseStatusPending, s)
	}
	if p, _ := data["price_minor_units"].(float64); p != 15000 {
		t.Fatalf("Step 1 - expected price 15000, got %v", p)
	}
	if pa, _ := data["platform_amount"].(float64); pa != 3000 {
		t.Fatalf("Step 1 - expected platform amount 3000, got %v", pa)
	}
	if ta, _ := data["trainer_amount"].(float64); ta != 12000 {
		t.Fatalf("Step 1 - expected trainer amount 12000, got %v", ta)
	}

	// Step 2: Initiate payment
	rec, payData, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, paymentRoute+purchaseID+"/payment", `{"payment_method":"card"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Step 2 - expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if pid, _ := payData["payment_id"].(string); pid == "" {
		t.Fatal("Step 2 - expected payment id")
	}
	if pp, _ := payData["purchase_id"].(string); pp != purchaseID {
		t.Fatalf("Step 2 - expected purchase id %q, got %q", purchaseID, pp)
	}

	// Step 2b: Verify purchase is still pending after initiation, snapshot immutable
	pm, err := purchaseRepo.FindByID(context.Background(), purchaseID)
	if err != nil {
		t.Fatalf("Step 2b - find purchase: %v", err)
	}
	if pm.Status != models.PurchaseStatusPending {
		t.Fatalf("Step 2b - purchase must still be pending, got %q", pm.Status)
	}
	if pm.PriceMinorUnits != 15000 {
		t.Fatalf("Step 2b - snapshot price must be 15000, got %d", pm.PriceMinorUnits)
	}
	if pm.Currency != "EUR" {
		t.Fatalf("Step 2b - snapshot currency must be EUR, got %s", pm.Currency)
	}
	if pm.CommissionBPS != 2000 {
		t.Fatalf("Step 2b - snapshot commission must be 2000, got %d", pm.CommissionBPS)
	}
	if pm.PlatformAmount != 3000 {
		t.Fatalf("Step 2b - snapshot platform amount must be 3000, got %d", pm.PlatformAmount)
	}
	if pm.TrainerAmount != 12000 {
		t.Fatalf("Step 2b - snapshot trainer amount must be 12000, got %d", pm.TrainerAmount)
	}

	// Step 3: Simulate webhook completion (this is what Stripe/PayPal webhooks do)
	completedPurchase, err := svc.CompletePurchase(context.Background(), purchaseID)
	if err != nil {
		t.Fatalf("Step 3 - CompletePurchase: %v", err)
	}
	if completedPurchase.Status != models.PurchaseStatusCompleted {
		t.Fatalf("Step 3 - expected status %q, got %q", models.PurchaseStatusCompleted, completedPurchase.Status)
	}

	// Step 3b: Verify snapshot intact after completion
	if completedPurchase.PriceMinorUnits != 15000 {
		t.Fatalf("Step 3b - snapshot price must be 15000, got %d", completedPurchase.PriceMinorUnits)
	}
	if completedPurchase.CommissionBPS != 2000 {
		t.Fatalf("Step 3b - snapshot commission must be 2000, got %d", completedPurchase.CommissionBPS)
	}
	if completedPurchase.PlatformAmount != 3000 {
		t.Fatalf("Step 3b - snapshot platform amount must be 3000, got %d", completedPurchase.PlatformAmount)
	}
	if completedPurchase.TrainerAmount != 12000 {
		t.Fatalf("Step 3b - snapshot trainer amount must be 12000, got %d", completedPurchase.TrainerAmount)
	}

	// Step 4: Verify entitlement was created
	entitlements, err := entitlementRepo.ListByUser(context.Background(), clientUser.ID)
	if err != nil {
		t.Fatalf("Step 4 - list entitlements: %v", err)
	}
	if len(entitlements) != 1 {
		t.Fatalf("Step 4 - expected 1 entitlement, got %d", len(entitlements))
	}
	if entitlements[0].ProgramID != program.ID {
		t.Fatalf("Step 4 - expected entitlement for program %q, got %q", program.ID, entitlements[0].ProgramID)
	}
	if entitlements[0].UserID != clientUser.ID {
		t.Fatalf("Step 4 - expected entitlement for user %q, got %q", clientUser.ID, entitlements[0].UserID)
	}

	// Step 5: Verify idempotency - calling CompletePurchase again succeeds without duplicating entitlements
	completedAgain, err := svc.CompletePurchase(context.Background(), purchaseID)
	if err != nil {
		t.Fatalf("Step 5 - idempotent CompletePurchase: %v", err)
	}
	if completedAgain.Status != models.PurchaseStatusCompleted {
		t.Fatalf("Step 5 - expected status %q on idempotent call, got %q", models.PurchaseStatusCompleted, completedAgain.Status)
	}
	entitlementsAfter, err := entitlementRepo.ListByUser(context.Background(), clientUser.ID)
	if err != nil {
		t.Fatalf("Step 5 - list entitlements: %v", err)
	}
	if len(entitlementsAfter) != 1 {
		t.Fatalf("Step 5 - expected still 1 entitlement after idempotent call, got %d", len(entitlementsAfter))
	}

	// Step 6: Verify no new payment can be initiated after completion
	rec, _, raw = trainerClientsRequest(router, jwtValue, http.MethodPost, paymentRoute+purchaseID+"/payment", `{"payment_method":"card"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("Step 6 - expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, "PURCHASE_NOT_PENDING") {
		t.Fatalf("Step 6 - expected PURCHASE_NOT_PENDING, got %s", raw)
	}
}
