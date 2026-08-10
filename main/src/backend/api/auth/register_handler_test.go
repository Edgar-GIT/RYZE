package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/password"
	"ryze/backend/services/registration"
)

const registerRoute = "/api/v1/auth/register"

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestRouter builds a router backed by a real database transaction so the
// created users are rolled back and never persisted.
func newTestRouter(t *testing.T) *gin.Engine {
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

	svc := registration.NewRegistrationService(repositories.NewUserRepository(tx), password.Hasher{})
	handler := auth.NewRegisterHandler(svc)

	router := gin.New()
	router.POST(registerRoute, handler.Register)

	return router
}

// failingRepository simulates an unexpected repository failure.
type failingRepository struct{}

func (failingRepository) Create(_ context.Context, _ *models.User) error {
	return errors.New("database unreachable")
}
func (failingRepository) FindByID(_ context.Context, _ string) (*models.User, error) {
	return nil, repositories.ErrUserNotFound
}
func (failingRepository) FindByEmail(_ context.Context, _ string) (*models.User, error) {
	return nil, repositories.ErrUserNotFound
}
func (failingRepository) Update(_ context.Context, _ *models.User) error {
	return nil
}
func (failingRepository) SoftDelete(_ context.Context, _ string) error {
	return nil
}

func register(router http.Handler, body string) (*httptest.ResponseRecorder, map[string]any, string) {
	req := httptest.NewRequest(http.MethodPost, registerRoute, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
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

func uniqueEmail() string {
	return fmt.Sprintf("http-test-%d@ryze.local", time.Now().UnixNano())
}

func TestRegisterSuccess(t *testing.T) {
	router := newTestRouter(t)

	rec, data, raw := register(router, fmt.Sprintf(`{
		"email": %q,
		"password": "Str0ng!Passw0rd",
		"first_name": "John",
		"last_name": "Doe"
	}`, uniqueEmail()))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if data == nil {
		t.Fatal("expected data object in response")
	}
	if id, _ := data["id"].(string); id == "" {
		t.Fatal("expected non-empty user id")
	}
	if email, _ := data["email"].(string); email == "" {
		t.Fatal("expected non-empty user email")
	}
	if strings.Contains(raw, "password_hash") {
		t.Fatal("response must never contain password_hash")
	}
	if strings.Contains(raw, "Str0ng!Passw0rd") {
		t.Fatal("response must never contain the plaintext password")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	router := newTestRouter(t)
	email := uniqueEmail()
	body := fmt.Sprintf(`{
		"email": %q,
		"password": "Str0ng!Passw0rd",
		"first_name": "John",
		"last_name": "Doe"
	}`, email)

	rec, _, _ := register(router, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first registration: expected 201, got %d", rec.Code)
	}

	rec, _, raw := register(router, body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate registration: expected 409, got %d (body: %s)", rec.Code, raw)
	}
}

func TestRegisterInvalidJSON(t *testing.T) {
	router := newTestRouter(t)

	rec, _, _ := register(router, `{"email": "broken"`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON: expected 400, got %d", rec.Code)
	}
}

func TestRegisterMissingRequiredFields(t *testing.T) {
	router := newTestRouter(t)

	for name, body := range map[string]string{
		"empty object": `{}`,
		"missing email": `{
			"password": "Str0ng!Passw0rd",
			"first_name": "John",
			"last_name": "Doe"
		}`,
		"missing password": `{
			"email": "someone@ryze.local",
			"first_name": "John",
			"last_name": "Doe"
		}`,
		"missing first_name": `{
			"email": "someone@ryze.local",
			"password": "Str0ng!Passw0rd",
			"last_name": "Doe"
		}`,
		"missing last_name": `{
			"email": "someone@ryze.local",
			"password": "Str0ng!Passw0rd",
			"first_name": "John"
		}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec, _, _ := register(router, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRegisterResponseNeverContainsPassword(t *testing.T) {
	router := newTestRouter(t)

	rec, data, raw := register(router, fmt.Sprintf(`{
		"email": %q,
		"password": "Sup3rSecret!x",
		"first_name": "John",
		"last_name": "Doe"
	}`, uniqueEmail()))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if _, ok := data["password_hash"]; ok {
		t.Fatal("response data must not expose password_hash")
	}
	if strings.Contains(raw, "Sup3rSecret!x") {
		t.Fatal("response must never contain the plaintext password")
	}
}

func TestRegisterInternalError(t *testing.T) {
	svc := registration.NewRegistrationService(failingRepository{}, password.Hasher{})
	handler := auth.NewRegisterHandler(svc)

	router := gin.New()
	router.POST(registerRoute, handler.Register)

	rec, _, raw := register(router, `{
		"email": "someone@ryze.local",
		"password": "Str0ng!Passw0rd",
		"first_name": "John",
		"last_name": "Doe"
	}`)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, "database unreachable") {
		t.Fatal("internal error details must never be exposed")
	}
}
