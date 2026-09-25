package auth_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/middleware/adminroles"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
	"ryze/backend/services/purchases"
	"ryze/backend/services/test_mode"
	"ryze/backend/services/token"
)

func newTestModeTestRouter(t *testing.T, enabled bool) (*gin.Engine, repositories.UserRepository, *gorm.DB, token.Service) {
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
	trainerRepo := repositories.NewTrainerRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	purchaseRepo := repositories.NewPurchaseRepository(tx)
	entitlementRepo := repositories.NewEntitlementRepository(tx)
	sessionRepo := repositories.NewTestSessionRepository(tx)

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	testModeSvc := test_mode.NewService(enabled, sessionRepo, userRepo, trainerRepo, password.Hasher{})
	testModeHandler := auth.NewTestModeHandler(testModeSvc, tokenSvc, userRepo, testTokenTTL, false)

	purchaseSvc := purchases.NewService(programRepo, purchaseRepo, entitlementRepo, nil, nil, nil)
	testModePurchaseHandler := auth.NewTestModePurchaseHandler(testModeSvc, purchaseSvc)
	adminMeHandler := auth.NewAdminMeHandler()

	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.POST("/admin/auth/test-mode",
		middleware.AdminAuthenticate(tokenSvc),
		middleware.RequireAdminRole(adminroles.RoleTechnicalAdministrator),
		testModeHandler.Enter)
	v1.POST("/admin/auth/test-mode/exit", testModeHandler.Exit)
	v1.GET("/auth/test-mode", testModeHandler.Status)
	v1.GET("/admin/auth/me", middleware.AdminAuthenticate(tokenSvc), adminMeHandler.GetMe)
	v1.POST("/auth/test-mode/programs/:programID/purchase",
		middleware.Authenticate(tokenSvc, userRepo),
		testModePurchaseHandler.Purchase)

	return router, userRepo, tx, tokenSvc
}

func testModeRequest(router http.Handler, method, path, body string, cookies map[string]string) (*httptest.ResponseRecorder, map[string]any, string) {
	var reqBody *bytes.Reader
	if body == "" {
		reqBody = bytes.NewReader(nil)
	} else {
		reqBody = bytes.NewReader([]byte(body))
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for name, value := range cookies {
		if value == "" {
			continue
		}
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		return rec, nil, ""
	}
	data, _ := payload["data"].(map[string]any)
	return rec, data, string(rec.Body.Bytes())
}

func responseCookies(rec *httptest.ResponseRecorder) map[string]string {
	result := rec.Result()
	defer result.Body.Close()
	cookies := make(map[string]string)
	for _, c := range result.Cookies() {
		cookies[c.Name] = c.Value
	}
	return cookies
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func testModeAdminToken(t *testing.T, tokenSvc token.Service) string {
	t.Helper()
	raw, err := tokenSvc.GenerateAdminToken("ADMIN_1")
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}
	return raw
}

func enterTestMode(t *testing.T, router http.Handler, tokenSvc token.Service, persona string) (map[string]string, string) {
	t.Helper()
	body := `{"persona":` + strconvQuote(persona) + `,"return_path":""}`
	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode", body, map[string]string{
		auth.AdminAccessTokenCookieName: testModeAdminToken(t, tokenSvc),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 entering Test Mode, got %d (body: %s)", rec.Code, raw)
	}
	cookies := responseCookies(rec)
	if cookies[auth.TestSessionTokenCookieName] == "" {
		t.Fatal("expected the test session cookie to be set")
	}
	if cookies[auth.AdminAccessTokenCookieName] != "" {
		t.Fatal("expected the admin session cookie to be cleared on enter")
	}
	if cookies[auth.AccessTokenCookieName] == "" {
		t.Fatal("expected a persona access token to be issued")
	}
	return cookies, raw
}

func strconvQuote(v string) string {
	return `"` + v + `"`
}

func seedPurchasableProgram(t *testing.T, tx *gorm.DB, programType, status string) (*models.Program, string) {
	t.Helper()
	ctx := context.Background()
	userRepo := repositories.NewUserRepository(tx)
	trainerRepo := repositories.NewTrainerRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)

	trainerUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, trainerUser)

	program := &models.Program{
		TrainerID:       trainer.ID,
		Name:            "Test Mode Premium Program",
		Type:            programType,
		Status:          status,
		PriceMinorUnits: 15000,
		Currency:        "EUR",
	}
	if err := programRepo.Create(ctx, program); err != nil {
		t.Fatalf("seed program: %v", err)
	}
	return program, trainerUser.ID
}

func personaUserID(t *testing.T, userRepo repositories.UserRepository) string {
	t.Helper()
	user, err := userRepo.FindByEmail(context.Background(), test_mode.PersonaClientEmail)
	if err != nil {
		t.Fatalf("find persona user: %v", err)
	}
	return user.ID
}

func TestTestModeEnterAndStatusActive(t *testing.T) {
	router, userRepo, tx, tokenSvc := newTestModeTestRouter(t, true)
	_ = tx
	cookies, raw := enterTestMode(t, router, tokenSvc, "client")

	rec, data, _ := testModeRequest(router, http.MethodGet, "/api/v1/auth/test-mode", "", map[string]string{
		auth.TestSessionTokenCookieName: cookies[auth.TestSessionTokenCookieName],
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body body: %s)", rec.Code, raw)
	}
	active, _ := data["active"].(bool)
	if !active {
		t.Fatal("expected Test Mode to be reported as active")
	}
	if persona, _ := data["persona"].(string); persona != "client" {
		t.Fatalf("expected persona client, got %q", persona)
	}

	sessionRepo := repositories.NewTestSessionRepository(tx)
	session, err := sessionRepo.FindActiveByTokenHash(context.Background(), tokenHash(cookies[auth.TestSessionTokenCookieName]))
	if err != nil {
		t.Fatalf("find active session: %v", err)
	}
	if session.TokenHash == cookies[auth.TestSessionTokenCookieName] {
		t.Fatal("the raw session token must never be persisted")
	}
	if session.Persona != "client" {
		t.Fatalf("expected session persona client, got %q", session.Persona)
	}

	personaUser, err := userRepo.FindByEmail(context.Background(), test_mode.PersonaClientEmail)
	if err != nil {
		t.Fatalf("find persona user: %v", err)
	}
	if session.PersonaUserID != personaUser.ID {
		t.Fatalf("session must reference the persona user, got %q want %q", session.PersonaUserID, personaUser.ID)
	}
}

func TestTestModeEnterWhenDisabled(t *testing.T) {
	router, _, _, tokenSvc := newTestModeTestRouter(t, false)

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode", `{"persona":"client","return_path":""}`, map[string]string{
		auth.AdminAccessTokenCookieName: testModeAdminToken(t, tokenSvc),
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when disabled, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TEST_MODE_DISABLED"`) {
		t.Fatalf("expected TEST_MODE_DISABLED, got %s", raw)
	}
}

func TestTestModeEnterRequiresAuthentication(t *testing.T) {
	router, _, _, _ := newTestModeTestRouter(t, true)

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode", `{"persona":"client","return_path":""}`, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without admin auth, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTestModeEnterRestrictedToTechnicalAdministrator(t *testing.T) {
	router, _, _, tokenSvc := newTestModeTestRouter(t, true)

	admin2Token, err := tokenSvc.GenerateAdminToken("ADMIN_2")
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode", `{"persona":"client","return_path":""}`, map[string]string{
		auth.AdminAccessTokenCookieName: admin2Token,
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a non-technical admin, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
		t.Fatalf("expected FORBIDDEN, got %s", raw)
	}
}

func TestTestModeEnterRejectsUnknownPersona(t *testing.T) {
	router, _, _, tokenSvc := newTestModeTestRouter(t, true)

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode", `{"persona":"hacker","return_path":""}`, map[string]string{
		auth.AdminAccessTokenCookieName: testModeAdminToken(t, tokenSvc),
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unknown persona, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTestModeEnterRejectsMaliciousReturnPath(t *testing.T) {
	router, _, _, tokenSvc := newTestModeTestRouter(t, true)

	invalid := []string{
		"https://evil.com",
		"http://evil.com/path",
		"//evil.com",
		"\\\\evil.com",
		"javascript:alert(1)",
		"evil.com",
		"/path\x00with-control",
		"/" + strings.Repeat("a", 513),
	}
	for _, returnPath := range invalid {
		body := `{"persona":"client","return_path":` + strconvQuote(returnPath) + `}`
		rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode", body, map[string]string{
			auth.AdminAccessTokenCookieName: testModeAdminToken(t, tokenSvc),
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for return path %q, got %d (body: %s)", returnPath, rec.Code, raw)
		}
	}
}

func TestTestModeEnterAllowsSafeReturnPath(t *testing.T) {
	router, _, _, tokenSvc := newTestModeTestRouter(t, true)

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode", `{"persona":"client","return_path":"/programs/abc"}`, map[string]string{
		auth.AdminAccessTokenCookieName: testModeAdminToken(t, tokenSvc),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a root-relative return path, got %d (body: %s)", rec.Code, raw)
	}
}

func TestTestModeEnterReusesPersonaAccount(t *testing.T) {
	router, userRepo, tx, tokenSvc := newTestModeTestRouter(t, true)

	first, _ := enterTestMode(t, router, tokenSvc, "client")
	second, _ := enterTestMode(t, router, tokenSvc, "client")

	firstID := personaUserID(t, userRepo)

	sessionRepo := repositories.NewTestSessionRepository(tx)
	firstSession, err := sessionRepo.FindActiveByTokenHash(context.Background(), tokenHash(first[auth.TestSessionTokenCookieName]))
	if err != nil {
		t.Fatalf("find first session: %v", err)
	}
	secondSession, err := sessionRepo.FindActiveByTokenHash(context.Background(), tokenHash(second[auth.TestSessionTokenCookieName]))
	if err != nil {
		t.Fatalf("find second session: %v", err)
	}

	if firstID != firstSession.PersonaUserID || firstSession.PersonaUserID != secondSession.PersonaUserID {
		t.Fatalf("both enters must resolve to the same persona account, got %q and %q", firstSession.PersonaUserID, secondSession.PersonaUserID)
	}

	var count int64
	if err := tx.Model(&models.User{}).Where("email = ?", test_mode.PersonaClientEmail).Count(&count).Error; err != nil {
		t.Fatalf("count persona users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one persona account, got %d", count)
	}
}

func TestTestModeTrainerPersonaCreatesLinkedTrainer(t *testing.T) {
	router, userRepo, tx, tokenSvc := newTestModeTestRouter(t, true)

	cookies, _ := enterTestMode(t, router, tokenSvc, "trainer")
	_ = cookies

	personaUser, err := userRepo.FindByEmail(context.Background(), test_mode.PersonaTrainerEmail)
	if err != nil {
		t.Fatalf("find trainer persona user: %v", err)
	}

	trainerRepo := repositories.NewTrainerRepository(tx)
	trainer, err := trainerRepo.FindByUserID(context.Background(), personaUser.ID)
	if err != nil {
		t.Fatalf("trainer persona must have a linked trainer profile: %v", err)
	}
	if trainer.UserID != personaUser.ID {
		t.Fatalf("expected trainer linked to persona user, got %q want %q", trainer.UserID, personaUser.ID)
	}
}

func TestTestModeExitRestoresAdminSession(t *testing.T) {
	router, _, _, tokenSvc := newTestModeTestRouter(t, true)

	cookies, _ := enterTestMode(t, router, tokenSvc, "client")

	rec, data, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode/exit", "", map[string]string{
		auth.TestSessionTokenCookieName: cookies[auth.TestSessionTokenCookieName],
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if returnPath, _ := data["return_path"].(string); returnPath != "" {
		t.Fatalf("expected empty return path, got %q", returnPath)
	}

	after := responseCookies(rec)
	if after[auth.TestSessionTokenCookieName] != "" {
		t.Fatal("expected the test session cookie to be cleared on exit")
	}
	if after[auth.AccessTokenCookieName] != "" {
		t.Fatal("expected the persona access token cookie to be cleared on exit")
	}
	adminTokenAfter := after[auth.AdminAccessTokenCookieName]
	if adminTokenAfter == "" {
		t.Fatal("expected the admin session cookie to be restored on exit")
	}

	// The restored token must be a live admin session for the original identity.
	meRec, _, meRaw := testModeRequest(router, http.MethodGet, "/api/v1/admin/auth/me", "", map[string]string{
		auth.AdminAccessTokenCookieName: adminTokenAfter,
	})
	if meRec.Code != http.StatusOK {
		t.Fatalf("restored admin session must be valid, got %d (body: %s)", meRec.Code, meRaw)
	}
	if !strings.Contains(meRaw, `"id":"ADMIN_1"`) {
		t.Fatalf("expected restored identity ADMIN_1, got %s", meRaw)
	}

	// After exit, Test Mode is inactive again.
	statusRec, statusData, _ := testModeRequest(router, http.MethodGet, "/api/v1/auth/test-mode", "", nil)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected 200 status, got %d", statusRec.Code)
	}
	if active, _ := statusData["active"].(bool); active {
		t.Fatal("expected Test Mode to be inactive after exit")
	}
}

func TestTestModeExitWithoutSessionCookie(t *testing.T) {
	router, _, _, tokenSvc := newTestModeTestRouter(t, true)

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode/exit", "", map[string]string{
		auth.AdminAccessTokenCookieName: testModeAdminToken(t, tokenSvc),
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without a session cookie, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TEST_MODE_INACTIVE"`) {
		t.Fatalf("expected TEST_MODE_INACTIVE, got %s", raw)
	}
}

func TestTestModeExitWithUnknownToken(t *testing.T) {
	router, _, _, _ := newTestModeTestRouter(t, true)

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/admin/auth/test-mode/exit", "", map[string]string{
		auth.TestSessionTokenCookieName: "deadbeef",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for an unknown session token, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TEST_MODE_INACTIVE"`) {
		t.Fatalf("expected TEST_MODE_INACTIVE, got %s", raw)
	}
}

func TestTestModePurchaseCompletesWithoutProvider(t *testing.T) {
	router, userRepo, tx, tokenSvc := newTestModeTestRouter(t, true)

	cookies, _ := enterTestMode(t, router, tokenSvc, "client")
	personaID := personaUserID(t, userRepo)
	program, _ := seedPurchasableProgram(t, tx, models.ProgramTypePremium, models.ProgramStatusPublished)

	rec, data, raw := testModeRequest(router, http.MethodPost, "/api/v1/auth/test-mode/programs/"+program.ID+"/purchase", "", map[string]string{
		auth.AccessTokenCookieName:      cookies[auth.AccessTokenCookieName],
		auth.TestSessionTokenCookieName: cookies[auth.TestSessionTokenCookieName],
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if status, _ := data["status"].(string); status != models.PurchaseStatusCompleted {
		t.Fatalf("expected a completed purchase, got %q", status)
	}
	if price, _ := data["price_minor_units"].(float64); price != 0 {
		t.Fatalf("expected a zero test price, got %v", price)
	}

	entitlementRepo := repositories.NewEntitlementRepository(tx)
	if _, err := entitlementRepo.FindActiveByUserAndProgram(context.Background(), personaID, program.ID); err != nil {
		t.Fatalf("test purchase must create a real entitlement: %v", err)
	}

	purchaseRepo := repositories.NewPurchaseRepository(tx)
	purchaseID, _ := data["id"].(string)
	persisted, err := purchaseRepo.FindByID(context.Background(), purchaseID)
	if err != nil {
		t.Fatalf("load persisted purchase: %v", err)
	}
	if !persisted.Test {
		t.Fatal("expected the persisted purchase to carry the test marker")
	}
	if persisted.PriceMinorUnits != 0 {
		t.Fatalf("expected zero test price persisted, got %d", persisted.PriceMinorUnits)
	}
	if persisted.UserID != personaID {
		t.Fatalf("expected purchase owned by the persona, got %q want %q", persisted.UserID, personaID)
	}
}

func TestTestModePurchaseDuplicateEntitlement(t *testing.T) {
	router, _, tx, tokenSvc := newTestModeTestRouter(t, true)

	cookies, _ := enterTestMode(t, router, tokenSvc, "client")
	program, _ := seedPurchasableProgram(t, tx, models.ProgramTypePremium, models.ProgramStatusPublished)

	purchaseCookies := map[string]string{
		auth.AccessTokenCookieName:      cookies[auth.AccessTokenCookieName],
		auth.TestSessionTokenCookieName: cookies[auth.TestSessionTokenCookieName],
	}
	path := "/api/v1/auth/test-mode/programs/" + program.ID + "/purchase"

	first, _, rawFirst := testModeRequest(router, http.MethodPost, path, "", purchaseCookies)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected 201 on first purchase, got %d (body: %s)", first.Code, rawFirst)
	}

	second, _, rawSecond := testModeRequest(router, http.MethodPost, path, "", purchaseCookies)
	if second.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate purchase, got %d (body: %s)", second.Code, rawSecond)
	}
	if !strings.Contains(rawSecond, `"code":"DUPLICATE_ENTITLEMENT"`) {
		t.Fatalf("expected DUPLICATE_ENTITLEMENT, got %s", rawSecond)
	}
}

func TestTestModePurchaseRejectsFreeProgram(t *testing.T) {
	router, _, tx, tokenSvc := newTestModeTestRouter(t, true)

	cookies, _ := enterTestMode(t, router, tokenSvc, "client")
	program, _ := seedPurchasableProgram(t, tx, models.ProgramTypeFree, models.ProgramStatusPublished)

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/auth/test-mode/programs/"+program.ID+"/purchase", "", map[string]string{
		auth.AccessTokenCookieName:      cookies[auth.AccessTokenCookieName],
		auth.TestSessionTokenCookieName: cookies[auth.TestSessionTokenCookieName],
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for a free program, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_PURCHASABLE"`) {
		t.Fatalf("expected PROGRAM_NOT_PURCHASABLE, got %s", raw)
	}
}

func TestTestModePurchaseRejectsUnknownProgram(t *testing.T) {
	router, _, tx, tokenSvc := newTestModeTestRouter(t, true)
	_ = tx

	cookies, _ := enterTestMode(t, router, tokenSvc, "client")

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/auth/test-mode/programs/44444444-4444-4444-4444-444444444444/purchase", "", map[string]string{
		auth.AccessTokenCookieName:      cookies[auth.AccessTokenCookieName],
		auth.TestSessionTokenCookieName: cookies[auth.TestSessionTokenCookieName],
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for an unknown program, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestTestModePurchaseRejectsNonPersonaUser(t *testing.T) {
	router, userRepo, tx, tokenSvc := newTestModeTestRouter(t, true)

	program, _ := seedPurchasableProgram(t, tx, models.ProgramTypePremium, models.ProgramStatusPublished)
	regularUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	regularJWT, err := tokenSvc.GenerateAccessToken(regularUser.ID, regularUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	// A regular authenticated user must never complete a test purchase even
	// without any session cookie.
	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/auth/test-mode/programs/"+program.ID+"/purchase", "", map[string]string{
		auth.AccessTokenCookieName: regularJWT,
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a non-persona user, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TEST_MODE_INACTIVE"`) {
		t.Fatalf("expected TEST_MODE_INACTIVE, got %s", raw)
	}
}

func TestTestModePurchaseRejectsForeignPersonaIdentity(t *testing.T) {
	router, userRepo, tx, tokenSvc := newTestModeTestRouter(t, true)

	cookies, _ := enterTestMode(t, router, tokenSvc, "client")
	program, _ := seedPurchasableProgram(t, tx, models.ProgramTypePremium, models.ProgramStatusPublished)

	// A different authenticated user holding another user's session cookie must
	// still be rejected: the session identity is bound to the persona.
	regularUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	foreignJWT, err := tokenSvc.GenerateAccessToken(regularUser.ID, regularUser.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/auth/test-mode/programs/"+program.ID+"/purchase", "", map[string]string{
		auth.AccessTokenCookieName:      foreignJWT,
		auth.TestSessionTokenCookieName: cookies[auth.TestSessionTokenCookieName],
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a forged persona identity, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"TEST_MODE_INACTIVE"`) {
		t.Fatalf("expected TEST_MODE_INACTIVE, got %s", raw)
	}
}

func TestTestModePurchaseRequiresAuthentication(t *testing.T) {
	router, _, tx, _ := newTestModeTestRouter(t, true)

	program, _ := seedPurchasableProgram(t, tx, models.ProgramTypePremium, models.ProgramStatusPublished)

	rec, _, raw := testModeRequest(router, http.MethodPost, "/api/v1/auth/test-mode/programs/"+program.ID+"/purchase", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}
