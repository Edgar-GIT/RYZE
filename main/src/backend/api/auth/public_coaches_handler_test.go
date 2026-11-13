package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/api/auth"
	"ryze/backend/services/coach_showcase"
	"ryze/backend/services/token"
)

type fakeShowcase struct {
	viewer coach_showcase.Viewer
	err    error
}

func (f *fakeShowcase) RecordView(_ context.Context, _ string, viewer coach_showcase.Viewer) (bool, error) {
	f.viewer = viewer
	return f.err == nil, f.err
}

func (f *fakeShowcase) FeaturedPrograms(context.Context, int) ([]coach_showcase.Program, error) {
	return []coach_showcase.Program{{ID: "p1", Name: "Plan", Type: "free", Currency: "EUR", ProductType: "generic", Views: 4, Coach: coach_showcase.Coach{ID: "t1", Name: "Ana Ribeiro"}}}, nil
}

func (f *fakeShowcase) GetProfile(context.Context, string) (*coach_showcase.Profile, error) {
	return nil, coach_showcase.ErrNotFound
}

func (f *fakeShowcase) CoachOf(context.Context, string) (*coach_showcase.Coach, error) {
	return nil, coach_showcase.ErrNotFound
}

func coachesEngine(service coach_showcase.Service, tokens token.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := auth.NewPublicCoachesHandler(service, tokens)
	engine := gin.New()
	engine.POST("/programs/:programID/views", handler.RecordView)
	engine.GET("/coaches/featured-programs", handler.FeaturedPrograms)
	engine.GET("/coaches/:trainerID", handler.GetCoach)
	return engine
}

func TestPublicCoachesRecordViewUsesTheSessionWhenPresent(t *testing.T) {
	tokens := token.NewService([]byte("coach-handler-secret-longer-than-32-bytes"), 15*time.Minute)
	service := &fakeShowcase{}
	engine := coachesEngine(service, tokens)

	anonymous := httptest.NewRecorder()
	engine.ServeHTTP(anonymous, httptest.NewRequest(http.MethodPost, "/programs/p1/views", nil))
	if anonymous.Code != http.StatusNoContent || service.viewer.UserID != "" || !strings.HasPrefix(service.viewer.Key, "ip:") {
		t.Fatalf("anonymous view: %d %+v", anonymous.Code, service.viewer)
	}

	userID := uuid.NewString()
	access, err := tokens.GenerateAccessToken(userID, 0)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/programs/p1/views", nil)
	request.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: access})
	signedIn := httptest.NewRecorder()
	engine.ServeHTTP(signedIn, request)
	if signedIn.Code != http.StatusNoContent || service.viewer.UserID != userID || service.viewer.Key != "user:"+userID {
		t.Fatalf("signed-in view: %d %+v", signedIn.Code, service.viewer)
	}

	// A forged cookie is treated as anonymous, never rejected.
	forged := httptest.NewRequest(http.MethodPost, "/programs/p1/views", nil)
	forged.AddCookie(&http.Cookie{Name: auth.AccessTokenCookieName, Value: "forged"})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, forged)
	if recorder.Code != http.StatusNoContent || service.viewer.UserID != "" {
		t.Fatalf("forged cookie: %d %+v", recorder.Code, service.viewer)
	}
}

func TestPublicCoachesErrorsAndFeaturedShape(t *testing.T) {
	engine := coachesEngine(&fakeShowcase{err: coach_showcase.ErrInvalidInput}, nil)
	bad := httptest.NewRecorder()
	engine.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/programs/x/views", nil))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("malformed id = %d, want 400", bad.Code)
	}

	missing := httptest.NewRecorder()
	engine.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/coaches/t9", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("non-public coach = %d, want 404", missing.Code)
	}

	featured := httptest.NewRecorder()
	engine.ServeHTTP(featured, httptest.NewRequest(http.MethodGet, "/coaches/featured-programs", nil))
	var body struct {
		Data struct {
			Programs []map[string]any `json:"programs"`
		} `json:"data"`
	}
	if err := json.Unmarshal(featured.Body.Bytes(), &body); err != nil || len(body.Data.Programs) != 1 {
		t.Fatalf("featured body: %v %s", err, featured.Body.String())
	}
	program := body.Data.Programs[0]
	coach, _ := program["coach"].(map[string]any)
	if program["views"].(float64) != 4 || coach["name"] != "Ana Ribeiro" || coach["email"] != nil {
		t.Fatalf("unexpected featured program: %+v", program)
	}
}
