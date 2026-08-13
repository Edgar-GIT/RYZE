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
	"ryze/backend/services/trainer_clients"
)

const (
	clientsListRoute   = "/api/v1/trainer/clients"
	clientsRemoveRoute = "/api/v1/trainer/clients/"
)

// newTrainerClientsTestRouter wires the trainer client endpoints behind the
// real Authenticate, TrainerAuthenticate and RequireTrainerPermission
// middleware, backed by a database transaction so created records are rolled
// back. The required permissions can be customized to exercise the 403 path.
func newTrainerClientsTestRouter(t *testing.T, permissions ...trainerroles.Permission) (*gin.Engine, repositories.UserRepository, repositories.TrainerRepository, repositories.TrainerClientRepository, token.Service) {
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
	trainerClientRepo := repositories.NewTrainerClientRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	service := trainer_clients.NewService(trainerClientRepo, userRepo)
	handler := auth.NewTrainerClientsHandler(service)

	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(middleware.Authenticate(tokenSvc, userRepo))
	trainer.Use(middleware.TrainerAuthenticate(trainerRepo))
	trainer.GET("/clients", middleware.RequireTrainerPermission(permissions...), handler.ListClients)
	trainer.GET("/clients/:userID", middleware.RequireTrainerPermission(permissions...), handler.GetClient)
	trainer.POST("/clients", middleware.RequireTrainerPermission(permissions...), handler.AddClient)
	trainer.DELETE("/clients/:userID", middleware.RequireTrainerPermission(permissions...), handler.RemoveClient)
	trainer.POST("/clients/:userID/reactivate", middleware.RequireTrainerPermission(permissions...), handler.ReactivateClient)

	return router, userRepo, trainerRepo, trainerClientRepo, tokenSvc
}

// newTrainerClientsHandlerRouter mounts only the handler with a pre-set trainer
// context identity, so the handler's own error mapping can be tested without
// the full middleware chain. nil identity simulates a missing context.
func newTrainerClientsHandlerRouter(svc trainer_clients.Service, identity any) *gin.Engine {
	handler := auth.NewTrainerClientsHandler(svc)
	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(trainercontext.TrainerContextKey, identity)
		}
		c.Next()
	})
	trainer.GET("/clients", handler.ListClients)
	trainer.GET("/clients/:userID", handler.GetClient)
	trainer.POST("/clients", handler.AddClient)
	trainer.DELETE("/clients/:userID", handler.RemoveClient)
	trainer.POST("/clients/:userID/reactivate", handler.ReactivateClient)
	return router
}

// stubTrainerClientsService is a scripted fake used to exercise the handler's
// error mapping and identity forwarding without touching the database.
type stubTrainerClientsService struct {
	listResult trainer_clients.ListClientsResult
	client     *trainer_clients.Client
	err        error
	gotTrainer string
	gotUserID  string
}

func (s *stubTrainerClientsService) ListClients(_ context.Context, trainerID string, _, _ int) (trainer_clients.ListClientsResult, error) {
	s.gotTrainer = trainerID
	return s.listResult, s.err
}

func (s *stubTrainerClientsService) GetClient(_ context.Context, trainerID, userID string) (*trainer_clients.Client, error) {
	s.gotTrainer = trainerID
	s.gotUserID = userID
	return s.client, s.err
}

func (s *stubTrainerClientsService) AddClient(_ context.Context, trainerID, userID string) (*trainer_clients.Client, error) {
	s.gotTrainer = trainerID
	s.gotUserID = userID
	return s.client, s.err
}

func (s *stubTrainerClientsService) RemoveClient(_ context.Context, trainerID, userID string) error {
	s.gotTrainer = trainerID
	s.gotUserID = userID
	return s.err
}

func (s *stubTrainerClientsService) ReactivateClient(_ context.Context, trainerID, userID string) (*trainer_clients.Client, error) {
	s.gotTrainer = trainerID
	s.gotUserID = userID
	return s.client, s.err
}

func trainerClientsRequest(router http.Handler, cookieValue, method, path, body string) (*httptest.ResponseRecorder, map[string]any, string) {
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

// authenticatedTrainerCookie creates an active trainer owned by a fresh user
// and returns the trainer, the owning user and a valid access cookie.
func authenticatedTrainerCookie(t *testing.T, userRepo repositories.UserRepository, trainerRepo repositories.TrainerRepository, tokenSvc token.Service) (*models.Trainer, *models.User, string) {
	t.Helper()
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	trainer := seedTrainerForUser(t, trainerRepo, user)

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	return trainer, user, jwtValue
}

func TestTrainerClientsAddSuccess(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	body := `{"user_id":"` + clientUser.ID + `"}`
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, clientsListRoute, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	if id, _ := data["id"].(string); id == "" {
		t.Fatal("expected relationship id")
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("expected trainer_id %q, got %q", trainer.ID, trainerID)
	}
	if userID, _ := data["user_id"].(string); userID != clientUser.ID {
		t.Fatalf("expected user_id %q, got %q", clientUser.ID, userID)
	}
	nested, _ := data["user"].(map[string]any)
	if email, _ := nested["email"].(string); email != clientUser.Email {
		t.Fatalf("expected nested email %q, got %q", clientUser.Email, email)
	}

	relation, err := clientRepo.FindActiveByTrainerAndUser(context.Background(), trainer.ID, clientUser.ID)
	if err != nil {
		t.Fatalf("expected persisted active relationship: %v", err)
	}
	if relation.UserID != clientUser.ID {
		t.Fatalf("unexpected persisted relation %+v", relation)
	}
}

func TestTrainerClientsAddRejectsClientSuppliedTrainer(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	// A client-supplied trainer_id in the body must be ignored: the relationship
	// can only ever be created for the authenticated trainer.
	body := `{"user_id":"` + clientUser.ID + `","trainer_id":"` + otherTrainer.ID + `"}`
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, clientsListRoute, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("authenticated trainer must own the relationship, got %q", trainerID)
	}
}

func TestTrainerClientsListSuccess(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsListRoute+"?page=1&limit=10", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	clients, _ := data["clients"].([]any)
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}
	first, _ := clients[0].(map[string]any)
	if userID, _ := first["user_id"].(string); userID != clientUser.ID {
		t.Fatalf("expected client user %q, got %q", clientUser.ID, userID)
	}
	pagination, _ := data["pagination"].(map[string]any)
	if total, _ := pagination["total"].(float64); total != 1 {
		t.Fatalf("expected total 1, got %v", total)
	}
	if pages, _ := pagination["total_pages"].(float64); pages != 1 {
		t.Fatalf("expected total_pages 1, got %v", pages)
	}
}

func TestTrainerClientsListIgnoresQueryIdentity(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	otherClient := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	for _, pair := range [][2]string{{trainer.ID, clientUser.ID}, {otherTrainer.ID, otherClient.ID}} {
		if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: pair[0], UserID: pair[1]}); err != nil {
			t.Fatalf("seed relationship: %v", err)
		}
	}

	// A client-supplied trainer_id in the query must be ignored: only the
	// authenticated trainer's clients are ever listed.
	path := clientsListRoute + "?trainer_id=" + otherTrainer.ID
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	clients, _ := data["clients"].([]any)
	if len(clients) != 1 {
		t.Fatalf("expected only the authenticated trainer's client, got %d", len(clients))
	}
	first, _ := clients[0].(map[string]any)
	if userID, _ := first["user_id"].(string); userID != clientUser.ID {
		t.Fatalf("expected client %q, got %q", clientUser.ID, userID)
	}
}

func TestTrainerClientsGetProfileSuccess(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsRemoveRoute+clientUser.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("expected trainer_id %q, got %q", trainer.ID, trainerID)
	}
	if userID, _ := data["user_id"].(string); userID != clientUser.ID {
		t.Fatalf("expected user_id %q, got %q", clientUser.ID, userID)
	}
	nested, _ := data["user"].(map[string]any)
	if email, _ := nested["email"].(string); email != clientUser.Email {
		t.Fatalf("expected nested email %q, got %q", clientUser.Email, email)
	}
	if first, _ := nested["first_name"].(string); first != clientUser.FirstName {
		t.Fatalf("expected nested first_name %q, got %q", clientUser.FirstName, first)
	}
}

func TestTrainerClientsGetProfileIDOR(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)

	trainerA, _, jwtA := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	trainerB, _, jwtB := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	clientB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	for _, pair := range [][2]string{{trainerA.ID, clientA.ID}, {trainerB.ID, clientB.ID}} {
		if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: pair[0], UserID: pair[1]}); err != nil {
			t.Fatalf("seed relationship: %v", err)
		}
	}

	// Trainer A must never read client B's profile.
	rec, _, raw := trainerClientsRequest(router, jwtA, http.MethodGet, clientsRemoveRoute+clientB.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-trainer read, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, clientB.Email) {
		t.Fatalf("response must never contain the foreign client's data, got %s", raw)
	}

	// Trainer B must never read client A's profile.
	rec, _, raw = trainerClientsRequest(router, jwtB, http.MethodGet, clientsRemoveRoute+clientA.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-trainer read, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, clientA.Email) {
		t.Fatalf("response must never contain the foreign client's data, got %s", raw)
	}
}

func TestTrainerClientsGetProfileIgnoresQueryIdentity(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	// A client-supplied trainer_id query value must be ignored: the profile is
	// always resolved for the authenticated trainer only.
	path := clientsRemoveRoute + clientUser.ID + "?trainer_id=" + otherTrainer.ID
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("authenticated trainer must own the profile, got %q", trainerID)
	}
}

func TestTrainerClientsGetProfileIgnoresHeaderIdentity(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, clientsRemoveRoute+clientUser.ID, nil)
	req.Header.Set("X-Trainer-Id", otherTrainer.ID)
	req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: jwtValue})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	raw, _ := json.Marshal(payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	data, _ := payload["data"].(map[string]any)
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("header trainer id must never change ownership, got %q", trainerID)
	}
}

func TestTrainerClientsGetProfileIgnoresBodyIdentity(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	// A trainer_id in the body must be ignored: the profile is always resolved
	// for the authenticated trainer only.
	body := `{"trainer_id":"` + otherTrainer.ID + `"}`
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsRemoveRoute+clientUser.ID, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("body trainer id must never change ownership, got %q", trainerID)
	}
}

func TestTrainerClientsGetProfileUserWithoutRelation(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	_ = trainer
	otherUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	// The user exists but has no relationship with the trainer: the profile
	// must be denied and no user data may leak.
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsRemoveRoute+otherUser.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for user without relation, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, otherUser.Email) {
		t.Fatalf("response must never contain the unrelated user's data, got %s", raw)
	}
}

func TestTrainerClientsGetProfileUserNotFound(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	_ = trainer

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsRemoveRoute+uuid.NewString(), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown user, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"CLIENT_RELATION_NOT_FOUND"`) {
		t.Fatalf("expected CLIENT_RELATION_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerClientsGetProfileSoftDeletedUser(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	if err := userRepo.SoftDelete(context.Background(), clientUser.ID); err != nil {
		t.Fatalf("seed soft-deleted user: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsRemoveRoute+clientUser.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted user, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, clientUser.Email) {
		t.Fatalf("response must never contain a soft-deleted user's data, got %s", raw)
	}
}

func TestTrainerClientsGetProfileSoftDeletedRelation(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	relation := &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}
	if err := clientRepo.Create(context.Background(), relation); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	if err := clientRepo.SoftDelete(context.Background(), trainer.ID, clientUser.ID); err != nil {
		t.Fatalf("seed soft-deleted relation: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsRemoveRoute+clientUser.ID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted relation, got %d (body: %s)", rec.Code, raw)
	}
	if strings.Contains(raw, clientUser.Email) {
		t.Fatalf("response must never contain the soft-deleted relation's data, got %s", raw)
	}
}

func TestTrainerClientsGetProfileInvalidUserID(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	_ = trainer

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsRemoveRoute+"not-a-uuid", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed user id, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerClientsGetProfileInvalidToken(t *testing.T) {
	router, _, _, _, _ := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)

	rec, _, raw := trainerClientsRequest(router, "not.a.jwt", http.MethodGet, clientsRemoveRoute+uuid.NewString(), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTrainerClientsRemoveSuccess(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodDelete, clientsRemoveRoute+clientUser.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	// Only the relationship is gone; the user account and trainer profile stay.
	if _, err := clientRepo.FindActiveByTrainerAndUser(context.Background(), trainer.ID, clientUser.ID); err == nil {
		t.Fatal("expected the relationship to be soft-deleted")
	}
	if _, err := userRepo.FindByID(context.Background(), clientUser.ID); err != nil {
		t.Fatalf("client user account must survive: %v", err)
	}
	if _, err := trainerRepo.FindByID(context.Background(), trainer.ID); err != nil {
		t.Fatalf("trainer profile must survive: %v", err)
	}
}

func TestTrainerClientsReactivateSuccess(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	relation := &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}
	if err := clientRepo.Create(context.Background(), relation); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	if err := clientRepo.SoftDelete(context.Background(), trainer.ID, clientUser.ID); err != nil {
		t.Fatalf("seed soft delete: %v", err)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, clientsRemoveRoute+clientUser.ID+"/reactivate", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	// Reactivation restores the exact same row.
	if id, _ := data["id"].(string); id != relation.ID {
		t.Fatalf("expected reactivated relation id %q, got %q", relation.ID, id)
	}
	restored, err := clientRepo.FindActiveByTrainerAndUser(context.Background(), trainer.ID, clientUser.ID)
	if err != nil {
		t.Fatalf("expected the relationship to be active again: %v", err)
	}
	if restored.ID != relation.ID {
		t.Fatalf("expected the same row, got %q", restored.ID)
	}
}

func TestTrainerClientsNotAuthenticated(t *testing.T) {
	router, _, _, _, _ := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "list", method: http.MethodGet, path: clientsListRoute},
		{name: "profile", method: http.MethodGet, path: clientsRemoveRoute + uuid.NewString()},
		{name: "add", method: http.MethodPost, path: clientsListRoute, body: `{"user_id":"` + uuid.NewString() + `"}`},
		{name: "remove", method: http.MethodDelete, path: clientsRemoveRoute + uuid.NewString()},
		{name: "reactivate", method: http.MethodPost, path: clientsRemoveRoute + uuid.NewString() + "/reactivate"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerClientsRequest(router, "", tc.method, tc.path, tc.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
				t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
			}
		})
	}
}

func TestTrainerClientsAuthenticatedNonTrainer(t *testing.T) {
	router, userRepo, _, _, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	cases := []struct {
		name string
		path string
	}{
		{name: "list", path: clientsListRoute},
		{name: "profile", path: clientsRemoveRoute + uuid.NewString()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, tc.path, "")
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
				t.Fatalf("expected FORBIDDEN, got %s", raw)
			}
		})
	}
}

func TestTrainerClientsPermissionNotGranted(t *testing.T) {
	router, userRepo, trainerRepo, _, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.Permission("trainer.schedule"))
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	_ = trainer

	cases := []struct {
		name string
		path string
	}{
		{name: "list", path: clientsListRoute},
		{name: "profile", path: clientsRemoveRoute + uuid.NewString()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, tc.path, "")
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
			}
			if strings.Contains(raw, "trainer.schedule") {
				t.Fatalf("forbidden error must not reveal the permission, got %s", raw)
			}
		})
	}
}

func TestTrainerClientsHandlerForwardsContextIdentity(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerClientsService{
		client: &trainer_clients.Client{
			RelationID: uuid.NewString(),
			TrainerID:  identity.TrainerID,
			UserID:     identity.UserID,
			Email:      "client@ryze.local",
			FirstName:  "Jane",
			LastName:   "Roe",
		},
	}
	router := newTrainerClientsHandlerRouter(svc, identity)

	// The service receives exactly the context identity, never a client-supplied
	// value passed in the path.
	rec, _, raw := trainerClientsRequest(router, "", http.MethodDelete, clientsRemoveRoute+uuid.NewString(), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainer != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainer)
	}
}

func TestTrainerClientsGetProfileHandlerForwardsContextIdentity(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerClientsService{
		client: &trainer_clients.Client{
			RelationID: uuid.NewString(),
			TrainerID:  identity.TrainerID,
			UserID:     identity.UserID,
			Email:      "client@ryze.local",
			FirstName:  "Jane",
			LastName:   "Roe",
		},
	}
	router := newTrainerClientsHandlerRouter(svc, identity)

	pathUserID := uuid.NewString()
	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientsRemoveRoute+pathUserID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainer != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainer)
	}
	if svc.gotUserID != pathUserID {
		t.Fatalf("expected path user %q, got %q", pathUserID, svc.gotUserID)
	}
}

func TestTrainerClientsGetProfileHandlerMissingContext(t *testing.T) {
	router := newTrainerClientsHandlerRouter(&stubTrainerClientsService{}, nil)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientsRemoveRoute+uuid.NewString(), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTrainerClientsGetProfileHandlerErrorMapping(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: trainer_clients.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "relation not found", err: trainer_clients.ErrClientRelationNotFound, status: http.StatusNotFound, code: "CLIENT_RELATION_NOT_FOUND"},
		{name: "client not found", err: trainer_clients.ErrClientNotFound, status: http.StatusNotFound, code: "CLIENT_NOT_FOUND"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubTrainerClientsService{err: tc.err}
			router := newTrainerClientsHandlerRouter(svc, identity)

			rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientsRemoveRoute+uuid.NewString(), "")
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestTrainerClientsGetProfileHandlerRepositoryFailureNotExposed(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerClientsService{err: errLoginRepoFailure}
	router := newTrainerClientsHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientsRemoveRoute+uuid.NewString(), "")
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

func TestTrainerClientsGetProfileNeverExposesSecrets(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsRemoveRoute+clientUser.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	for _, sensitive := range []string{
		jwtValue,
		"access_token",
		testSecret,
		"password_hash",
		"session_version",
		"deleted_at",
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}

func TestTrainerClientsHandlerMissingContext(t *testing.T) {
	router := newTrainerClientsHandlerRouter(&stubTrainerClientsService{}, nil)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientsListRoute, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTrainerClientsHandlerErrorMapping(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: trainer_clients.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "client not found", err: trainer_clients.ErrClientNotFound, status: http.StatusNotFound, code: "CLIENT_NOT_FOUND"},
		{name: "already active", err: trainer_clients.ErrClientAlreadyActive, status: http.StatusConflict, code: "CLIENT_ALREADY_ADDED"},
		{name: "relation not found", err: trainer_clients.ErrClientRelationNotFound, status: http.StatusNotFound, code: "CLIENT_RELATION_NOT_FOUND"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubTrainerClientsService{err: tc.err}
			router := newTrainerClientsHandlerRouter(svc, identity)

			rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, clientsListRoute, `{"user_id":"`+uuid.NewString()+`"}`)
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestTrainerClientsHandlerRepositoryFailureNotExposed(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerClientsService{err: errLoginRepoFailure}
	router := newTrainerClientsHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientsListRoute, "")
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

func TestTrainerClientsInvalidJSONBody(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	router := newTrainerClientsHandlerRouter(&stubTrainerClientsService{}, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, clientsListRoute, `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerClientsInvalidPagination(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	router := newTrainerClientsHandlerRouter(&stubTrainerClientsService{}, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientsListRoute+"?page=abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerClientsNeverExposesSecrets(t *testing.T) {
	router, userRepo, trainerRepo, clientRepo, tokenSvc := newTrainerClientsTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientsListRoute, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	for _, sensitive := range []string{
		jwtValue,
		"access_token",
		testSecret,
		"password_hash",
		"session_version",
		"deleted_at",
	} {
		if strings.Contains(raw, sensitive) {
			t.Fatalf("response must never contain %q", sensitive)
		}
	}
}
