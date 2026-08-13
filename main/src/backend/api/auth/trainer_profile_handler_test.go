package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/middleware"
	"ryze/backend/middleware/trainercontext"
	"ryze/backend/middleware/trainerroles"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/token"
	"ryze/backend/services/trainer_profile"
)

const profileRoute = "/api/v1/trainer/profile"

// newTrainerProfileTestRouter wires the trainer profile endpoint behind the
// real Authenticate, TrainerAuthenticate and RequireTrainerPermission
// middleware, backed by a database transaction so created records are rolled
// back. The required permissions can be customized to exercise the 403 path.
func newTrainerProfileTestRouter(t *testing.T, permissions ...trainerroles.Permission) (*gin.Engine, repositories.UserRepository, repositories.TrainerRepository, token.Service) {
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
	t.Cleanup(func() { tx.Rollback() })

	userRepo := repositories.NewUserRepository(tx)
	trainerRepo := repositories.NewTrainerRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	service := trainer_profile.NewService(trainerRepo)
	handler := auth.NewTrainerProfileHandler(service)

	router := gin.New()
	trainer := router.Group(profileRoute)
	trainer.Use(middleware.Authenticate(tokenSvc, userRepo))
	trainer.Use(middleware.TrainerAuthenticate(trainerRepo))
	trainer.GET("", middleware.RequireTrainerPermission(permissions...), handler.GetProfile)

	return router, userRepo, trainerRepo, tokenSvc
}

// newTrainerProfileHandlerRouter mounts only the handler with a pre-set
// trainer context identity, so the handler's own error mapping can be tested
// without the full middleware chain. nil identity simulates a missing context,
// any other value simulates a malformed identity.
func newTrainerProfileHandlerRouter(svc trainer_profile.Service, identity any) *gin.Engine {
	handler := auth.NewTrainerProfileHandler(svc)
	router := gin.New()
	router.GET(profileRoute, func(c *gin.Context) {
		if identity != nil {
			c.Set(trainercontext.TrainerContextKey, identity)
		}
		handler.GetProfile(c)
	})
	return router
}

// seedTrainerForUser creates a trainer record owned by the given user, so the
// trainer can be reached through the TrainerAuthenticate middleware when the
// token belongs to that same user.
func seedTrainerForUser(t *testing.T, trainerRepo repositories.TrainerRepository, user *models.User) *models.Trainer {
	t.Helper()

	trainer := &models.Trainer{UserID: user.ID}
	if err := trainerRepo.Create(context.Background(), trainer); err != nil {
		t.Fatalf("seed trainer: %v", err)
	}
	return trainer
}

// stubTrainerProfileService is a scripted fake used to exercise the handler's
// error mapping without touching the database.
type stubTrainerProfileService struct {
	profile *trainer_profile.Profile
	err     error
}

func (s stubTrainerProfileService) GetProfile(_ context.Context, _, _ string) (*trainer_profile.Profile, error) {
	return s.profile, s.err
}

func trainerProfileRequest(router http.Handler, cookieValue, query string) (*httptest.ResponseRecorder, map[string]any, string) {
	req := httptest.NewRequest(http.MethodGet, profileRoute+query, nil)
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: cookieValue})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		return rec, nil, ""
	}
	data, _ := payload["data"].(map[string]any)
	raw, _ := json.Marshal(payload)
	return rec, data, string(raw)
}

func trainerProfilePayload(raw string) map[string]any {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}
	return payload
}

func TestTrainerProfileSuccess(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newTrainerProfileTestRouter(t, trainerroles.PermissionProfile)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, user)

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := trainerProfileRequest(router, jwtValue, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if data == nil {
		t.Fatal("expected data object in response")
	}

	if id, _ := data["id"].(string); id != trainer.ID {
		t.Fatalf("expected trainer id %q, got %q", trainer.ID, id)
	}
	if userID, _ := data["user_id"].(string); userID != user.ID {
		t.Fatalf("expected user id %q, got %q", user.ID, userID)
	}

	nested, ok := data["user"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested user object, got %#v", data["user"])
	}
	if id, _ := nested["id"].(string); id != user.ID {
		t.Fatalf("expected nested user id %q, got %q", user.ID, id)
	}
	if email, _ := nested["email"].(string); email != user.Email {
		t.Fatalf("expected email %q, got %q", user.Email, email)
	}
	if firstName, _ := nested["first_name"].(string); firstName != user.FirstName {
		t.Fatalf("expected first_name %q, got %q", user.FirstName, firstName)
	}
	if lastName, _ := nested["last_name"].(string); lastName != user.LastName {
		t.Fatalf("expected last_name %q, got %q", user.LastName, lastName)
	}
	for _, field := range []string{"created_at", "updated_at"} {
		if value, _ := data[field].(string); value == "" {
			t.Fatalf("expected non-empty trainer %s, got %q", field, value)
		}
		if value, _ := nested[field].(string); value == "" {
			t.Fatalf("expected non-empty user %s, got %q", field, value)
		}
	}

	if success, _ := trainerProfilePayload(raw)["success"].(bool); !success {
		t.Fatalf("expected success:true, got %s", raw)
	}
	if !strings.Contains(raw, `"message":"Trainer profile retrieved successfully."`) {
		t.Fatalf("expected success message, got %s", raw)
	}
}

func TestTrainerProfileNeverExposesSecrets(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newTrainerProfileTestRouter(t, trainerroles.PermissionProfile)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	seedTrainerForUser(t, trainerRepo, user)

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, data, raw := trainerProfileRequest(router, jwtValue, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	nested, _ := data["user"].(map[string]any)
	for _, field := range []string{"password_hash", "session_version", "deleted_at"} {
		if _, ok := data[field]; ok {
			t.Fatalf("response must not expose trainer %s", field)
		}
		if _, ok := nested[field]; ok {
			t.Fatalf("response must not expose user %s", field)
		}
	}

	for _, sensitive := range []string{
		jwtValue,
		"access_token",
		testSecret,
		"password",
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}

func TestTrainerProfileNotAuthenticated(t *testing.T) {
	router, _, _, _ := newTrainerProfileTestRouter(t, trainerroles.PermissionProfile)

	rec, _, raw := trainerProfileRequest(router, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTrainerProfileInvalidToken(t *testing.T) {
	router, _, _, _ := newTrainerProfileTestRouter(t, trainerroles.PermissionProfile)

	rec, _, raw := trainerProfileRequest(router, "not.a.jwt", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
}

func TestTrainerProfileAuthenticatedNonTrainer(t *testing.T) {
	router, userRepo, _, tokenSvc := newTrainerProfileTestRouter(t, trainerroles.PermissionProfile)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerProfileRequest(router, jwtValue, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
		t.Fatalf("expected FORBIDDEN, got %s", raw)
	}
}

func TestTrainerProfilePermissionNotGranted(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newTrainerProfileTestRouter(t, trainerroles.Permission("trainer.schedule"))
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	seedTrainerForUser(t, trainerRepo, user)

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	rec, _, raw := trainerProfileRequest(router, jwtValue, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
		t.Fatalf("expected FORBIDDEN, got %s", raw)
	}
	if strings.Contains(raw, "trainer.schedule") {
		t.Fatalf("forbidden error must not reveal the permission, got %s", raw)
	}
}

func TestTrainerProfileQueryIdentityIgnored(t *testing.T) {
	router, userRepo, trainerRepo, tokenSvc := newTrainerProfileTestRouter(t, trainerroles.PermissionProfile)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, user)

	otherUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	otherTrainer := seedTrainerForUser(t, trainerRepo, otherUser)

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	query := "?user_id=" + otherUser.ID + "&trainer_id=" + otherTrainer.ID
	rec, data, raw := trainerProfileRequest(router, jwtValue, query)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if id, _ := data["id"].(string); id != trainer.ID {
		t.Fatalf("query identity must be ignored: expected trainer %q, got %q", trainer.ID, id)
	}
	if userID, _ := data["user_id"].(string); userID != user.ID {
		t.Fatalf("query identity must be ignored: expected user %q, got %q", user.ID, userID)
	}
}

func TestTrainerProfileMissingContextRejected(t *testing.T) {
	router := newTrainerProfileHandlerRouter(stubTrainerProfileService{}, nil)

	rec, _, raw := trainerProfileRequest(router, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTrainerProfileInvalidContextRejected(t *testing.T) {
	for name, identity := range map[string]any{
		"wrong type":    "some-trainer-id",
		"integer":       42,
		"empty both":    trainercontext.Identity{UserID: "", TrainerID: ""},
		"empty user":    trainercontext.Identity{UserID: "", TrainerID: uuid.NewString()},
		"empty trainer": trainercontext.Identity{UserID: uuid.NewString(), TrainerID: ""},
	} {
		t.Run(name, func(t *testing.T) {
			router := newTrainerProfileHandlerRouter(stubTrainerProfileService{}, identity)

			rec, _, raw := trainerProfileRequest(router, "", "")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
				t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
			}
		})
	}
}

func TestTrainerProfileTrainerNotFound(t *testing.T) {
	svc := stubTrainerProfileService{err: trainer_profile.ErrTrainerNotFound}
	router := newTrainerProfileHandlerRouter(svc, trainercontext.Identity{
		UserID:    uuid.NewString(),
		TrainerID: uuid.NewString(),
	})

	rec, _, raw := trainerProfileRequest(router, "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTrainerProfileRepositoryFailureNotExposed(t *testing.T) {
	svc := stubTrainerProfileService{err: errLoginRepoFailure}
	router := newTrainerProfileHandlerRouter(svc, trainercontext.Identity{
		UserID:    uuid.NewString(),
		TrainerID: uuid.NewString(),
	})

	rec, _, raw := trainerProfileRequest(router, "", "")
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
