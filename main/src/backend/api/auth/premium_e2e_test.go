package auth_test

import (
	"bytes"
	"context"
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
	"ryze/backend/services/commission_rules"
	"ryze/backend/services/nutrition_assignment"
	"ryze/backend/services/password"
	"ryze/backend/services/payments"
	"ryze/backend/services/purchases"
	"ryze/backend/services/questionnaires"
	"ryze/backend/services/test_mode"
	"ryze/backend/services/token"
)

const (
	premiumQuestionnaireRoute = "/api/v1/me/programs/%s/questionnaire"
	premiumQuestionsRoute     = "/api/v1/me/questionnaire/questions"
	premiumNutritionRoute     = "/api/v1/me/programs/%s/nutrition"
	premiumGenerateRoute      = "/api/v1/me/programs/%s/nutrition/generate"
)

// premiumFlowFixture is the wired stack a Premium Level 1 journey is exercised
// against, using the real middleware, repositories and database.
type premiumFlowFixture struct {
	router    *gin.Engine
	tx        *gorm.DB
	tokens    token.Service
	programs  repositories.ProgramRepository
	purchases purchases.Service
	nutrition nutrition_assignment.Service
}

func newPremiumFlowRouter(t *testing.T) *premiumFlowFixture {
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
	questionnaireRepo := repositories.NewNutritionQuestionnaireRepository(tx)
	assignmentRepo := repositories.NewNutritionAssignmentRepository(tx)

	commissionSvc := commission_rules.NewService(
		commissionRuleRepo, trainerRepo, config.CommissionConfig{DefaultPlatformCommissionBPS: 2000},
	)

	questionnaireSvc := questionnaires.NewService(programRepo, questionnaireRepo, entitlementRepo)
	questionnaireHandler := auth.NewQuestionnaireHandler(questionnaireSvc)

	nutritionSvc := nutrition_assignment.NewService(
		programRepo, entitlementRepo, questionnaireSvc, assignmentRepo,
		nutrition_assignment.NewDeterministicGenerator(),
	)
	nutritionHandler := auth.NewNutritionHandler(nutritionSvc)

	purchaseSvc := purchases.NewService(
		programRepo, purchaseRepo, entitlementRepo,
		&commissionAdapter{svc: commissionSvc},
		payments.NewFakeProvider(),
		func(_ context.Context, _ payments.PaymentMethod) (payments.Provider, error) {
			return payments.NewFakeProvider(), nil
		},
		purchases.WithCheckoutPrerequisites(questionnaireSvc),
	)
	purchaseHandler := auth.NewPurchaseHandler(purchaseSvc)

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	router := gin.New()
	me := router.Group("/api/v1/me")
	me.Use(middleware.Authenticate(tokenSvc, userRepo))
	me.GET("/questionnaire/questions", questionnaireHandler.GetQuestions)
	me.GET("/programs/:programID/questionnaire", questionnaireHandler.GetRequirement)
	me.POST("/programs/:programID/questionnaire", questionnaireHandler.Submit)
	me.GET("/programs/:programID/nutrition", nutritionHandler.GetStatus)
	me.POST("/programs/:programID/nutrition/generate", nutritionHandler.Generate)
	me.POST("/programs/:programID/purchase", purchaseHandler.CreatePurchase)
	me.POST("/purchases/:purchaseID/payment", purchaseHandler.InitiatePayment)
	me.POST("/purchases/:purchaseID/capture", purchaseHandler.CapturePayment)

	// Test Mode is a real purchase path, so it is wired into this fixture to
	// prove it cannot bypass the questionnaire gate or the post-purchase lock.
	testModeSvc := test_mode.NewService(
		true,
		repositories.NewTestSessionRepository(tx),
		userRepo,
		trainerRepo,
		password.Hasher{},
	)
	testModeHandler := auth.NewTestModeHandler(testModeSvc, tokenSvc, userRepo, testTokenTTL, false)
	testModePurchaseHandler := auth.NewTestModePurchaseHandler(testModeSvc, purchaseSvc)

	v1 := router.Group("/api/v1")
	v1.POST("/admin/auth/test-mode",
		middleware.AdminAuthenticate(tokenSvc),
		middleware.RequireAdminRole(adminroles.RoleTechnicalAdministrator),
		testModeHandler.Enter)
	v1.POST("/auth/test-mode/programs/:programID/purchase",
		middleware.Authenticate(tokenSvc, userRepo),
		testModePurchaseHandler.Purchase)

	return &premiumFlowFixture{
		router:    router,
		tx:        tx,
		tokens:    tokenSvc,
		programs:  programRepo,
		purchases: purchaseSvc,
		nutrition: nutritionSvc,
	}
}

// seedPremiumProgram creates a published, platform-owned Premium Level 1 program.
func (f *premiumFlowFixture) seedPremiumProgram(t *testing.T) *models.Program {
	t.Helper()

	program := &models.Program{
		Name:             "Premium Level 1 — Training + Nutrition",
		Description:      "A complete package combining a training programme with a personalised nutrition plan.",
		Type:             models.ProgramTypePremium,
		ProductType:      models.ProgramProductTypePremiumLevel1,
		Status:           models.ProgramStatusPublished,
		PriceMinorUnits:  12900,
		Currency:         "EUR",
		DurationWeeks:    intPtr(8),
		FrequencyPerWeek: intPtr(4),
		TrainingType:     strPtr("strength"),
	}
	// trainer_id is omitted so the column defaults to NULL, marking the program
	// platform-owned. This mirrors the production generic-program create path.
	if err := f.tx.Omit("TrainerID").Create(program).Error; err != nil {
		t.Fatalf("seed premium program: %v", err)
	}
	return program
}

// seedGenericProgram creates a published, platform-owned Generic program.
func (f *premiumFlowFixture) seedGenericProgram(t *testing.T) *models.Program {
	t.Helper()

	program := &models.Program{
		Name:            "Generic Training Plan",
		Description:     "A standard training plan.",
		Type:            models.ProgramTypePremium,
		ProductType:     models.ProgramProductTypeGeneric,
		Status:          models.ProgramStatusPublished,
		PriceMinorUnits: 4900,
		Currency:        "EUR",
	}
	if err := f.tx.Omit("TrainerID").Create(program).Error; err != nil {
		t.Fatalf("seed generic program: %v", err)
	}
	return program
}

type session struct {
	router *gin.Engine
	token  string
	user   *models.User
}

func (f *premiumFlowFixture) login(t *testing.T, email string) *session {
	t.Helper()

	userRepo := repositories.NewUserRepository(f.tx)
	user := seedLoginUser(t, userRepo, email, "Password123!")

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)
	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	return &session{router: f.router, token: jwtValue, user: user}
}

func (s *session) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: s.token})

	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)
	return recorder
}

// validIntakeBody is a complete, valid intake submission.
func validIntakeBody() map[string]any {
	return map[string]any{
		"goal":                   "fat_loss",
		"experience":             "beginner",
		"training_days_per_week": 4,
		"session_length_minutes": 60,
		"diet":                   "omnivore",
		"meals_per_day":          4,
		"snacks_per_day":         1,
		"cooking_effort":         "moderate",
		"body_weight_kg":         82.5,
		"body_height_cm":         178,
		"activity_level":         "moderate",
		"sleep_hours":            7,
		"stress_level":           "moderate",
		"allergies":              []string{"peanuts"},
		"medical_conditions":     []string{"type 1 diabetes"},
	}
}

func TestE2EPremiumQuestionnaireGateBlocksCheckoutUntilSubmitted(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)
	client := f.login(t, uniqueEmail())

	// A Premium Level 1 program must not be sellable before the intake exists.
	blocked := client.do(t, http.MethodPost, "/api/v1/me/programs/"+program.ID+"/purchase", nil)
	if blocked.Code != http.StatusConflict {
		t.Fatalf("purchase before intake = %d, want 409; body %s", blocked.Code, blocked.Body.String())
	}
	if code := errorCode(t, decodeBody(t, blocked.Body.String())); code != "PURCHASE_PREREQUISITE_NOT_MET" {
		t.Errorf("error code = %q, want PURCHASE_PREREQUISITE_NOT_MET", code)
	}

	// No purchase row may exist after a rejected checkout.
	var purchaseRows int64
	f.tx.Model(&models.Purchase{}).Where("program_id = ?", program.ID).Count(&purchaseRows)
	if purchaseRows != 0 {
		t.Errorf("expected no purchase row after a rejected checkout, found %d", purchaseRows)
	}

	submitted := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": validIntakeBody()})
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit questionnaire = %d, want 200; body %s", submitted.Code, submitted.Body.String())
	}
	if submitted := decodeBody(t, submitted.Body.String()); submitted["submitted"] != true {
		t.Errorf("submitted = %v, want true", submitted["submitted"])
	}

	// With the intake stored, checkout proceeds.
	allowed := client.do(t, http.MethodPost, "/api/v1/me/programs/"+program.ID+"/purchase", nil)
	if allowed.Code != http.StatusCreated {
		t.Fatalf("purchase after intake = %d, want 201; body %s", allowed.Code, allowed.Body.String())
	}
}

func TestE2EPremiumQuestionnaireNeverEchoesAnswers(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)
	client := f.login(t, uniqueEmail())

	response := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": validIntakeBody()})
	if response.Code != http.StatusOK {
		t.Fatalf("submit = %d; body %s", response.Code, response.Body.String())
	}

	// A submission acknowledgement carries state, never the health answers that
	// were just submitted.
	body := response.Body.String()
	for _, marker := range []string{"type 1 diabetes", "peanuts", "body_weight_kg", "82.5"} {
		if containsString(body, marker) {
			t.Errorf("submission response leaked %q: %s", marker, body)
		}
	}

	status := client.do(t, http.MethodGet, sprintfPath(premiumQuestionnaireRoute, program.ID), nil)
	if containsString(status.Body.String(), "type 1 diabetes") {
		t.Errorf("requirement response leaked intake content: %s", status.Body.String())
	}
}

func TestE2EPremiumValidationRejectsBadIntakeWithoutPersisting(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)
	client := f.login(t, uniqueEmail())

	invalid := validIntakeBody()
	invalid["diet"] = "carnivore-ish"
	delete(invalid, "body_weight_kg")

	response := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": invalid})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid submit = %d, want 400; body %s", response.Code, response.Body.String())
	}

	details := decodeBody(t, response.Body.String())["error"].(map[string]any)["details"].([]any)
	fields := make([]string, 0, len(details))
	for _, detail := range details {
		fields = append(fields, detail.(string))
	}
	joined := joinStrings(fields, ", ")
	for _, expected := range []string{"diet", "body_weight_kg"} {
		if !containsString(joined, expected) {
			t.Errorf("expected %q among rejection reasons, got %v", expected, fields)
		}
	}

	// A rejected intake must not be partially stored.
	var rows int64
	f.tx.Model(&models.NutritionQuestionnaire{}).Count(&rows)
	if rows != 0 {
		t.Errorf("expected no questionnaire row after a rejected submission, found %d", rows)
	}
}

func TestE2EPremiumPlanRequiresAPurchase(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)
	client := f.login(t, uniqueEmail())

	// Submitting an intake must not grant a plan: only a purchase does.
	if response := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": validIntakeBody()}); response.Code != http.StatusOK {
		t.Fatalf("submit = %d; body %s", response.Code, response.Body.String())
	}

	status := client.do(t, http.MethodGet, sprintfPath(premiumNutritionRoute, program.ID), nil)
	if status.Code != http.StatusNotFound {
		t.Fatalf("nutrition before purchase = %d, want 404; body %s", status.Code, status.Body.String())
	}

	generated := client.do(t, http.MethodPost, sprintfPath(premiumGenerateRoute, program.ID), nil)
	if generated.Code != http.StatusNotFound {
		t.Fatalf("generate before purchase = %d, want 404; body %s", generated.Code, generated.Body.String())
	}
}

func TestE2EPremiumPlanIsPrivateToTheBuyer(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)
	buyer := f.login(t, uniqueEmail())
	intruder := f.login(t, uniqueEmail())

	// The intruder submits their own intake for the same program.
	if response := intruder.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": validIntakeBody()}); response.Code != http.StatusOK {
		t.Fatalf("intruder submit = %d; body %s", response.Code, response.Body.String())
	}

	// The buyer completes the purchase and their plan generation.
	completePremiumPurchase(t, f, buyer, program)

	// The intruder must not be able to read or generate a plan they never bought.
	read := intruder.do(t, http.MethodGet, sprintfPath(premiumNutritionRoute, program.ID), nil)
	if read.Code != http.StatusNotFound {
		t.Errorf("cross-account read = %d, want 404; body %s", read.Code, read.Body.String())
	}
	generate := intruder.do(t, http.MethodPost, sprintfPath(premiumGenerateRoute, program.ID), nil)
	if generate.Code != http.StatusNotFound {
		t.Errorf("cross-account generate = %d, want 404; body %s", generate.Code, generate.Body.String())
	}

	// The buyer's own plan remains readable.
	own := buyer.do(t, http.MethodGet, sprintfPath(premiumNutritionRoute, program.ID), nil)
	if own.Code != http.StatusOK {
		t.Fatalf("buyer read = %d, want 200; body %s", own.Code, own.Body.String())
	}
}

func TestE2EPremiumPlanIsGeneratedAfterPurchaseAndIsIdempotent(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)
	client := f.login(t, uniqueEmail())

	completePremiumPurchase(t, f, client, program)

	// Immediately after checkout the plan is owed but not yet produced.
	pending := client.do(t, http.MethodGet, sprintfPath(premiumNutritionRoute, program.ID), nil)
	if pending.Code != http.StatusOK {
		t.Fatalf("status after purchase = %d, want 200; body %s", pending.Code, pending.Body.String())
	}
	if status := decodeBody(t, pending.Body.String())["status"]; status != models.NutritionAssignmentStatusPending {
		t.Errorf("status after purchase = %v, want pending", status)
	}

	generated := client.do(t, http.MethodPost, sprintfPath(premiumGenerateRoute, program.ID), nil)
	if generated.Code != http.StatusOK {
		t.Fatalf("generate = %d, want 200; body %s", generated.Code, generated.Body.String())
	}
	payload := decodeBody(t, generated.Body.String())
	if payload["status"] != models.NutritionAssignmentStatusCompleted {
		t.Errorf("status = %v, want completed", payload["status"])
	}

	plan, ok := payload["plan"].(map[string]any)
	if !ok {
		t.Fatalf("expected a plan in the response, got %v", payload["plan"])
	}
	energy, ok := plan["energy_targets"].(map[string]any)
	if !ok {
		t.Fatalf("expected energy targets in the plan, got %v", plan["energy_targets"])
	}
	if energy["target_calories"].(float64) <= 0 {
		t.Errorf("expected a positive calorie target, got %v", energy["target_calories"])
	}

	// A repeated generation must converge on the same single plan.
	again := client.do(t, http.MethodPost, sprintfPath(premiumGenerateRoute, program.ID), nil)
	if again.Code != http.StatusOK {
		t.Fatalf("repeat generate = %d, want 200; body %s", again.Code, again.Body.String())
	}
	if againPlan := decodeBody(t, again.Body.String())["plan"].(map[string]any); againPlan["fingerprint"] != plan["fingerprint"] {
		t.Error("expected a repeated generation to return the same plan")
	}

	var assignmentRows int64
	f.tx.Model(&models.NutritionAssignment{}).Where("program_id = ?", program.ID).Count(&assignmentRows)
	if assignmentRows != 1 {
		t.Errorf("expected exactly one nutrition assignment, found %d", assignmentRows)
	}
}

func TestE2EPremiumQuestionnaireLocksAfterPurchase(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)
	client := f.login(t, uniqueEmail())

	// Before purchase the intake is freely editable, including resubmission.
	first := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": validIntakeBody()})
	if first.Code != http.StatusOK {
		t.Fatalf("initial submit = %d; body %s", first.Code, first.Body.String())
	}
	updated := validIntakeBody()
	updated["goal"] = "muscle_gain"
	if response := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": updated}); response.Code != http.StatusOK {
		t.Fatalf("resubmit before purchase = %d, want 200; body %s", response.Code, response.Body.String())
	}

	// The requirement view advertises the locked state before the purchase.
	preLock := decodeBody(t, client.do(t, http.MethodGet, sprintfPath(premiumQuestionnaireRoute, program.ID), nil).Body.String())
	if preLock["locked"] != false {
		t.Fatalf("locked before purchase = %v, want false", preLock["locked"])
	}

	completePremiumPurchase(t, f, client, program)

	// The server, not the client, decides editability.
	postLock := decodeBody(t, client.do(t, http.MethodGet, sprintfPath(premiumQuestionnaireRoute, program.ID), nil).Body.String())
	if postLock["locked"] != true {
		t.Fatalf("locked after purchase = %v, want true", postLock["locked"])
	}
	if postLock["submitted"] != true {
		t.Fatal("reading the requirement after locking must still report the stored intake")
	}

	// Every mutation attempt is refused with the documented conflict.
	changed := validIntakeBody()
	changed["goal"] = "fat_loss"
	refused := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": changed})
	if refused.Code != http.StatusConflict {
		t.Fatalf("mutate after purchase = %d, want 409; body %s", refused.Code, refused.Body.String())
	}
	if code := errorCode(t, decodeBody(t, refused.Body.String())); code != "QUESTIONNAIRE_LOCKED" {
		t.Fatalf("conflict code = %q, want QUESTIONNAIRE_LOCKED", code)
	}

	// The stored revision is untouched: a refused mutation changes nothing.
	after := decodeBody(t, client.do(t, http.MethodGet, sprintfPath(premiumQuestionnaireRoute, program.ID), nil).Body.String())
	if after["version"] != postLock["version"] {
		t.Fatalf("a refused mutation must not bump the revision: before %v, after %v", postLock["version"], after["version"])
	}
}

// The lock must survive nutrition generation: a delivered package stays
// immutable on both the questionnaire and the plan side.
// Locking is decided per authenticated owner by their own entitlement, so a
// stored intake is never a shared program-level record. This test pins both
// halves of that property: the owner is locked by their purchase, and a second
// account observes only its own revision.
func TestE2EPremiumLockedQuestionnaireStaysPrivateToItsOwner(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)
	owner := f.login(t, uniqueEmail())
	intruder := f.login(t, uniqueEmail())

	// The intruder submits their own intake for the same program, so a refusal
	// can never be mistaken for "nothing stored".
	intruderBody := validIntakeBody()
	intruderBody["goal"] = "fat_loss"
	if response := intruder.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": intruderBody}); response.Code != http.StatusOK {
		t.Fatalf("intruder submit = %d; body %s", response.Code, response.Body.String())
	}
	intruderState := decodeBody(t, intruder.do(t, http.MethodGet, sprintfPath(premiumQuestionnaireRoute, program.ID), nil).Body.String())
	if intruderState["submitted"] != true {
		t.Fatal("the intruder must see their own submitted intake")
	}

	// The owner submits and buys, which locks the owner's intake only.
	if response := owner.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": validIntakeBody()}); response.Code != http.StatusOK {
		t.Fatalf("owner submit = %d; body %s", response.Code, response.Body.String())
	}
	completePremiumPurchase(t, f, owner, program)

	ownerState := decodeBody(t, owner.do(t, http.MethodGet, sprintfPath(premiumQuestionnaireRoute, program.ID), nil).Body.String())
	if ownerState["locked"] != true {
		t.Fatalf("owner locked after purchase = %v, want true", ownerState["locked"])
	}

	// Re-reading as the intruder returns the intruder's own state, not the
	// owner's locked revision.
	afterOwnerPurchase := decodeBody(t, intruder.do(t, http.MethodGet, sprintfPath(premiumQuestionnaireRoute, program.ID), nil).Body.String())
	if afterOwnerPurchase["version"] != intruderState["version"] {
		t.Fatalf("a second account observed a foreign revision: before %v, after %v", intruderState["version"], afterOwnerPurchase["version"])
	}
	// The intruder's own intake is not locked, because they never purchased.
	if afterOwnerPurchase["locked"] != false {
		t.Fatalf("intruder locked = %v, want false: the lock follows their own entitlement", afterOwnerPurchase["locked"])
	}

	// The owner is still refused, so the private record is genuinely immutable.
	changed := validIntakeBody()
	changed["goal"] = "muscle_gain"
	refused := owner.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": changed})
	if refused.Code != http.StatusConflict {
		t.Fatalf("owner mutate after purchase = %d, want 409; body %s", refused.Code, refused.Body.String())
	}
	if code := errorCode(t, decodeBody(t, refused.Body.String())); code != "QUESTIONNAIRE_LOCKED" {
		t.Fatalf("owner conflict code = %q, want QUESTIONNAIRE_LOCKED", code)
	}
}

func TestE2EPremiumQuestionnaireLocksAfterNutritionGeneration(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)
	client := f.login(t, uniqueEmail())

	completePremiumPurchase(t, f, client, program)
	if response := client.do(t, http.MethodPost, sprintfPath(premiumGenerateRoute, program.ID), nil); response.Code != http.StatusOK {
		t.Fatalf("generate = %d; body %s", response.Code, response.Body.String())
	}

	changed := validIntakeBody()
	changed["goal"] = "fat_loss"
	if refused := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": changed}); refused.Code != http.StatusConflict {
		t.Fatalf("mutate after generation = %d, want 409; body %s", refused.Code, refused.Body.String())
	}

	// Regenerating must return the delivered plan unchanged rather than
	// producing a second, different one.
	status := client.do(t, http.MethodGet, sprintfPath(premiumNutritionRoute, program.ID), nil)
	if status.Code != http.StatusOK {
		t.Fatalf("status after generation = %d; body %s", status.Code, status.Body.String())
	}
	delivered := decodeBody(t, status.Body.String())
	if delivered["status"] != models.NutritionAssignmentStatusCompleted {
		t.Fatalf("status = %v, want completed", delivered["status"])
	}
	if delivered["out_of_date"] == true {
		t.Fatal("a locked questionnaire can never make the delivered plan out of date")
	}
	repeat := client.do(t, http.MethodPost, sprintfPath(premiumGenerateRoute, program.ID), nil)
	if repeat.Code != http.StatusOK {
		t.Fatalf("repeat generate = %d; body %s", repeat.Code, repeat.Body.String())
	}
	if decodeBody(t, repeat.Body.String())["status"] != models.NutritionAssignmentStatusCompleted {
		t.Fatal("regeneration must not produce a second plan state")
	}
}

// A Test Mode purchase creates the same entitlement, so it must lock the
// questionnaire exactly like a paid purchase.
func TestE2EPremiumQuestionnaireLocksAfterTestModePurchase(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)

	cookies, _ := enterTestMode(t, f.router, f.tokens, "client")

	// The questionnaire gate applies to Test Mode too.
	cookies = testModePersona(t, f, cookies)
	blocked := testModePurchase(t, f, cookies, program.ID)
	if blocked.Code == http.StatusOK || blocked.Code == http.StatusCreated {
		t.Fatalf("Test Mode purchase without an intake = %d, want a rejection; body %s", blocked.Code, blocked.Body.String())
	}
	if code := errorCode(t, decodeBody(t, blocked.Body.String())); code != "PURCHASE_PREREQUISITE_NOT_MET" {
		t.Fatalf("Test Mode gate code = %q, want PURCHASE_PREREQUISITE_NOT_MET", code)
	}

	// Submit the persona intake through the same questionnaire endpoint.
	payload := testModeRequestWith(t, f, cookies, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), `{"answers":`+mustJSON(t, validIntakeBody())+`}`)
	if payload.Code != http.StatusOK {
		t.Fatalf("persona intake = %d; body %s", payload.Code, payload.Body.String())
	}

	completed := testModePurchase(t, f, cookies, program.ID)
	if completed.Code != http.StatusCreated {
		t.Fatalf("Test Mode purchase = %d, want 201; body %s", completed.Code, completed.Body.String())
	}
	if status := purchaseData(t, completed)["status"]; status != models.PurchaseStatusCompleted {
		t.Fatalf("Test Mode purchase status = %v, want completed", status)
	}

	// The lock is identical after a Test Mode purchase.
	changed := validIntakeBody()
	changed["goal"] = "fat_loss"
	refused := testModeRequestWith(t, f, cookies, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), `{"answers":`+mustJSON(t, changed)+`}`)
	if refused.Code != http.StatusConflict {
		t.Fatalf("mutate after Test Mode purchase = %d, want 409; body %s", refused.Code, refused.Body.String())
	}
}

func TestE2EGenericProgramIsUnaffectedByTheIntakeGate(t *testing.T) {
	// The gate must be invisible for products outside the Premium Level 1
	// family, so existing checkout behaviour is unchanged.
	f := newPremiumFlowRouter(t)
	program := f.seedGenericProgram(t)
	client := f.login(t, uniqueEmail())

	// A generic program needs no intake.
	response := client.do(t, http.MethodPost, "/api/v1/me/programs/"+program.ID+"/purchase", nil)
	if response.Code != http.StatusCreated {
		t.Fatalf("generic purchase without an intake = %d, want 201; body %s", response.Code, response.Body.String())
	}

	// And it has no nutrition plan.
	status := client.do(t, http.MethodGet, sprintfPath(premiumNutritionRoute, program.ID), nil)
	if status.Code != http.StatusNotFound {
		t.Errorf("generic nutrition = %d, want 404; body %s", status.Code, status.Body.String())
	}
	// Submitting an intake for a generic program is refused.
	submit := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": validIntakeBody()})
	if submit.Code != http.StatusNotFound {
		t.Errorf("generic questionnaire = %d, want 404; body %s", submit.Code, submit.Body.String())
	}
}

func TestE2EPremiumEndpointsRequireAuthentication(t *testing.T) {
	f := newPremiumFlowRouter(t)
	program := f.seedPremiumProgram(t)

	anonymous := &session{router: f.router}
	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, premiumQuestionsRoute},
		{http.MethodGet, sprintfPath(premiumQuestionnaireRoute, program.ID)},
		{http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID)},
		{http.MethodGet, sprintfPath(premiumNutritionRoute, program.ID)},
		{http.MethodPost, sprintfPath(premiumGenerateRoute, program.ID)},
	} {
		recorder := anonymous.do(t, route.method, route.path, nil)
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a session = %d, want 401", route.method, route.path, recorder.Code)
		}
	}
}

func TestE2EQuestionCatalogDescribesTheContract(t *testing.T) {
	f := newPremiumFlowRouter(t)
	client := f.login(t, uniqueEmail())

	response := client.do(t, http.MethodGet, premiumQuestionsRoute, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("questions = %d, want 200; body %s", response.Code, response.Body.String())
	}

	payload := decodeBody(t, response.Body.String())
	questions, ok := payload["questions"].([]any)
	if !ok || len(questions) == 0 {
		t.Fatalf("expected a non-empty question catalog, got %v", payload["questions"])
	}
	if payload["version"].(float64) <= 0 {
		t.Error("expected the catalog to report its contract version")
	}
}

// completePremiumPurchase submits the intake and drives a paid purchase to a
// completed, entitlement-backed state through the fake provider.
func completePremiumPurchase(t *testing.T, f *premiumFlowFixture, client *session, program *models.Program) {
	t.Helper()

	if response := client.do(t, http.MethodPost, sprintfPath(premiumQuestionnaireRoute, program.ID), map[string]any{"answers": validIntakeBody()}); response.Code != http.StatusOK {
		t.Fatalf("submit questionnaire = %d; body %s", response.Code, response.Body.String())
	}

	created := client.do(t, http.MethodPost, "/api/v1/me/programs/"+program.ID+"/purchase", nil)
	if created.Code != http.StatusCreated {
		t.Fatalf("create purchase = %d, want 201; body %s", created.Code, created.Body.String())
	}
	purchaseID := purchaseData(t, created)["id"].(string)

	initiated := client.do(t, http.MethodPost, "/api/v1/me/purchases/"+purchaseID+"/payment", map[string]any{"payment_method": "card"})
	if initiated.Code != http.StatusOK {
		t.Fatalf("initiate payment = %d; body %s", initiated.Code, initiated.Body.String())
	}

	captured := client.do(t, http.MethodPost, "/api/v1/me/purchases/"+purchaseID+"/capture", map[string]any{"provider_payment_id": "fake-session"})
	if captured.Code != http.StatusOK {
		t.Fatalf("capture payment = %d; body %s", captured.Code, captured.Body.String())
	}
	if status := purchaseData(t, captured)["status"]; status != models.PurchaseStatusCompleted {
		t.Fatalf("purchase status = %v, want completed; body %s", status, captured.Body.String())
	}
}

// purchaseData unwraps the `data` envelope a purchase response uses.
func purchaseData(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	payload := decodeBody(t, recorder.Body.String())
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected a data envelope in %q", recorder.Body.String())
	}
	return data
}

// errorCode extracts the error code from a failure response body.
func errorCode(t *testing.T, payload map[string]any) string {
	t.Helper()

	failure, ok := payload["error"].(map[string]any)
	if !ok {
		return ""
	}
	code, _ := failure["code"].(string)
	return code
}

// containsString reports whether haystack contains needle.
func containsString(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// joinStrings concatenates values for a readable assertion message.
func joinStrings(values []string, separator string) string {
	return strings.Join(values, separator)
}

// strPtr is a small helper for building seeded program metadata.
func strPtr(v string) *string { return &v }

// testModePersona returns the cookie set of the active Test Mode persona, so a
// request can be issued as the persona the administrator switched into.
func testModePersona(t *testing.T, f *premiumFlowFixture, cookies map[string]string) map[string]string {
	t.Helper()

	if cookies[auth.AccessTokenCookieName] == "" {
		t.Fatal("Test Mode must hand back a persona access token")
	}
	return cookies
}

// testModePurchase issues a Test Mode purchase as the active persona.
func testModePurchase(t *testing.T, f *premiumFlowFixture, cookies map[string]string, programID string) *httptest.ResponseRecorder {
	t.Helper()

	return testModeRequestWith(t, f, cookies, http.MethodPost, "/api/v1/auth/test-mode/programs/"+programID+"/purchase", "")
}

// testModeRequestWith issues an authenticated request carrying both the persona
// access token and the Test Mode session token.
func testModeRequestWith(t *testing.T, f *premiumFlowFixture, cookies map[string]string, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: cookies[auth.AccessTokenCookieName]})
	req.AddCookie(&http.Cookie{Name: auth.TestSessionTokenCookieName, Value: cookies[auth.TestSessionTokenCookieName]})

	f.router.ServeHTTP(recorder, req)
	return recorder
}

// mustJSON encodes a request payload that is statically known to be valid.
func mustJSON(t *testing.T, value map[string]any) string {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	return string(encoded)
}
