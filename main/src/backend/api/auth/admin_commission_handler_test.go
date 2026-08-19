package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/services/commission_rules"
)

// --- stubs ---

type stubCommissionService struct {
	rule       *commission_rules.CommissionRule
	resolution commission_rules.CommissionResolution
	err        error
	gotTrainer string
	gotBPS     uint32
}

func (s *stubCommissionService) GetCommissionRule(_ context.Context, trainerID string) (*commission_rules.CommissionRule, error) {
	s.gotTrainer = trainerID
	return s.rule, s.err
}

func (s *stubCommissionService) UpsertCommissionRule(_ context.Context, trainerID string, commissionBPS uint32) (*commission_rules.CommissionRule, error) {
	s.gotTrainer = trainerID
	s.gotBPS = commissionBPS
	return s.rule, s.err
}

func (s *stubCommissionService) DeleteCommissionRule(_ context.Context, trainerID string) error {
	s.gotTrainer = trainerID
	return s.err
}

func (s *stubCommissionService) ResolveCommission(_ context.Context, trainerID string) (commission_rules.CommissionResolution, error) {
	s.gotTrainer = trainerID
	return s.resolution, s.err
}

func (s *stubCommissionService) CalculateCommissionSplit(priceMinorUnits int64, resolution commission_rules.CommissionResolution) commission_rules.CommissionCalculation {
	platformAmount := (priceMinorUnits * int64(resolution.CommissionBPS)) / 10000
	return commission_rules.CommissionCalculation{
		PlatformAmount: platformAmount,
		TrainerAmount:  priceMinorUnits - platformAmount,
	}
}

// --- helpers ---

func newCommissionHandlerRouter(svc commission_rules.Service) *gin.Engine {
	handler := auth.NewAdminCommissionHandler(svc)
	router := gin.New()
	admin := router.Group("/api/v1/admin")
	admin.GET("/trainers/:id/commission", handler.GetCommissionRule)
	admin.PATCH("/trainers/:id/commission", handler.UpsertCommissionRule)
	admin.DELETE("/trainers/:id/commission", handler.DeleteCommissionRule)
	admin.GET("/trainers/:id/commission/resolve", handler.GetCommissionResolution)
	return router
}

func validTrainerID() string {
	return "11111111-1111-1111-1111-111111111111"
}

func performRequest(router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return result
}

// --- GetCommissionRule tests ---

func TestGetCommissionRule_Success(t *testing.T) {
	expectedRule := &commission_rules.CommissionRule{
		ID:            "22222222-2222-2222-2222-222222222222",
		TrainerID:     validTrainerID(),
		CommissionBPS: 1500,
	}

	svc := &stubCommissionService{rule: expectedRule}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "GET", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	result := parseResponse(t, w)
	if success, ok := result["success"].(bool); !ok || !success {
		t.Fatalf("expected success=true, got %v", result["success"])
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["commission_bps"].(float64) != 1500 {
		t.Fatalf("expected commission_bps=1500, got %v", data["commission_bps"])
	}
	if data["trainer_id"] != validTrainerID() {
		t.Fatalf("expected trainer_id=%s, got %v", validTrainerID(), data["trainer_id"])
	}
}

func TestGetCommissionRule_NotFound(t *testing.T) {
	svc := &stubCommissionService{err: commission_rules.ErrCommissionRuleNotFound}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "GET", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	result := parseResponse(t, w)
	errorObj, ok := result["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error to be an object")
	}
	if code, ok := errorObj["code"].(string); !ok || code != "COMMISSION_RULE_NOT_FOUND" {
		t.Fatalf("expected code=COMMISSION_RULE_NOT_FOUND, got %v", errorObj["code"])
	}
}

func TestGetCommissionRule_InternalError(t *testing.T) {
	svc := &stubCommissionService{err: errors.New("db failure")}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "GET", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

// --- UpsertCommissionRule tests ---

func TestUpsertCommissionRule_Success(t *testing.T) {
	expectedRule := &commission_rules.CommissionRule{
		ID:            "22222222-2222-2222-2222-222222222222",
		TrainerID:     validTrainerID(),
		CommissionBPS: 1500,
	}

	svc := &stubCommissionService{rule: expectedRule}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "PATCH", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", map[string]any{
		"commission_bps": 1500,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	result := parseResponse(t, w)
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["commission_bps"].(float64) != 1500 {
		t.Fatalf("expected commission_bps=1500, got %v", data["commission_bps"])
	}
	if svc.gotBPS != 1500 {
		t.Fatalf("expected service to receive bps=1500, got %d", svc.gotBPS)
	}
}

func TestUpsertCommissionRule_InvalidBody(t *testing.T) {
	svc := &stubCommissionService{}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "PATCH", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", "invalid")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestUpsertCommissionRule_TrainerNotFound(t *testing.T) {
	svc := &stubCommissionService{err: commission_rules.ErrTrainerNotFound}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "PATCH", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", map[string]any{
		"commission_bps": 1500,
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestUpsertCommissionRule_InvalidInput(t *testing.T) {
	svc := &stubCommissionService{err: commission_rules.ErrInvalidInput}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "PATCH", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", map[string]any{
		"commission_bps": 1500,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

// --- DeleteCommissionRule tests ---

func TestDeleteCommissionRule_Success(t *testing.T) {
	svc := &stubCommissionService{}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "DELETE", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	result := parseResponse(t, w)
	if success, ok := result["success"].(bool); !ok || !success {
		t.Fatalf("expected success=true, got %v", result["success"])
	}
	if svc.gotTrainer != validTrainerID() {
		t.Fatalf("expected service to receive trainer_id=%s, got %s", validTrainerID(), svc.gotTrainer)
	}
}

func TestDeleteCommissionRule_NotFound(t *testing.T) {
	svc := &stubCommissionService{err: commission_rules.ErrCommissionRuleNotFound}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "DELETE", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestDeleteCommissionRule_InternalError(t *testing.T) {
	svc := &stubCommissionService{err: errors.New("db failure")}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "DELETE", "/api/v1/admin/trainers/"+validTrainerID()+"/commission", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

// --- GetCommissionResolution tests ---

func TestGetCommissionResolution_Override(t *testing.T) {
	svc := &stubCommissionService{
		resolution: commission_rules.CommissionResolution{CommissionBPS: 1500, IsOverride: true},
	}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "GET", "/api/v1/admin/trainers/"+validTrainerID()+"/commission/resolve", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	result := parseResponse(t, w)
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["commission_bps"].(float64) != 1500 {
		t.Fatalf("expected commission_bps=1500, got %v", data["commission_bps"])
	}
	if data["is_override"] != true {
		t.Fatalf("expected is_override=true, got %v", data["is_override"])
	}
}

func TestGetCommissionResolution_Default(t *testing.T) {
	svc := &stubCommissionService{
		resolution: commission_rules.CommissionResolution{CommissionBPS: 2000, IsOverride: false},
	}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "GET", "/api/v1/admin/trainers/"+validTrainerID()+"/commission/resolve", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	result := parseResponse(t, w)
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data to be an object")
	}
	if data["commission_bps"].(float64) != 2000 {
		t.Fatalf("expected commission_bps=2000, got %v", data["commission_bps"])
	}
	if data["is_override"] != false {
		t.Fatalf("expected is_override=false, got %v", data["is_override"])
	}
}

func TestGetCommissionResolution_InternalError(t *testing.T) {
	svc := &stubCommissionService{err: errors.New("db failure")}
	router := newCommissionHandlerRouter(svc)

	w := performRequest(router, "GET", "/api/v1/admin/trainers/"+validTrainerID()+"/commission/resolve", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}
