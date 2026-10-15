package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware"
	"ryze/backend/middleware/adminroles"
	"ryze/backend/models"
	"ryze/backend/services/generic_programs"
	"ryze/backend/services/token"
)

var errGenericRepoFailure = errors.New("repository failure")

const (
	genericProgramsRoute        = "/api/v1/programs/generic"
	genericProgramsRouteProgram = "/api/v1/programs/generic/"
	premiumLevel1ProgramsRoute  = "/api/v1/programs/premium-level-1"
)

// newGenericProgramsTestRouter wires the platform-owned program endpoints behind
// the real AdminAuthenticate and RequireAdminPermission middleware, mounting
// both the generic and the premium-level-1 family groups on the same scripted
// service so the family binding, security and mapping paths can be exercised
// without a database.
func newGenericProgramsTestRouter(t *testing.T, svc generic_programs.Service) (*gin.Engine, token.Service) {
	t.Helper()

	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	genericHandler := auth.NewGenericProgramsHandler(svc, models.ProgramProductTypeGeneric)
	premiumHandler := auth.NewGenericProgramsHandler(svc, models.ProgramProductTypePremiumLevel1)

	router := gin.New()
	mount := func(route string, handler *auth.GenericProgramsHandler) {
		group := router.Group(route)
		group.Use(middleware.AdminAuthenticate(tokenSvc))
		group.Use(middleware.RequireAdminPermission(adminroles.PermissionPlans))
		group.GET("", handler.ListPrograms)
		group.POST("", handler.CreateProgram)
		group.GET("/:programID", handler.GetProgram)
		group.PATCH("/:programID", handler.UpdateProgram)
		group.POST("/:programID/publish", handler.PublishProgram)
		group.DELETE("/:programID", handler.DeleteProgram)
	}
	mount(genericProgramsRoute, genericHandler)
	mount(premiumLevel1ProgramsRoute, premiumHandler)

	return router, tokenSvc
}

func genericAdminCookie(t *testing.T, tokenSvc token.Service, adminID string) string {
	t.Helper()
	tokenSvc.RegisterAdminCredentials(adminID, "test-admin-password", "test-admin-access-code")
	value, err := tokenSvc.GenerateAdminToken(adminID)
	if err != nil {
		t.Fatalf("GenerateAdminToken: %v", err)
	}
	return value
}

func genericProgramsRequest(router http.Handler, cookieValue, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: auth.AdminAccessTokenCookieName, Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// stubGenericProgramsService is a scripted fake of the generic programs service
// used to exercise the handler's security wiring, identity forwarding and error
// mapping without touching the database.
type stubGenericProgramsService struct {
	detail         *generic_programs.ProgramDetail
	summary        generic_programs.Program
	listResult     generic_programs.ListProgramsResult
	err            error
	gotProgramID   string
	gotPage        int
	gotLimit       int
	gotProductType string
	gotCreateInput generic_programs.ProgramInput
	gotUpdateInput generic_programs.ProgramInput
}

func (s *stubGenericProgramsService) CreateProgram(_ context.Context, productType string, input generic_programs.ProgramInput) (*generic_programs.ProgramDetail, error) {
	s.gotProductType = productType
	s.gotCreateInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.detail, nil
}

func (s *stubGenericProgramsService) ListPrograms(_ context.Context, productType string, _ generic_programs.ProgramFilter, page, limit int) (generic_programs.ListProgramsResult, error) {
	s.gotProductType = productType
	s.gotPage = page
	s.gotLimit = limit
	if s.err != nil {
		return generic_programs.ListProgramsResult{}, s.err
	}
	return s.listResult, nil
}

func (s *stubGenericProgramsService) GetProgram(_ context.Context, productType, programID string) (*generic_programs.ProgramDetail, error) {
	s.gotProductType = productType
	s.gotProgramID = programID
	if s.err != nil {
		return nil, s.err
	}
	return s.detail, nil
}

func (s *stubGenericProgramsService) UpdateProgram(_ context.Context, productType, programID string, input generic_programs.ProgramInput) (*generic_programs.ProgramDetail, error) {
	s.gotProductType = productType
	s.gotProgramID = programID
	s.gotUpdateInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.detail, nil
}

func (s *stubGenericProgramsService) PublishProgram(_ context.Context, productType, programID string) (generic_programs.Program, error) {
	s.gotProductType = productType
	s.gotProgramID = programID
	if s.err != nil {
		return generic_programs.Program{}, s.err
	}
	return s.summary, nil
}

func (s *stubGenericProgramsService) DeleteProgram(_ context.Context, productType, programID string) error {
	s.gotProductType = productType
	s.gotProgramID = programID
	return s.err
}

func validGenericDetail() *generic_programs.ProgramDetail {
	level := "Intermediate"
	duration := 12
	frequency := 4
	return &generic_programs.ProgramDetail{
		Program: generic_programs.Program{
			ID:               "22222222-2222-2222-2222-222222222222",
			Name:             "Hypertrophy 101",
			Description:      "A 12 week hypertrophy program.",
			Type:             "premium",
			Status:           "draft",
			ProductType:      models.ProgramProductTypeGeneric,
			Level:            &level,
			DurationWeeks:    &duration,
			FrequencyPerWeek: &frequency,
			PriceMinorUnits:  1449,
			Currency:         "EUR",
			CreatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Weeks: []generic_programs.Week{
			{
				WeekNumber: 1,
				Workouts: []generic_programs.Workout{
					{
						Position: 1,
						Exercises: []generic_programs.WorkoutExercise{
							{
								Position:     1,
								Instructions: "Keep your back straight.",
								Notes:        "Focus on control.",
								Exercise: generic_programs.Exercise{
									ID:          "44444444-4444-4444-4444-444444444444",
									Name:        "Barbell Row",
									Description: "A horizontal pull.",
								},
								Sets: []generic_programs.Set{
									{
										SetNumber: 1,
										SetType:   "working",
										Reps:      intPtr(10),
										WeightKg:  floatPtr(100),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

func validGenericProgramBody() string {
	return `{
		"name": "Hypertrophy 101",
		"description": "A 12 week hypertrophy program.",
		"type": "premium",
		"status": "draft",
		"level": "Intermediate",
		"duration_weeks": 12,
		"frequency_per_week": 4,
		"price_minor_units": 1449,
		"currency": "EUR",
		"weeks": [
			{
				"workouts": [
					{
						"exercises": [
							{
								"exercise_id": "44444444-4444-4444-4444-444444444444",
								"instructions": "Keep your back straight.",
								"notes": "Focus on control.",
								"sets": [
									{
										"set_type": "working",
										"reps": 10,
										"weight_kg": 100,
										"rir": 1,
										"rpe": 8,
										"rest_seconds": 90,
										"tempo": "2010"
									}
								]
							}
						]
					}
				]
			}
		]
	}`
}

func TestGenericProgramsRequireAuthentication(t *testing.T) {
	router, _ := newGenericProgramsTestRouter(t, &stubGenericProgramsService{})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: genericProgramsRoute},
		{method: http.MethodPost, path: genericProgramsRoute},
	} {
		rec := genericProgramsRequest(router, "", tc.method, tc.path, "")
		body := decodeBody(t, rec.Body.String())
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d", tc.method, tc.path, rec.Code)
		}
		if body["error"].(map[string]any)["code"] != "AUTHENTICATION_REQUIRED" {
			t.Fatalf("expected AUTHENTICATION_REQUIRED, got %v", body["error"])
		}
	}
}

func TestGenericProgramsRejectUserToken(t *testing.T) {
	router, tokenSvc := newGenericProgramsTestRouter(t, &stubGenericProgramsService{})

	userToken := userToken(t, tokenSvc, "99999999-9999-9999-9999-999999999999", 1)
	rec := genericProgramsRequest(router, userToken, http.MethodGet, genericProgramsRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a user token, got %d", rec.Code)
	}
}

func TestGenericProgramsRequirePermission(t *testing.T) {
	router, tokenSvc := newGenericProgramsTestRouter(t, &stubGenericProgramsService{})

	// ADMIN_1 (Technical Administrator) holds no plans permission.
	cookie := genericAdminCookie(t, tokenSvc, "ADMIN_1")
	rec := genericProgramsRequest(router, cookie, http.MethodGet, genericProgramsRoute, "")
	body := decodeBody(t, rec.Body.String())
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for ADMIN_1, got %d", rec.Code)
	}
	if body["error"].(map[string]any)["code"] != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %v", body["error"])
	}
}

func TestGenericProgramsCreateAdminOnly(t *testing.T) {
	svc := &stubGenericProgramsService{detail: validGenericDetail()}
	router, tokenSvc := newGenericProgramsTestRouter(t, svc)

	cookie := genericAdminCookie(t, tokenSvc, "ADMIN_2")
	rec := genericProgramsRequest(router, cookie, http.MethodPost, genericProgramsRoute, validGenericProgramBody())
	body := decodeBody(t, rec.Body.String())
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if svc.gotCreateInput.Name != "Hypertrophy 101" {
		t.Fatalf("expected name forwarded, got %q", svc.gotCreateInput.Name)
	}
	data := body["data"].(map[string]any)
	if data["id"] != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("expected program id in body, got %v", data["id"])
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("trainer_id")) {
		t.Fatal("generic programs must never expose a trainer id")
	}
	weeks := data["weeks"].([]any)
	workouts := weeks[0].(map[string]any)["workouts"].([]any)
	exercises := workouts[0].(map[string]any)["exercises"].([]any)
	exercise := exercises[0].(map[string]any)
	if exercise["name"] != "Barbell Row" {
		t.Fatalf("expected exercise metadata, got %v", exercise)
	}
	if _, present := exercise["sets"]; !present {
		t.Fatal("expected sets in exercise response")
	}
}

func TestGenericProgramsErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid input", err: generic_programs.ErrInvalidInput, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "program not found", err: generic_programs.ErrProgramNotFound, wantStatus: http.StatusNotFound, wantCode: "PROGRAM_NOT_FOUND"},
		{name: "exercise not found", err: generic_programs.ErrExerciseNotFound, wantStatus: http.StatusUnprocessableEntity, wantCode: "EXERCISE_NOT_FOUND"},
		{name: "duplicate exercise", err: generic_programs.ErrDuplicateExercise, wantStatus: http.StatusConflict, wantCode: "DUPLICATE_EXERCISE"},
		{name: "already published", err: generic_programs.ErrProgramAlreadyPublished, wantStatus: http.StatusConflict, wantCode: "PROGRAM_ALREADY_PUBLISHED"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubGenericProgramsService{err: tc.err}
			router, tokenSvc := newGenericProgramsTestRouter(t, svc)

			cookie := genericAdminCookie(t, tokenSvc, "ADMIN_2")
			rec := genericProgramsRequest(router, cookie, http.MethodGet, genericProgramsRoute, "")
			body := decodeBody(t, rec.Body.String())
			code := body["error"].(map[string]any)["code"]
			if rec.Code != tc.wantStatus || code != tc.wantCode {
				t.Fatalf("expected %d %s, got %d %v", tc.wantStatus, tc.wantCode, rec.Code, code)
			}
		})
	}
}

func TestGenericProgramsInternalErrorNotExposed(t *testing.T) {
	svc := &stubGenericProgramsService{err: errGenericRepoFailure}
	router, tokenSvc := newGenericProgramsTestRouter(t, svc)

	cookie := genericAdminCookie(t, tokenSvc, "ADMIN_2")
	rec := genericProgramsRequest(router, cookie, http.MethodGet, genericProgramsRoute, "")
	body := decodeBody(t, rec.Body.String())
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if body["error"].(map[string]any)["code"] != "INTERNAL_ERROR" {
		t.Fatalf("expected INTERNAL_ERROR, got %v", body["error"])
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("repository failure")) {
		t.Fatal("internal details must never be exposed")
	}
}

// decodeBody parses the JSON response body.
func decodeBody(t *testing.T, raw string) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("decode body: %v (raw: %s)", err, raw)
	}
	return body
}

func TestGenericProgramsQueryParsing(t *testing.T) {
	svc := &stubGenericProgramsService{
		listResult: generic_programs.ListProgramsResult{
			Programs: []generic_programs.Program{validGenericDetail().Program},
			Total:    1,
			Page:     2,
			Limit:    10,
		},
	}
	router, tokenSvc := newGenericProgramsTestRouter(t, svc)

	cookie := genericAdminCookie(t, tokenSvc, "ADMIN_2")
	path := genericProgramsRoute + "?page=2&limit=10&search=hypertrophy&type=premium&level=Intermediate&duration=12&frequency=4"
	rec := genericProgramsRequest(router, cookie, http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if svc.gotPage != 2 || svc.gotLimit != 10 {
		t.Fatalf("expected page 2 limit 10 forwarded, got %d/%d", svc.gotPage, svc.gotLimit)
	}
	data := decodeBody(t, rec.Body.String())["data"].(map[string]any)
	pagination := data["pagination"].(map[string]any)
	if pagination["total_pages"] != float64(1) {
		t.Fatalf("expected 1 total page, got %v", pagination["total_pages"])
	}
}

// TestGenericProgramsFamilyBinding verifies that the handler derives the product
// family exclusively from the route group it is mounted on and never from the
// request. A body that tries to smuggle the opposite family is ignored because
// the request schema has no product_type field; the premium route must always
// operate on the premium_level_1 family and the generic route on generic.
func TestGenericProgramsFamilyBinding(t *testing.T) {
	tests := []struct {
		name       string
		route      string
		wantFamily string
	}{
		{name: "generic route binds generic family", route: genericProgramsRoute, wantFamily: models.ProgramProductTypeGeneric},
		{name: "premium level 1 route binds premium family", route: premiumLevel1ProgramsRoute, wantFamily: models.ProgramProductTypePremiumLevel1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opposite := models.ProgramProductTypeGeneric
			if tc.wantFamily == opposite {
				opposite = models.ProgramProductTypePremiumLevel1
			}

			detail := validGenericDetail()
			detail.ProductType = tc.wantFamily
			svc := &stubGenericProgramsService{detail: detail}
			router, tokenSvc := newGenericProgramsTestRouter(t, svc)

			cookie := genericAdminCookie(t, tokenSvc, "ADMIN_2")
			body := strings.Replace(validGenericProgramBody(),
				`"currency": "EUR",`,
				`"currency": "EUR",
		"product_type": "`+opposite+`",`, 1)
			rec := genericProgramsRequest(router, cookie, http.MethodPost, tc.route, body)
			if rec.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
			}
			if svc.gotProductType != tc.wantFamily {
				t.Fatalf("family forwarded to service = %q, want %q", svc.gotProductType, tc.wantFamily)
			}
			data := decodeBody(t, rec.Body.String())["data"].(map[string]any)
			if data["product_type"] != tc.wantFamily {
				t.Fatalf("response product_type = %v, want %q", data["product_type"], tc.wantFamily)
			}
		})
	}
}
