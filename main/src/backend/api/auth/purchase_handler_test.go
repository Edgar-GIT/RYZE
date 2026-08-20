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
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/commission_rules"
	"ryze/backend/services/purchases"
	"ryze/backend/services/token"
)

const purchaseRoute = "/api/v1/me/programs/"

// newPurchaseTestRouter wires the purchase endpoint behind the real
// Authenticate middleware, backed by a database transaction so created records
// are rolled back.
func newPurchaseTestRouter(t *testing.T) (*gin.Engine, repositories.UserRepository, *gorm.DB, token.Service) {
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

	svc := purchases.NewService(programRepo, purchaseRepo, entitlementRepo, &commissionAdapter{svc: commissionSvc})
	handler := auth.NewPurchaseHandler(svc)

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(middleware.Authenticate(tokenSvc, userRepo))
	me.POST("/programs/:programID/purchase", handler.CreatePurchase)

	return router, userRepo, tx, tokenSvc
}

// newPurchaseHandlerRouter mounts only the handler with a pre-set
// authentication-context identity, so the handler's own error mapping and
// identity forwarding can be tested without the full middleware chain. nil
// identity simulates a missing context.
func newPurchaseHandlerRouter(svc purchases.Service, identity any) *gin.Engine {
	handler := auth.NewPurchaseHandler(svc)
	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(authcontext.UserIDContextKey, identity)
		}
		c.Next()
	})
	me.POST("/programs/:programID/purchase", handler.CreatePurchase)
	return router
}

// stubPurchaseService is a scripted fake used to exercise the handler's error
// mapping and identity forwarding without touching the database.
type stubPurchaseService struct {
	purchase *purchases.Purchase
	err      error
	gotUser  string
	gotProg  string
}

func (s *stubPurchaseService) CreatePurchaseIntent(_ context.Context, userID, programID string) (*purchases.Purchase, error) {
	s.gotUser = userID
	s.gotProg = programID
	return s.purchase, s.err
}

// commissionAdapter adapts commission_rules.Service to the
// purchases.CommissionResolver interface for integration tests.
type commissionAdapter struct {
	svc commission_rules.Service
}

func (a *commissionAdapter) ResolveCommission(ctx context.Context, trainerID string) (purchases.CommissionResolution, error) {
	res, err := a.svc.ResolveCommission(ctx, trainerID)
	if err != nil {
		return purchases.CommissionResolution{}, err
	}
	return purchases.CommissionResolution{
		CommissionBPS: res.CommissionBPS,
		IsOverride:    res.IsOverride,
	}, nil
}

func (a *commissionAdapter) CalculateCommissionSplit(priceMinorUnits int64, resolution purchases.CommissionResolution) purchases.CommissionCalculation {
	res := commission_rules.CommissionResolution{
		CommissionBPS: resolution.CommissionBPS,
		IsOverride:    resolution.IsOverride,
	}
	calc := a.svc.CalculateCommissionSplit(priceMinorUnits, res)
	return purchases.CommissionCalculation{
		PlatformAmount: calc.PlatformAmount,
		TrainerAmount:  calc.TrainerAmount,
	}
}

func TestPurchaseHandlerForwardsContextIdentity(t *testing.T) {
	identity := "33333333-3333-3333-3333-333333333333"
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{
			ID:     "00000000-0000-0000-0000-000000000001",
			Status: models.PurchaseStatusPending,
		},
	}
	router := newPurchaseHandlerRouter(svc, identity)

	programID := "11111111-1111-1111-1111-111111111111"
	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, purchaseRoute+programID+"/purchase", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotUser != identity {
		t.Fatalf("expected context user %q, got %q", identity, svc.gotUser)
	}
	if svc.gotProg != programID {
		t.Fatalf("expected program id %q, got %q", programID, svc.gotProg)
	}
}

func TestPurchaseHandlerMissingContext(t *testing.T) {
	router := newPurchaseHandlerRouter(&stubPurchaseService{}, nil)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, purchaseRoute+"11111111-1111-1111-1111-111111111111/purchase", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestPurchaseHandlerErrorMapping(t *testing.T) {
	identity := "33333333-3333-3333-3333-333333333333"

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: purchases.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "program not found", err: purchases.ErrProgramNotFound, status: http.StatusNotFound, code: "PROGRAM_NOT_FOUND"},
		{name: "program not purchasable", err: purchases.ErrProgramNotPurchasable, status: http.StatusConflict, code: "PROGRAM_NOT_PURCHASABLE"},
		{name: "duplicate entitlement", err: purchases.ErrDuplicateEntitlement, status: http.StatusConflict, code: "DUPLICATE_ENTITLEMENT"},
		{name: "duplicate purchase", err: purchases.ErrDuplicatePurchase, status: http.StatusConflict, code: "DUPLICATE_PURCHASE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubPurchaseService{err: tc.err}
			router := newPurchaseHandlerRouter(svc, identity)

			rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, purchaseRoute+"11111111-1111-1111-1111-111111111111/purchase", "")
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestPurchaseHandlerRepositoryFailureNotExposed(t *testing.T) {
	svc := &stubPurchaseService{err: errLoginRepoFailure}
	router := newPurchaseHandlerRouter(svc, "33333333-3333-3333-3333-333333333333")

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, purchaseRoute+"11111111-1111-1111-1111-111111111111/purchase", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "repository failure") {
		t.Fatalf("internal error details must never be exposed, got %s", raw)
	}
	if !strings.Contains(raw, `"code":"INTERNAL_ERROR"`) {
		t.Fatalf("expected INTERNAL_ERROR, got %s", raw)
	}
}

func TestPurchaseHandlerIgnoresClientSuppliedIdentity(t *testing.T) {
	identity := "33333333-3333-3333-3333-333333333333"
	otherUser := "44444444-4444-4444-4444-444444444444"
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{
			ID:     "00000000-0000-0000-0000-000000000001",
			Status: models.PurchaseStatusPending,
		},
	}
	router := newPurchaseHandlerRouter(svc, identity)

	// A client-supplied user_id in the body must be ignored: the purchase is
	// always created for the authenticated user.
	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost,
		purchaseRoute+"11111111-1111-1111-1111-111111111111/purchase",
		`{"user_id":"`+otherUser+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotUser != identity {
		t.Fatalf("expected context user %q, got %q", identity, svc.gotUser)
	}
}

func TestPurchaseHandlerResponseNeverExposesSensitiveData(t *testing.T) {
	identity := "33333333-3333-3333-3333-333333333333"
	svc := &stubPurchaseService{
		purchase: &purchases.Purchase{
			ID:              "00000000-0000-0000-0000-000000000001",
			UserID:          identity,
			ProgramID:       "11111111-1111-1111-1111-111111111111",
			PriceMinorUnits: 10000,
			Currency:        "EUR",
			CommissionBPS:   2000,
			PlatformAmount:  2000,
			TrainerAmount:   8000,
			Status:          models.PurchaseStatusPending,
		},
	}
	router := newPurchaseHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, purchaseRoute+"11111111-1111-1111-1111-111111111111/purchase", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	for _, sensitive := range []string{
		"password",
		"access_token",
		testSecret,
		"deleted_at",
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}

// --- Integration tests (real database) ---

func TestPurchaseIntegrationSuccess(t *testing.T) {
	router, userRepo, tx, tokenSvc := newPurchaseTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	trainerRepo := repositories.NewTrainerRepository(tx)
	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, trainerUser)

	programRepo := repositories.NewProgramRepository(tx)
	program := &models.Program{
		TrainerID:       trainer.ID,
		Name:            "Premium Program",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 10000,
		Currency:        "EUR",
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, purchaseRoute+program.ID+"/purchase", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	if id, _ := data["id"].(string); id == "" {
		t.Fatal("expected purchase id")
	}
	if status, _ := data["status"].(string); status != models.PurchaseStatusPending {
		t.Fatalf("expected status %q, got %q", models.PurchaseStatusPending, status)
	}
	if priceMinorUnits, _ := data["price_minor_units"].(float64); priceMinorUnits != 10000 {
		t.Fatalf("expected price 10000, got %v", priceMinorUnits)
	}
	if currency, _ := data["currency"].(string); currency != "EUR" {
		t.Fatalf("expected currency EUR, got %s", currency)
	}
	if platformAmount, _ := data["platform_amount"].(float64); platformAmount != 2000 {
		t.Fatalf("expected platform amount 2000, got %v", platformAmount)
	}
	if trainerAmount, _ := data["trainer_amount"].(float64); trainerAmount != 8000 {
		t.Fatalf("expected trainer amount 8000, got %v", trainerAmount)
	}

	// The owning user id must never be exposed to the client.
	if _, exists := data["user_id"]; exists {
		t.Fatal("response must never expose the owning user id")
	}
}

func TestPurchaseIntegrationUnauthenticated(t *testing.T) {
	router, _, _, _ := newPurchaseTestRouter(t)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, purchaseRoute+"11111111-1111-1111-1111-111111111111/purchase", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestPurchaseIntegrationFreeProgram(t *testing.T) {
	router, userRepo, tx, tokenSvc := newPurchaseTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	trainerRepo := repositories.NewTrainerRepository(tx)
	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, trainerUser)

	programRepo := repositories.NewProgramRepository(tx)
	program := &models.Program{
		TrainerID:       trainer.ID,
		Name:            "Free Program",
		Type:            models.ProgramTypeFree,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 0,
		Currency:        "EUR",
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, purchaseRoute+program.ID+"/purchase", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_PURCHASABLE"`) {
		t.Fatalf("expected PROGRAM_NOT_PURCHASABLE, got %s", raw)
	}
}

func TestPurchaseIntegrationDuplicateEntitlement(t *testing.T) {
	router, userRepo, tx, tokenSvc := newPurchaseTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	trainerRepo := repositories.NewTrainerRepository(tx)
	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, trainerUser)

	programRepo := repositories.NewProgramRepository(tx)
	program := &models.Program{
		TrainerID:       trainer.ID,
		Name:            "Premium Program",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 10000,
		Currency:        "EUR",
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	entitlementRepo := repositories.NewEntitlementRepository(tx)
	if err := entitlementRepo.Create(context.Background(), clientUser.ID, program.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("seed entitlement: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, purchaseRoute+program.ID+"/purchase", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"DUPLICATE_ENTITLEMENT"`) {
		t.Fatalf("expected DUPLICATE_ENTITLEMENT, got %s", raw)
	}
}

func TestPurchaseIntegrationDuplicatePendingPurchase(t *testing.T) {
	router, userRepo, tx, tokenSvc := newPurchaseTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	trainerRepo := repositories.NewTrainerRepository(tx)
	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, trainerUser)

	programRepo := repositories.NewProgramRepository(tx)
	program := &models.Program{
		TrainerID:       trainer.ID,
		Name:            "Premium Program",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 10000,
		Currency:        "EUR",
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	purchaseRepo := repositories.NewPurchaseRepository(tx)
	existingPurchase := &models.Purchase{
		UserID:          clientUser.ID,
		ProgramID:       program.ID,
		PriceMinorUnits: 10000,
		Currency:        "EUR",
		Status:          models.PurchaseStatusPending,
	}
	if err := purchaseRepo.Create(context.Background(), existingPurchase); err != nil {
		t.Fatalf("seed purchase: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, purchaseRoute+program.ID+"/purchase", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"DUPLICATE_PURCHASE"`) {
		t.Fatalf("expected DUPLICATE_PURCHASE, got %s", raw)
	}
}

func TestPurchaseIntegrationProgramNotFound(t *testing.T) {
	router, userRepo, _, tokenSvc := newPurchaseTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, purchaseRoute+"11111111-1111-1111-1111-111111111111/purchase", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestPurchaseIntegrationDraftProgram(t *testing.T) {
	router, userRepo, tx, tokenSvc := newPurchaseTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	trainerRepo := repositories.NewTrainerRepository(tx)
	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, trainerUser)

	programRepo := repositories.NewProgramRepository(tx)
	program := &models.Program{
		TrainerID:       trainer.ID,
		Name:            "Draft Program",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusDraft,
		PriceMinorUnits: 10000,
		Currency:        "EUR",
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, purchaseRoute+program.ID+"/purchase", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for draft program, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestPurchaseIntegrationNeverExposesSensitiveData(t *testing.T) {
	router, userRepo, tx, tokenSvc := newPurchaseTestRouter(t)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	trainerRepo := repositories.NewTrainerRepository(tx)
	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, trainerUser)

	programRepo := repositories.NewProgramRepository(tx)
	program := &models.Program{
		TrainerID:       trainer.ID,
		Name:            "Premium Program",
		Type:            models.ProgramTypePremium,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 10000,
		Currency:        "EUR",
	}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	jwtValue, err := tokenSvc.GenerateAccessToken(clientUser.ID, clientUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, purchaseRoute+program.ID+"/purchase", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	for _, sensitive := range []string{
		jwtValue,
		"access_token",
		testSecret,
		"password_hash",
		"session_version",
		"deleted_at",
		"trainer_id",
		clientUser.Email,
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}
