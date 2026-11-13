package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"ryze/backend/api/auth"
	"ryze/backend/services/commission_rules"
)

func TestAdminSettingsReportsConfiguredCommerce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := auth.NewAdminSettingsHandler(199, 3000, func() []string { return []string{"stripe", "paypal"} })
	router := gin.New()
	router.GET("/settings", handler.GetCommerce)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/settings", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Data struct {
			Currency          string   `json:"currency"`
			MinPrice          int64    `json:"min_program_price_minor_units"`
			DefaultCommission uint32   `json:"default_platform_commission_bps"`
			MaxCommission     uint32   `json:"max_commission_bps"`
			PaymentMethods    []string `json:"payment_methods"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Currency != "EUR" || body.Data.MinPrice != 199 || body.Data.DefaultCommission != 3000 {
		t.Fatalf("unexpected settings: %+v", body.Data)
	}
	if body.Data.MaxCommission != commission_rules.MaxCommissionBPS {
		t.Fatalf("max commission = %d, want %d", body.Data.MaxCommission, commission_rules.MaxCommissionBPS)
	}
	if strings.Join(body.Data.PaymentMethods, ",") != "stripe,paypal" {
		t.Fatalf("payment methods = %v", body.Data.PaymentMethods)
	}
}

func TestAdminSettingsWithoutPaymentMethodsReturnsEmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/settings", auth.NewAdminSettingsHandler(100, 3000, nil).GetCommerce)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/settings", nil))
	if !strings.Contains(rec.Body.String(), `"payment_methods":[]`) {
		t.Fatalf("payment methods must be an empty list, got %s", rec.Body.String())
	}
}

// The system view must answer (not fail or hang) when the database is down,
// and must never carry configuration secrets.
func TestAdminSystemStatusWithUnreachableDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(postgres.Open("host=127.0.0.1 port=1 user=nobody dbname=none sslmode=disable connect_timeout=1"), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	handler := auth.NewAdminSystemHandler(db, auth.SystemFeatures{EmailDelivery: true, DefaultCommission: 3000, SessionSeconds: 900})
	router := gin.New()
	router.GET("/status", handler.Status)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Data struct {
			Database map[string]any `json:"database"`
			Features struct {
				EmailDelivery  bool     `json:"email_delivery"`
				PaymentMethods []string `json:"payment_methods"`
			} `json:"features"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Database["reachable"] != false {
		t.Fatalf("database reachable = %v, want false", body.Data.Database["reachable"])
	}
	if !body.Data.Features.EmailDelivery || body.Data.Features.PaymentMethods == nil {
		t.Fatalf("features not reported: %+v", body.Data.Features)
	}
	lower := strings.ToLower(rec.Body.String())
	for _, secret := range []string{"password", "secret", "dsn", "host=", "api_key"} {
		if strings.Contains(lower, secret) {
			t.Fatalf("system status leaks %q: %s", secret, rec.Body.String())
		}
	}
}
