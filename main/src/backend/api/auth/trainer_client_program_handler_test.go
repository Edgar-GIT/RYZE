package auth_test

import (
	"context"
	"errors"
	"net/http"
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
	"ryze/backend/services/program_assignments"
	"ryze/backend/services/token"
)

// clientProgramsBaseRoute is the prefix shared by every program assignment
// endpoint; the requested client id and assignment id are appended by tests.
const clientProgramsBaseRoute = "/api/v1/trainer/clients/"

// newTrainerClientProgramTestRouter wires the trainer client program endpoints
// behind the real Authenticate, TrainerAuthenticate and
// RequireTrainerPermission middleware, backed by a database transaction so
// created records are rolled back. The required permissions can be customized
// to exercise the 403 path.
func newTrainerClientProgramTestRouter(t *testing.T, permissions ...trainerroles.Permission) (*gin.Engine, repositories.UserRepository, repositories.TrainerRepository, repositories.ProgramRepository, repositories.TrainerClientRepository, repositories.ProgramAssignmentRepository, token.Service) {
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
	trainerClientRepo := repositories.NewTrainerClientRepository(tx)
	programAssignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	tokenSvc := token.NewService([]byte(testSecret), testTokenTTL)

	service := program_assignments.NewService(programAssignmentRepo)
	handler := auth.NewTrainerClientProgramHandler(service)

	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(middleware.Authenticate(tokenSvc, userRepo))
	trainer.Use(middleware.TrainerAuthenticate(trainerRepo))
	trainer.POST("/clients/:userID/programs", middleware.RequireTrainerPermission(permissions...), handler.AssignProgram)
	trainer.GET("/clients/:userID/programs", middleware.RequireTrainerPermission(permissions...), handler.ListClientPrograms)
	trainer.DELETE("/clients/:userID/programs/:assignmentID", middleware.RequireTrainerPermission(permissions...), handler.RemoveAssignment)

	return router, userRepo, trainerRepo, programRepo, trainerClientRepo, programAssignmentRepo, tokenSvc
}

// newTrainerClientProgramHandlerRouter mounts only the handler with a pre-set
// trainer context identity, so the handler's own error mapping can be tested
// without the full middleware chain. nil identity simulates a missing context.
func newTrainerClientProgramHandlerRouter(svc program_assignments.Service, identity any) *gin.Engine {
	handler := auth.NewTrainerClientProgramHandler(svc)
	router := gin.New()
	trainer := router.Group("/api/v1/trainer")
	trainer.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set(trainercontext.TrainerContextKey, identity)
		}
		c.Next()
	})
	trainer.POST("/clients/:userID/programs", handler.AssignProgram)
	trainer.GET("/clients/:userID/programs", handler.ListClientPrograms)
	trainer.DELETE("/clients/:userID/programs/:assignmentID", handler.RemoveAssignment)
	return router
}

// stubTrainerClientProgramService is a scripted fake used to exercise the
// handler's error mapping and identity forwarding without touching the
// database.
type stubTrainerClientProgramService struct {
	assignment   *program_assignments.Assignment
	assignments  []program_assignments.Assignment
	err          error
	gotTrainer   string
	gotUserID    string
	gotProgramID string
	gotEntryID   string
}

func (s *stubTrainerClientProgramService) AssignProgram(_ context.Context, trainerID, userID, programID string) (*program_assignments.Assignment, error) {
	s.gotTrainer = trainerID
	s.gotUserID = userID
	s.gotProgramID = programID
	return s.assignment, s.err
}

func (s *stubTrainerClientProgramService) ListClientPrograms(_ context.Context, trainerID, userID string) ([]program_assignments.Assignment, error) {
	s.gotTrainer = trainerID
	s.gotUserID = userID
	return s.assignments, s.err
}

func (s *stubTrainerClientProgramService) RemoveAssignment(_ context.Context, trainerID, userID, assignmentID string) error {
	s.gotTrainer = trainerID
	s.gotUserID = userID
	s.gotEntryID = assignmentID
	return s.err
}

func TestTrainerClientProgramAssignSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, assignmentRepo, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Strength Builder", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+program.ID+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	if id, _ := data["id"].(string); id == "" {
		t.Fatal("expected assignment id")
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("expected trainer_id %q, got %q", trainer.ID, trainerID)
	}
	if userID, _ := data["user_id"].(string); userID != clientUser.ID {
		t.Fatalf("expected user_id %q, got %q", clientUser.ID, userID)
	}
	if programID, _ := data["program_id"].(string); programID != program.ID {
		t.Fatalf("expected program_id %q, got %q", program.ID, programID)
	}
	nested, _ := data["program"].(map[string]any)
	if name, _ := nested["name"].(string); name != program.Name {
		t.Fatalf("expected nested program name %q, got %q", program.Name, name)
	}

	assignments, err := assignmentRepo.ListByClient(context.Background(), trainer.ID, clientUser.ID)
	if err != nil {
		t.Fatalf("expected persisted assignment: %v", err)
	}
	if len(assignments) != 1 || assignments[0].UserID != clientUser.ID {
		t.Fatalf("unexpected persisted assignments %+v", assignments)
	}
}

func TestTrainerClientProgramAssignRejectsClientSuppliedTrainer(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Owned Program", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	// A client-supplied trainer_id in the body must be ignored: the assignment
	// can only ever be created for the authenticated trainer.
	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	body := `{"program_id":"` + program.ID + `","trainer_id":"` + otherTrainer.ID + `"}`
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	if trainerID, _ := data["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("authenticated trainer must own the assignment, got %q", trainerID)
	}
}

// TestTrainerClientProgramIDORMatrix exercises the full middleware chain and
// database against two trainers, two clients and two programs: every operation
// must be scoped to the authenticated trainer's owned program and managed
// client, and cross-tenant references must never leak data. This is the
// end-to-end coverage of the assignment lifecycle under the existing
// MariaDB-backed test infrastructure.
func TestTrainerClientProgramIDORMatrix(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)

	trainerA, _, jwtA := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	trainerB, _, jwtB := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	clientB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	for _, pair := range [][2]string{{trainerA.ID, clientA.ID}, {trainerB.ID, clientB.ID}} {
		if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: pair[0], UserID: pair[1]}); err != nil {
			t.Fatalf("seed relationship: %v", err)
		}
	}

	programA := &models.Program{TrainerID: trainerA.ID, Name: "Program A", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	programB := &models.Program{TrainerID: trainerB.ID, Name: "Program B", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	for _, p := range []*models.Program{programA, programB} {
		if err := programRepo.Create(context.Background(), p); err != nil {
			t.Fatalf("seed program: %v", err)
		}
	}

	assign := func(jwt, clientID, programID string) (int, string) {
		rec, _, raw := trainerClientsRequest(router, jwt, http.MethodPost, clientProgramsBaseRoute+clientID+"/programs", `{"program_id":"`+programID+`"}`)
		return rec.Code, raw
	}

	// Trainer A can assign Program A → Client A.
	if code, raw := assign(jwtA, clientA.ID, programA.ID); code != http.StatusCreated {
		t.Fatalf("expected 201 for A→(A,A), got %d (body: %s)", code, raw)
	}
	// Trainer A can never assign Program A → Client B.
	if code, raw := assign(jwtA, clientB.ID, programA.ID); code != http.StatusNotFound {
		t.Fatalf("expected 404 for A→(B,A), got %d (body: %s)", code, raw)
	} else if !strings.Contains(raw, `"code":"CLIENT_RELATION_NOT_FOUND"`) {
		t.Fatalf("expected CLIENT_RELATION_NOT_FOUND, got %s", raw)
	}
	// Trainer A can never assign Program B → Client A.
	if code, raw := assign(jwtA, clientA.ID, programB.ID); code != http.StatusNotFound {
		t.Fatalf("expected 404 for A→(A,B), got %d (body: %s)", code, raw)
	} else if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
	// Trainer A can never assign Program B → Client B (foreign client wins).
	if code, raw := assign(jwtA, clientB.ID, programB.ID); code != http.StatusNotFound {
		t.Fatalf("expected 404 for A→(B,B), got %d (body: %s)", code, raw)
	} else if !strings.Contains(raw, `"code":"CLIENT_RELATION_NOT_FOUND"`) {
		t.Fatalf("expected CLIENT_RELATION_NOT_FOUND, got %s", raw)
	}
	// Trainer B can assign Program B → Client B.
	if code, raw := assign(jwtB, clientB.ID, programB.ID); code != http.StatusCreated {
		t.Fatalf("expected 201 for B→(B,B), got %d (body: %s)", code, raw)
	}
	// Trainer B can never assign Program B → Client A.
	if code, raw := assign(jwtB, clientA.ID, programB.ID); code != http.StatusNotFound {
		t.Fatalf("expected 404 for B→(A,B), got %d (body: %s)", code, raw)
	}
	// Trainer B can never assign Program A → Client B.
	if code, raw := assign(jwtB, clientB.ID, programA.ID); code != http.StatusNotFound {
		t.Fatalf("expected 404 for B→(B,A), got %d (body: %s)", code, raw)
	} else if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}

	// No foreign program or client data may ever leak into a response.
	for _, raw := range []string{
		mustRaw(t, router, jwtA, http.MethodGet, clientProgramsBaseRoute+clientA.ID+"/programs"),
		mustRaw(t, router, jwtB, http.MethodGet, clientProgramsBaseRoute+clientB.ID+"/programs"),
	} {
		if strings.Contains(raw, "Program B") || strings.Contains(raw, "Program A") {
			if strings.Contains(raw, clientA.Email) || strings.Contains(raw, clientB.Email) {
				t.Fatalf("foreign client data leaked: %s", raw)
			}
		}
		if strings.Contains(raw, clientA.Email) || strings.Contains(raw, clientB.Email) {
			t.Fatalf("response must never contain client emails: %s", raw)
		}
	}
}

func mustRaw(t *testing.T, router http.Handler, jwtValue, method, path string) string {
	t.Helper()
	rec, _, raw := trainerClientsRequest(router, jwtValue, method, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	return raw
}

func TestTrainerClientProgramAssignAlreadyActive(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Strength Builder", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	if rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+program.ID+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	second := &models.Program{TrainerID: trainer.ID, Name: "HIIT Blaster", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), second); err != nil {
		t.Fatalf("seed second program: %v", err)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+second.ID+`"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"ASSIGNMENT_ALREADY_ACTIVE"`) {
		t.Fatalf("expected ASSIGNMENT_ALREADY_ACTIVE, got %s", raw)
	}
}

func TestTrainerClientProgramAssignSoftDeletedClient(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	if err := userRepo.SoftDelete(context.Background(), clientUser.ID); err != nil {
		t.Fatalf("seed soft-deleted user: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Strength Builder", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+program.ID+`"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted client, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"CLIENT_RELATION_NOT_FOUND"`) {
		t.Fatalf("expected CLIENT_RELATION_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerClientProgramAssignSoftDeletedRelation(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	if err := clientRepo.SoftDelete(context.Background(), trainer.ID, clientUser.ID); err != nil {
		t.Fatalf("seed soft-deleted relation: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Strength Builder", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+program.ID+`"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted relation, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"CLIENT_RELATION_NOT_FOUND"`) {
		t.Fatalf("expected CLIENT_RELATION_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerClientProgramAssignSoftDeletedProgram(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Soon Deleted", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}
	if err := programRepo.SoftDelete(context.Background(), trainer.ID, program.ID); err != nil {
		t.Fatalf("seed soft-deleted program: %v", err)
	}

	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+program.ID+`"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for soft-deleted program, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"PROGRAM_NOT_FOUND"`) {
		t.Fatalf("expected PROGRAM_NOT_FOUND, got %s", raw)
	}
}

func TestTrainerClientProgramAssignMissingProgramID(t *testing.T) {
	router, userRepo, trainerRepo, _, _, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	_ = trainer

	path := clientProgramsBaseRoute + uuid.NewString() + "/programs"
	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerClientProgramListSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Strength Builder", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	if rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+program.ID+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	programs, _ := data["programs"].([]any)
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	first, _ := programs[0].(map[string]any)
	if programID, _ := first["program_id"].(string); programID != program.ID {
		t.Fatalf("expected program_id %q, got %q", program.ID, programID)
	}
	if userID, _ := first["user_id"].(string); userID != clientUser.ID {
		t.Fatalf("expected user_id %q, got %q", clientUser.ID, userID)
	}
	nested, _ := first["program"].(map[string]any)
	if name, _ := nested["name"].(string); name != program.Name {
		t.Fatalf("expected nested program name %q, got %q", program.Name, name)
	}
}

func TestTrainerClientProgramListIgnoresQueryIdentity(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Strength Builder", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	if rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+program.ID+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	// A client-supplied trainer_id in the query must be ignored.
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, path+"?trainer_id="+otherTrainer.ID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	programs, _ := data["programs"].([]any)
	if len(programs) != 1 {
		t.Fatalf("expected the authenticated trainer's program, got %d", len(programs))
	}
	first, _ := programs[0].(map[string]any)
	if trainerID, _ := first["trainer_id"].(string); trainerID != trainer.ID {
		t.Fatalf("authenticated trainer must own the assignment, got %q", trainerID)
	}
}

func TestTrainerClientProgramListForeignClientEmpty(t *testing.T) {
	router, userRepo, trainerRepo, _, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	_ = trainer
	otherTrainer := seedTrainerForUser(t, trainerRepo, seedLoginUser(t, userRepo, uniqueEmail(), "Password123!"))
	otherClient := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: otherTrainer.ID, UserID: otherClient.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	// A client managed by another trainer is indistinguishable from a client
	// without assignments; no foreign data may leak.
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, clientProgramsBaseRoute+otherClient.ID+"/programs", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	programs, _ := data["programs"].([]any)
	if len(programs) != 0 {
		t.Fatalf("expected empty list for a foreign client, got %d", len(programs))
	}
	if strings.Contains(raw, otherClient.Email) {
		t.Fatalf("response must never contain the foreign client's data, got %s", raw)
	}
}

func TestTrainerClientProgramRemoveSuccess(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, assignmentRepo, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Strength Builder", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	rec, data, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+program.ID+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	assignmentID, _ := data["id"].(string)

	rec, _, raw = trainerClientsRequest(router, jwtValue, http.MethodDelete, path+"/"+assignmentID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}

	if _, err := assignmentRepo.FindByIDAndClient(context.Background(), trainer.ID, clientUser.ID, assignmentID); !errors.Is(err, repositories.ErrAssignmentNotFound) {
		t.Fatalf("expected the assignment to be soft-deleted, got %v", err)
	}
	if _, err := programRepo.FindByIDAndTrainer(context.Background(), trainer.ID, program.ID); err != nil {
		t.Fatalf("the program must survive the assignment delete: %v", err)
	}
	if _, err := clientRepo.FindActiveByTrainerAndUser(context.Background(), trainer.ID, clientUser.ID); err != nil {
		t.Fatalf("the relationship must survive the assignment delete: %v", err)
	}
}

func TestTrainerClientProgramRemoveIDOR(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)

	trainerA, _, jwtA := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	trainerB, _, jwtB := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientA := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	clientB := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	for _, pair := range [][2]string{{trainerA.ID, clientA.ID}, {trainerB.ID, clientB.ID}} {
		if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: pair[0], UserID: pair[1]}); err != nil {
			t.Fatalf("seed relationship: %v", err)
		}
	}
	programA := &models.Program{TrainerID: trainerA.ID, Name: "Program A", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	programB := &models.Program{TrainerID: trainerB.ID, Name: "Program B", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	for _, p := range []*models.Program{programA, programB} {
		if err := programRepo.Create(context.Background(), p); err != nil {
			t.Fatalf("seed program: %v", err)
		}
	}

	pathA := clientProgramsBaseRoute + clientA.ID + "/programs"
	rec, data, raw := trainerClientsRequest(router, jwtA, http.MethodPost, pathA, `{"program_id":"`+programA.ID+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}
	assignmentID, _ := data["id"].(string)

	// Trainer A cannot remove client B's assignment path, and trainer B cannot
	// remove trainer A's assignment even with its own client path.
	cases := []struct {
		name   string
		jwt    string
		client string
		id     string
	}{
		{name: "cross-trainer", jwt: jwtB, client: clientA.ID, id: assignmentID},
		{name: "cross-client", jwt: jwtA, client: clientB.ID, id: assignmentID},
		{name: "unknown assignment", jwt: jwtA, client: clientA.ID, id: uuid.NewString()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerClientsRequest(router, tc.jwt, http.MethodDelete, clientProgramsBaseRoute+tc.client+"/programs/"+tc.id, "")
			if rec.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"ASSIGNMENT_NOT_FOUND"`) {
				t.Fatalf("expected ASSIGNMENT_NOT_FOUND, got %s", raw)
			}
		})
	}

	// The target assignment must be untouched.
	if _, err := clientRepo.FindActiveByTrainerAndUser(context.Background(), trainerA.ID, clientA.ID); err != nil {
		t.Fatalf("relationship must survive failed removes: %v", err)
	}
}

func TestTrainerClientProgramNotAuthenticated(t *testing.T) {
	router, _, _, _, _, _, _ := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "assign", method: http.MethodPost, path: clientProgramsBaseRoute + uuid.NewString() + "/programs", body: `{"program_id":"` + uuid.NewString() + `"}`},
		{name: "list", method: http.MethodGet, path: clientProgramsBaseRoute + uuid.NewString() + "/programs"},
		{name: "remove", method: http.MethodDelete, path: clientProgramsBaseRoute + uuid.NewString() + "/programs/" + uuid.NewString()},
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

func TestTrainerClientProgramAuthenticatedNonTrainer(t *testing.T) {
	router, userRepo, _, _, _, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	user := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")

	jwtValue, err := tokenSvc.GenerateAccessToken(user.ID, user.SessionVersion)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "assign", method: http.MethodPost, path: clientProgramsBaseRoute + uuid.NewString() + "/programs", body: `{"program_id":"` + uuid.NewString() + `"}`},
		{name: "list", method: http.MethodGet, path: clientProgramsBaseRoute + uuid.NewString() + "/programs"},
		{name: "remove", method: http.MethodDelete, path: clientProgramsBaseRoute + uuid.NewString() + "/programs/" + uuid.NewString()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, _, raw := trainerClientsRequest(router, jwtValue, tc.method, tc.path, tc.body)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d (body: %s)", rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"FORBIDDEN"`) {
				t.Fatalf("expected FORBIDDEN, got %s", raw)
			}
		})
	}
}

func TestTrainerClientProgramPermissionNotGranted(t *testing.T) {
	router, userRepo, trainerRepo, _, _, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.Permission("trainer.schedule"))
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	_ = trainer

	cases := []struct {
		name string
		path string
	}{
		{name: "list", path: clientProgramsBaseRoute + uuid.NewString() + "/programs"},
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

func TestTrainerClientProgramHandlerForwardsContextIdentity(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerClientProgramService{
		assignment: &program_assignments.Assignment{
			ID:        uuid.NewString(),
			TrainerID: identity.TrainerID,
			UserID:    identity.UserID,
			ProgramID: uuid.NewString(),
			Program: program_assignments.Program{
				ID:   uuid.NewString(),
				Name: "Strength Builder",
			},
		},
	}
	router := newTrainerClientProgramHandlerRouter(svc, identity)

	pathUserID := uuid.NewString()
	pathAssignmentID := uuid.NewString()
	rec, _, raw := trainerClientsRequest(router, "", http.MethodDelete, clientProgramsBaseRoute+pathUserID+"/programs/"+pathAssignmentID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, raw)
	}
	if svc.gotTrainer != identity.TrainerID {
		t.Fatalf("expected context trainer %q, got %q", identity.TrainerID, svc.gotTrainer)
	}
	if svc.gotUserID != pathUserID {
		t.Fatalf("expected path user %q, got %q", pathUserID, svc.gotUserID)
	}
	if svc.gotEntryID != pathAssignmentID {
		t.Fatalf("expected path assignment %q, got %q", pathAssignmentID, svc.gotEntryID)
	}
}

func TestTrainerClientProgramHandlerMissingContext(t *testing.T) {
	router := newTrainerClientProgramHandlerRouter(&stubTrainerClientProgramService{}, nil)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodGet, clientProgramsBaseRoute+uuid.NewString()+"/programs", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"AUTHENTICATION_REQUIRED"`) {
		t.Fatalf("expected AUTHENTICATION_REQUIRED, got %s", raw)
	}
}

func TestTrainerClientProgramHandlerErrorMapping(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid input", err: program_assignments.ErrInvalidInput, status: http.StatusBadRequest, code: "VALIDATION_ERROR"},
		{name: "client relation not found", err: program_assignments.ErrClientRelationNotFound, status: http.StatusNotFound, code: "CLIENT_RELATION_NOT_FOUND"},
		{name: "program not found", err: program_assignments.ErrProgramNotFound, status: http.StatusNotFound, code: "PROGRAM_NOT_FOUND"},
		{name: "assignment not found", err: program_assignments.ErrAssignmentNotFound, status: http.StatusNotFound, code: "ASSIGNMENT_NOT_FOUND"},
		{name: "already active", err: program_assignments.ErrAssignmentAlreadyActive, status: http.StatusConflict, code: "ASSIGNMENT_ALREADY_ACTIVE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubTrainerClientProgramService{err: tc.err}
			router := newTrainerClientProgramHandlerRouter(svc, identity)

			rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, clientProgramsBaseRoute+uuid.NewString()+"/programs", `{"program_id":"`+uuid.NewString()+`"}`)
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d (body: %s)", tc.status, rec.Code, raw)
			}
			if !strings.Contains(raw, `"code":"`+tc.code+`"`) {
				t.Fatalf("expected code %s, got %s", tc.code, raw)
			}
		})
	}
}

func TestTrainerClientProgramHandlerRepositoryFailureNotExposed(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	svc := &stubTrainerClientProgramService{err: errLoginRepoFailure}
	router := newTrainerClientProgramHandlerRouter(svc, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, clientProgramsBaseRoute+uuid.NewString()+"/programs", `{"program_id":"`+uuid.NewString()+`"}`)
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

func TestTrainerClientProgramInvalidJSONBody(t *testing.T) {
	identity := trainercontext.Identity{UserID: uuid.NewString(), TrainerID: uuid.NewString()}
	router := newTrainerClientProgramHandlerRouter(&stubTrainerClientProgramService{}, identity)

	rec, _, raw := trainerClientsRequest(router, "", http.MethodPost, clientProgramsBaseRoute+uuid.NewString()+"/programs", `not json`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", rec.Code, raw)
	}
	if !strings.Contains(raw, `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("expected VALIDATION_ERROR, got %s", raw)
	}
}

func TestTrainerClientProgramNeverExposesSecrets(t *testing.T) {
	router, userRepo, trainerRepo, programRepo, clientRepo, _, tokenSvc := newTrainerClientProgramTestRouter(t, trainerroles.PermissionClients)
	trainer, _, jwtValue := authenticatedTrainerCookie(t, userRepo, trainerRepo, tokenSvc)
	clientUser := seedLoginUser(t, userRepo, uniqueEmail(), "Password123!")
	if err := clientRepo.Create(context.Background(), &models.TrainerClient{TrainerID: trainer.ID, UserID: clientUser.ID}); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Strength Builder", Type: models.ProgramTypePremium, Status: models.ProgramStatusDraft}
	if err := programRepo.Create(context.Background(), program); err != nil {
		t.Fatalf("seed program: %v", err)
	}

	path := clientProgramsBaseRoute + clientUser.ID + "/programs"
	if rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodPost, path, `{"program_id":"`+program.ID+`"}`); rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body: %s)", rec.Code, raw)
	}

	rec, _, raw := trainerClientsRequest(router, jwtValue, http.MethodGet, path, "")
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
