package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/trainercontext"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/statistics"
	"ryze/backend/services/trainer_client_insights"
	"ryze/backend/services/trainer_clients"
	"ryze/backend/services/workout_sessions"
)

type insightClientStub struct {
	err error
}

func (s *insightClientStub) GetClient(context.Context, string, string) (*trainer_clients.Client, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &trainer_clients.Client{}, nil
}

type insightStatisticsStub struct{}

func (s *insightStatisticsStub) GetClientStatistics(context.Context, string) (*statistics.ClientStatisticsResponse, error) {
	return &statistics.ClientStatisticsResponse{}, nil
}

type insightSessionStub struct {
	err error
}

func (s *insightSessionStub) Get(context.Context, string, string) (*workout_sessions.Session, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &workout_sessions.Session{ID: "33333333-3333-3333-3333-333333333333", Status: models.WorkoutSessionStatusCompleted}, nil
}

type insightRepositoryStub struct{}

func (insightRepositoryStub) ListActiveAssignmentsByUser(context.Context, string) ([]models.ProgramAssignment, error) {
	return []models.ProgramAssignment{}, nil
}
func (insightRepositoryStub) ListActiveEntitlementsByUser(context.Context, string) ([]models.Entitlement, error) {
	return []models.Entitlement{}, nil
}
func (insightRepositoryStub) ListActiveFreeActivationsByUser(context.Context, string) ([]models.FreeProgramActivation, error) {
	return []models.FreeProgramActivation{}, nil
}
func (insightRepositoryStub) ListSessions(context.Context, string, int, int) ([]repositories.TrainerClientSessionSummary, error) {
	return []repositories.TrainerClientSessionSummary{}, nil
}
func (insightRepositoryStub) CountSessions(context.Context, string) (int64, error) {
	return 0, nil
}
func (insightRepositoryStub) FindActiveSession(context.Context, string) (*repositories.TrainerClientSessionSummary, error) {
	return nil, nil
}
func (insightRepositoryStub) LegacySummary(context.Context, string) (repositories.TrainerClientLegacySummary, error) {
	return repositories.TrainerClientLegacySummary{}, nil
}
func (insightRepositoryStub) ListExerciseSummaries(context.Context, string) ([]repositories.TrainerClientExerciseSummary, error) {
	return []repositories.TrainerClientExerciseSummary{}, nil
}
func (insightRepositoryStub) ListExerciseHistory(context.Context, string, string, int) ([]models.WorkoutSession, error) {
	return []models.WorkoutSession{}, nil
}

func newInsightHandlerRouter(clients trainer_client_insights.ClientAccess, sessions trainer_client_insights.SessionService, identity *trainercontext.Identity) *gin.Engine {
	service := trainer_client_insights.NewService(clients, &insightStatisticsStub{}, sessions, insightRepositoryStub{})
	handler := auth.NewTrainerClientInsightsHandler(service)
	router := gin.New()
	group := router.Group("/api/v1/trainer")
	group.Use(func(c *gin.Context) {
		if identity != nil {
			trainercontext.SetIdentity(c, *identity)
		}
		c.Next()
	})
	group.GET("/clients/:userID/overview", handler.GetOverview)
	group.GET("/clients/:userID/statistics", handler.GetStatistics)
	group.GET("/clients/:userID/workout-sessions", handler.ListWorkoutSessions)
	group.GET("/clients/:userID/workout-sessions/:sessionID", handler.GetWorkoutSession)
	group.GET("/clients/:userID/exercises/:exerciseID/history", handler.GetExerciseHistory)
	return router
}

func insightRequest(router http.Handler, method, path string) (*httptest.ResponseRecorder, map[string]any) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	router.ServeHTTP(recorder, request)
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		body = nil
	}
	return recorder, body
}

func TestTrainerClientInsightsHandlerOverview(t *testing.T) {
	identity := trainercontext.Identity{
		UserID:    "99999999-9999-9999-9999-999999999999",
		TrainerID: "11111111-1111-1111-1111-111111111111",
	}
	router := newInsightHandlerRouter(&insightClientStub{}, &insightSessionStub{}, &identity)

	recorder, body := insightRequest(router, http.MethodGet, "/api/v1/trainer/clients/22222222-2222-2222-2222-222222222222/overview")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if body["success"] != true {
		t.Fatalf("success missing: %v", body)
	}
	data, ok := body["data"].(map[string]any)
	if !ok || data["access"] == nil || data["statistics"] == nil {
		t.Fatalf("overview data = %v", body["data"])
	}
}

func TestTrainerClientInsightsHandlerErrors(t *testing.T) {
	identity := trainercontext.Identity{
		UserID:    "99999999-9999-9999-9999-999999999999",
		TrainerID: "11111111-1111-1111-1111-111111111111",
	}
	notFoundRouter := newInsightHandlerRouter(
		&insightClientStub{err: trainer_clients.ErrClientNotFound},
		&insightSessionStub{},
		&identity,
	)
	recorder, body := insightRequest(notFoundRouter, http.MethodGet, "/api/v1/trainer/clients/22222222-2222-2222-2222-222222222222/overview")
	if recorder.Code != http.StatusNotFound || body["error"].(map[string]any)["code"] != "CLIENT_NOT_FOUND" {
		t.Fatalf("unrelated client response = %d %v", recorder.Code, body)
	}

	missingContextRouter := newInsightHandlerRouter(&insightClientStub{}, &insightSessionStub{}, nil)
	recorder, _ = insightRequest(missingContextRouter, http.MethodGet, "/api/v1/trainer/clients/22222222-2222-2222-2222-222222222222/overview")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("missing context status = %d", recorder.Code)
	}

	router := newInsightHandlerRouter(&insightClientStub{}, &insightSessionStub{}, &identity)
	recorder, _ = insightRequest(router, http.MethodGet, "/api/v1/trainer/clients/22222222-2222-2222-2222-222222222222/workout-sessions?page=0")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid pagination status = %d", recorder.Code)
	}

	sessionNotFoundRouter := newInsightHandlerRouter(
		&insightClientStub{},
		&insightSessionStub{err: workout_sessions.ErrSessionNotFound},
		&identity,
	)
	recorder, body = insightRequest(sessionNotFoundRouter, http.MethodGet, "/api/v1/trainer/clients/22222222-2222-2222-2222-222222222222/workout-sessions/33333333-3333-3333-3333-333333333333")
	if recorder.Code != http.StatusNotFound || body["error"].(map[string]any)["code"] != "WORKOUT_SESSION_NOT_FOUND" {
		t.Fatalf("missing session response = %d %v", recorder.Code, body)
	}
}
