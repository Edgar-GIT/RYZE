package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/middleware/adminroles"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_commerce"
	"ryze/backend/services/token"
)

const (
	adminCommerceListRoute     = "/api/v1/admin/purchases"
	adminCommerceDetailRoute   = "/api/v1/admin/purchases/%s"
	adminCommerceProgramsRoute = "/api/v1/admin/purchases/programs"
)

// newAdminCommerceTestRouter wires the admin commerce endpoints behind the real
// admin authentication and authorization middleware. It is backed by a database
// transaction so seeded records are rolled back on cleanup.
func newAdminCommerceTestRouter(t *testing.T) (*gin.Engine, repositories.UserRepository, *gorm.DB, token.Service) {
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
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	purchaseRepo := repositories.NewPurchaseRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	commerceSvc := admin_commerce.NewService(purchaseRepo, programRepo)
	commerceHandler := auth.NewAdminCommerceHandler(commerceSvc)

	router := gin.New()
	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.AdminAuthenticate(tokenSvc))
	adminCommerce := admin.Group("")
	adminCommerce.Use(middleware.RequireAdminPermission(adminroles.PermissionCommerce))
	// The static /purchases/programs path must be registered before
	// /purchases/:id so the router resolves both without ambiguity.
	adminCommerce.GET("/purchases/programs", commerceHandler.ListProgramSales)
	adminCommerce.GET("/purchases", commerceHandler.ListPurchases)
	adminCommerce.GET("/purchases/:id", commerceHandler.GetPurchase)

	return router, userRepo, tx, tokenSvc
}

// seedCommercePurchase creates a purchase with an explicit status, price and
// creation time so tests can exercise filters and the dashboard metrics. A
// user may complete at most one purchase per program, so tests must spread
// completed purchases across buyers.
func seedCommercePurchase(t *testing.T, tx *gorm.DB, userID, programID string, status string, test bool, price int64, createdAt time.Time) *models.Purchase {
	t.Helper()
	purchaseRepo := repositories.NewPurchaseRepository(tx)
	purchase := &models.Purchase{
		UserID:          userID,
		ProgramID:       programID,
		PriceMinorUnits: price,
		Currency:        "EUR",
		Status:          status,
		Test:            test,
		CreatedAt:       createdAt,
	}
	if err := purchaseRepo.Create(context.Background(), purchase); err != nil {
		t.Fatalf("seed purchase: %v", err)
	}
	return purchase
}

// adminCommerceRequest performs an admin-commerce request and unwraps the data
// envelope for assertions.
func adminCommerceRequest(router http.Handler, cookieValue, method, path string) (*httptest.ResponseRecorder, map[string]any, string) {
	rec, raw := adminUsersRequest(router, cookieValue, method, path)
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return rec, nil, raw
	}
	data, _ := payload["data"].(map[string]any)
	return rec, data, raw
}

func TestAdminCommerceAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cookie     string
		wantStatus int
	}{
		{name: "no cookie", wantStatus: http.StatusUnauthorized},
		{name: "client user cookie", cookie: "some-jwt", wantStatus: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, _, _, _ := newAdminCommerceTestRouter(t)
			rec, raw := adminUsersRequest(router, tc.cookie, http.MethodGet, adminCommerceListRoute)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d (body: %s)", tc.wantStatus, rec.Code, raw)
			}
		})
	}
}

func TestAdminCommerceForbiddenForTechnicalAdministrator(t *testing.T) {
	router, _, _, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin1ID)

	rec, raw := adminUsersRequest(router, cookie, http.MethodGet, adminCommerceListRoute)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminCommerceListPurchasesAndSummary(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	program, _ := seedPremiumProgram(t, tx, userRepo)
	now := time.Now().UTC()

	buyerA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	buyerB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	buyerC := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	buyerD := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	seedCommercePurchase(t, tx, buyerA.ID, program.ID, models.PurchaseStatusCompleted, false, 10000, now.Add(-3*time.Hour))
	seedCommercePurchase(t, tx, buyerB.ID, program.ID, models.PurchaseStatusCompleted, false, 10000, now.Add(-2*time.Hour))
	seedCommercePurchase(t, tx, buyerC.ID, program.ID, models.PurchaseStatusPending, false, 10000, now.Add(-1*time.Hour))
	seedCommercePurchase(t, tx, buyerD.ID, program.ID, models.PurchaseStatusCompleted, true, 10000, now)

	rec, data, raw := adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceListRoute)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	summary := data["summary"].(map[string]any)
	if fmt.Sprint(summary["completed_count"]) != "3" {
		t.Fatalf("expected 3 completed including test, got %v", summary["completed_count"])
	}
	if fmt.Sprint(summary["pending_count"]) != "1" {
		t.Fatalf("expected 1 pending, got %v", summary["pending_count"])
	}
	if fmt.Sprint(summary["failed_count"]) != "0" {
		t.Fatalf("expected 0 failed, got %v", summary["failed_count"])
	}
	if fmt.Sprint(summary["real_revenue_minor_units"]) != "20000" {
		t.Fatalf("expected real revenue 20000 (test excluded), got %v", summary["real_revenue_minor_units"])
	}
	if fmt.Sprint(summary["test_purchase_count"]) != "1" {
		t.Fatalf("expected 1 test purchase, got %v", summary["test_purchase_count"])
	}
	if fmt.Sprint(summary["programs_with_sales"]) != "1" {
		t.Fatalf("expected 1 program with sales, got %v", summary["programs_with_sales"])
	}

	purchases := data["purchases"].([]any)
	if len(purchases) != 4 {
		t.Fatalf("expected 4 purchases, got %d", len(purchases))
	}
	// Newest (test) purchase leads the list.
	first := purchases[0].(map[string]any)
	customer := first["customer"].(map[string]any)
	if customer["email"] != buyerD.Email {
		t.Fatalf("customer email not exposed: %v", customer["email"])
	}
	if fmt.Sprint(first["test"]) != "true" {
		t.Fatalf("test purchase must be flagged, got %v", first["test"])
	}
	programObj := first["program"].(map[string]any)
	if programObj["id"] != program.ID || fmt.Sprint(programObj["deleted"]) != "false" {
		t.Fatalf("program summary not mapped: %v", programObj)
	}
}

func TestAdminCommerceListAppliesFilters(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	programA, _ := seedPremiumProgram(t, tx, userRepo)
	programB, _ := seedPremiumProgram(t, tx, userRepo)

	buyerA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	buyerB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	buyerC := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	now := time.Now().UTC()
	seedCommercePurchase(t, tx, buyerA.ID, programA.ID, models.PurchaseStatusCompleted, false, 10000, now)
	seedCommercePurchase(t, tx, buyerB.ID, programA.ID, models.PurchaseStatusPending, false, 10000, now)
	seedCommercePurchase(t, tx, buyerC.ID, programB.ID, models.PurchaseStatusCompleted, false, 10000, now)

	rec, data, raw := adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceListRoute+"?status=completed&program_id="+programA.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if len(data["purchases"].([]any)) != 1 {
		t.Fatalf("expected 1 completed purchase of program A, got %s", raw)
	}

	rec, data, raw = adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceListRoute+"?test=false")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if len(data["purchases"].([]any)) != 3 {
		t.Fatalf("expected all non-test purchases, got %s", raw)
	}

	rec, data, raw = adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceListRoute+"?from=2026-01-01&to=2030-01-01")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if len(data["purchases"].([]any)) != 3 {
		t.Fatalf("expected purchases within date range, got %s", raw)
	}
}

func TestAdminCommerceListRejectsInvalidQueries(t *testing.T) {
	router, _, _, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	for _, path := range []string{
		adminCommerceListRoute + "?status=refunded",
		adminCommerceListRoute + "?test=banana",
		adminCommerceListRoute + "?program_id=not-a-uuid",
		adminCommerceListRoute + "?from=2026-13-99",
		adminCommerceListRoute + "?from=2026-02-01&to=2026-01-01",
	} {
		rec, raw := adminUsersRequest(router, cookie, http.MethodGet, path)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path %s: expected 400, got %d (body: %s)", path, rec.Code, raw)
		}
	}
}

func TestAdminCommerceListPaginates(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	program, _ := seedPremiumProgram(t, tx, userRepo)
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		buyer := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
		seedCommercePurchase(t, tx, buyer.ID, program.ID, models.PurchaseStatusCompleted, false, 10000, now.Add(-time.Duration(i)*time.Hour))
	}

	rec, data, raw := adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceListRoute+"?page=1&limit=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	pagination := data["pagination"].(map[string]any)
	if fmt.Sprint(pagination["total"]) != "3" || fmt.Sprint(pagination["total_pages"]) != "3" {
		t.Fatalf("unexpected pagination metadata: %s", raw)
	}
	if len(data["purchases"].([]any)) != 1 {
		t.Fatalf("expected 1 purchase per page, got %s", raw)
	}

	rec, data, raw = adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceListRoute+"?limit=9999")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if fmt.Sprint(data["pagination"].(map[string]any)["limit"]) != "100" {
		t.Fatalf("expected limit clamped to 100, got %s", raw)
	}
}

func TestAdminCommerceDetail(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	buyer := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, _ := seedPremiumProgram(t, tx, userRepo)
	completed := seedCompletedPurchase(t, tx, buyer.ID, program.ID, false)

	rec, data, raw := adminCommerceRequest(router, cookie, http.MethodGet, fmt.Sprintf(adminCommerceDetailRoute, completed.ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	purchase := data["purchase"].(map[string]any)
	if fmt.Sprint(purchase["access"]) != "true" {
		t.Fatalf("completed purchase of published program must grant access: %s", raw)
	}

	rec, raw = adminUsersRequest(router, cookie, http.MethodGet, fmt.Sprintf(adminCommerceDetailRoute, "00000000-0000-0000-0000-00000000dead"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
	}
}

func TestAdminCommerceDetailRetiredProgramKeepsHistory(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	buyer := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, trainer := seedPremiumProgram(t, tx, userRepo)
	purchase := seedCompletedPurchase(t, tx, buyer.ID, program.ID, false)

	programRepo := repositories.NewProgramRepository(tx)
	if err := programRepo.SoftDelete(context.Background(), trainer.ID, program.ID); err != nil {
		t.Fatalf("soft delete program: %v", err)
	}

	rec, data, raw := adminCommerceRequest(router, cookie, http.MethodGet, fmt.Sprintf(adminCommerceDetailRoute, purchase.ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	purchaseObj := data["purchase"].(map[string]any)
	programObj := purchaseObj["program"].(map[string]any)
	if fmt.Sprint(programObj["deleted"]) != "true" {
		t.Fatalf("retired program must be flagged deleted: %s", raw)
	}
	if programObj["id"] != program.ID || programObj["name"] != program.Name {
		t.Fatalf("retired program must keep its identity: %s", raw)
	}
	if fmt.Sprint(purchaseObj["access"]) != "false" {
		t.Fatalf("retired program must not grant access: %s", raw)
	}
}

func TestAdminCommerceSoftDeletedBuyerKeepsName(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	buyer := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, _ := seedPremiumProgram(t, tx, userRepo)
	purchase := seedCompletedPurchase(t, tx, buyer.ID, program.ID, false)

	if err := userRepo.SoftDelete(context.Background(), buyer.ID); err != nil {
		t.Fatalf("soft delete buyer: %v", err)
	}

	rec, data, raw := adminCommerceRequest(router, cookie, http.MethodGet, fmt.Sprintf(adminCommerceDetailRoute, purchase.ID))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	customer := data["purchase"].(map[string]any)["customer"].(map[string]any)
	if customer["name"] != buyer.FirstName+" "+buyer.LastName {
		t.Fatalf("soft-deleted buyer must keep its name, got %v", customer["name"])
	}
	if customer["email"] != buyer.Email {
		t.Fatalf("soft-deleted buyer email must remain visible: %s", raw)
	}
}

func TestAdminCommerceProgramSales(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	programA, _ := seedPremiumProgram(t, tx, userRepo)
	programB, _ := seedPremiumProgram(t, tx, userRepo)
	now := time.Now().UTC()

	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programA.ID, models.PurchaseStatusCompleted, false, 10000, now)
	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programA.ID, models.PurchaseStatusCompleted, false, 10000, now)
	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programA.ID, models.PurchaseStatusCompleted, true, 10000, now)
	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programA.ID, models.PurchaseStatusPending, false, 10000, now)
	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programB.ID, models.PurchaseStatusCompleted, false, 10000, now)

	rec, data, raw := adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceProgramsRoute)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	sales := data["sales"].([]any)
	if len(sales) != 2 {
		t.Fatalf("expected 2 program sales rows, got %d (%s)", len(sales), raw)
	}

	// Program A has the most completed sales (including its test purchase), so
	// it must lead the list.
	top := sales[0].(map[string]any)
	if fmt.Sprint(top["completed_sales"]) != "3" {
		t.Fatalf("top program must have 3 completed sales including test, got %s", raw)
	}
	if fmt.Sprint(top["revenue_minor_units"]) != "20000" {
		t.Fatalf("expected revenue 20000 (test excluded), got %s", raw)
	}
	if fmt.Sprint(top["test_purchases"]) != "1" || fmt.Sprint(top["pending_purchases"]) != "1" {
		t.Fatalf("program A counts wrong: %s", raw)
	}
	if top["program"].(map[string]any)["id"] != programA.ID {
		t.Fatalf("top row must be program A: %s", raw)
	}

	rec, data, raw = adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceProgramsRoute+"?program_id="+programB.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if len(data["sales"].([]any)) != 1 {
		t.Fatalf("expected only program B, got %s", raw)
	}
}

func TestAdminCommerceProgramSalesBatch(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	programA, _ := seedPremiumProgram(t, tx, userRepo)
	programB, _ := seedPremiumProgram(t, tx, userRepo)
	// programC has no purchases at all.
	programC, _ := seedPremiumProgram(t, tx, userRepo)
	now := time.Now().UTC()

	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programA.ID, models.PurchaseStatusCompleted, false, 10000, now)
	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programA.ID, models.PurchaseStatusCompleted, false, 10000, now)
	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programA.ID, models.PurchaseStatusCompleted, true, 10000, now)
	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programA.ID, models.PurchaseStatusPending, false, 10000, now)
	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, programB.ID, models.PurchaseStatusCompleted, false, 10000, now)

	unknownID := "00000000-0000-0000-0000-00000000dead"
	rec, data, raw := adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceProgramsRoute+"?program_ids="+programA.ID+","+programC.ID+","+unknownID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	sales := data["sales"].([]any)
	if len(sales) != 3 {
		t.Fatalf("expected one row per requested program, got %d (%s)", len(sales), raw)
	}

	// Request order must be preserved.
	if sales[0].(map[string]any)["program"].(map[string]any)["id"] != programA.ID {
		t.Fatalf("first row must be program A: %s", raw)
	}
	rowA := sales[0].(map[string]any)
	if fmt.Sprint(rowA["real_sales"]) != "2" {
		t.Fatalf("program A must report 2 real sales (test excluded), got %s", raw)
	}
	if fmt.Sprint(rowA["completed_sales"]) != "3" {
		t.Fatalf("program A completed keeps the inclusive count, got %s", raw)
	}
	if fmt.Sprint(rowA["revenue_minor_units"]) != "20000" {
		t.Fatalf("program A revenue must exclude the test purchase, got %s", raw)
	}
	if fmt.Sprint(rowA["test_purchases"]) != "1" || fmt.Sprint(rowA["pending_purchases"]) != "1" {
		t.Fatalf("program A counts wrong: %s", raw)
	}

	if sales[1].(map[string]any)["program"].(map[string]any)["id"] != programC.ID {
		t.Fatalf("second row must be program C: %s", raw)
	}
	rowC := sales[1].(map[string]any)
	if fmt.Sprint(rowC["real_sales"]) != "0" || fmt.Sprint(rowC["revenue_minor_units"]) != "0" || fmt.Sprint(rowC["test_purchases"]) != "0" {
		t.Fatalf("program C must be zero-filled: %s", raw)
	}

	// An unknown program id still resolves to an attributable zero row.
	rowU := sales[2].(map[string]any)
	if rowU["program"].(map[string]any)["id"] != unknownID || fmt.Sprint(rowU["real_sales"]) != "0" {
		t.Fatalf("unknown program must resolve as a zero row: %s", raw)
	}

	pagination := data["pagination"].(map[string]any)
	if fmt.Sprint(pagination["total"]) != "3" || fmt.Sprint(pagination["total_pages"]) != "1" {
		t.Fatalf("batch pagination must reflect the requested set: %s", raw)
	}
}

func TestAdminCommerceProgramSalesBatchRejectsMalformedIDs(t *testing.T) {
	router, _, _, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	for _, path := range []string{
		adminCommerceProgramsRoute + "?program_ids=not-a-uuid",
		adminCommerceProgramsRoute + "?program_ids=00000000-0000-0000-0000-000000000001,not-a-uuid",
	} {
		rec, raw := adminUsersRequest(router, cookie, http.MethodGet, path)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("path %s: expected 400, got %d (body: %s)", path, rec.Code, raw)
		}
	}
}

func TestAdminCommerceProgramSalesBatchKeepsRetiredProgram(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	program, trainer := seedPremiumProgram(t, tx, userRepo)
	seedCommercePurchase(t, tx, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!").ID, program.ID, models.PurchaseStatusCompleted, false, 10000, time.Now().UTC())

	programRepo := repositories.NewProgramRepository(tx)
	if err := programRepo.SoftDelete(context.Background(), trainer.ID, program.ID); err != nil {
		t.Fatalf("soft delete program: %v", err)
	}

	rec, data, raw := adminCommerceRequest(router, cookie, http.MethodGet, adminCommerceProgramsRoute+"?program_ids="+program.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	sales := data["sales"].([]any)
	if len(sales) != 1 {
		t.Fatalf("expected the retired program row, got %d (%s)", len(sales), raw)
	}
	row := sales[0].(map[string]any)
	programPayload := row["program"].(map[string]any)
	if programPayload["id"] != program.ID || programPayload["name"] != program.Name {
		t.Fatalf("retired program must keep its identity: %s", raw)
	}
	if fmt.Sprint(programPayload["deleted"]) != "true" {
		t.Fatalf("retired program must be flagged deleted: %s", raw)
	}
	if fmt.Sprint(row["real_sales"]) != "1" || fmt.Sprint(row["revenue_minor_units"]) != "10000" {
		t.Fatalf("retired program must keep its historical sales: %s", raw)
	}
}

func TestAdminCommerceResponsesUseSafeShape(t *testing.T) {
	router, userRepo, tx, tokenSvc := newAdminCommerceTestRouter(t)
	cookie := adminToken(t, tokenSvc, config.Admin2ID)

	buyer := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	program, _ := seedPremiumProgram(t, tx, userRepo)
	seedCompletedPurchase(t, tx, buyer.ID, program.ID, false)

	var payload struct {
		Data struct {
			Purchases []map[string]any `json:"purchases"`
		} `json:"data"`
	}
	_, raw := adminUsersRequest(router, cookie, http.MethodGet, adminCommerceListRoute)
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v (body: %s)", err, raw)
	}
	first := payload.Data.Purchases[0]

	for _, forbidden := range []string{"user_id", "trainer_id", "password_hash", "session_version", "deleted_at", "commission_bps", "platform_amount", "trainer_amount"} {
		if _, ok := first[forbidden]; ok {
			t.Fatalf("sensitive key %q leaked: %s", forbidden, raw)
		}
	}

	customer := first["customer"].(map[string]any)
	if _, ok := customer["id"]; ok {
		t.Fatalf("customer must not expose its internal id: %s", raw)
	}
	programPayload := first["program"].(map[string]any)
	for _, forbidden := range []string{"trainer_id", "deleted_at", "description", "created_by"} {
		if _, ok := programPayload[forbidden]; ok {
			t.Fatalf("sensitive program key %q leaked: %s", forbidden, raw)
		}
	}
}
