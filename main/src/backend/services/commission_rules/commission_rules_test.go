package commission_rules_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/commission_rules"
)

// --- stubs ---

type stubCommissionRuleRepository struct {
	rule *models.CommissionRule
	err  error
}

func (s *stubCommissionRuleRepository) FindActiveOverride(_ context.Context, _ string) (*models.CommissionRule, error) {
	return s.rule, s.err
}

func (s *stubCommissionRuleRepository) UpsertOverride(_ context.Context, rule *models.CommissionRule) error {
	s.rule = rule
	return s.err
}

func (s *stubCommissionRuleRepository) SoftDelete(_ context.Context, _ string) error {
	return s.err
}

type stubTrainerRepository struct {
	trainer *models.Trainer
	err     error
}

func (s *stubTrainerRepository) FindByID(_ context.Context, _ string) (*models.Trainer, error) {
	return s.trainer, s.err
}

// --- helpers ---

func defaultConfig() config.CommissionConfig {
	return config.CommissionConfig{DefaultPlatformCommissionBPS: 2000}
}

func validTrainerID() string {
	return "11111111-1111-1111-1111-111111111111"
}

// --- GetCommissionRule ---

func TestGetCommissionRule_Success(t *testing.T) {
	trainerID := validTrainerID()
	expectedRule := &models.CommissionRule{
		ID:            "22222222-2222-2222-2222-222222222222",
		TrainerID:     trainerID,
		CommissionBPS: 1500,
	}

	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{rule: expectedRule},
		&stubTrainerRepository{trainer: &models.Trainer{ID: trainerID}},
		defaultConfig(),
	)

	rule, err := svc.GetCommissionRule(context.Background(), trainerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.CommissionBPS != 1500 {
		t.Fatalf("expected commission_bps=1500, got %d", rule.CommissionBPS)
	}
	if rule.TrainerID != trainerID {
		t.Fatalf("expected trainer_id=%s, got %s", trainerID, rule.TrainerID)
	}
}

func TestGetCommissionRule_EmptyTrainerID(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	_, err := svc.GetCommissionRule(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty trainer id")
	}
	if !errors.Is(err, commission_rules.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetCommissionRule_NotFound(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{err: repositories.ErrCommissionRuleNotFound},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	_, err := svc.GetCommissionRule(context.Background(), validTrainerID())
	if err == nil {
		t.Fatal("expected error for missing rule")
	}
	if !errors.Is(err, commission_rules.ErrCommissionRuleNotFound) {
		t.Fatalf("expected ErrCommissionRuleNotFound, got %v", err)
	}
}

func TestGetCommissionRule_RepositoryError(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{err: errors.New("db failure")},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	_, err := svc.GetCommissionRule(context.Background(), validTrainerID())
	if err == nil {
		t.Fatal("expected error for repository failure")
	}
}

// --- UpsertCommissionRule ---

func TestUpsertCommissionRule_Success(t *testing.T) {
	trainerID := validTrainerID()
	repo := &stubCommissionRuleRepository{}
	trainerRepo := &stubTrainerRepository{trainer: &models.Trainer{ID: trainerID}}

	svc := commission_rules.NewService(repo, trainerRepo, defaultConfig())

	rule, err := svc.UpsertCommissionRule(context.Background(), trainerID, 1500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.CommissionBPS != 1500 {
		t.Fatalf("expected commission_bps=1500, got %d", rule.CommissionBPS)
	}
	if rule.TrainerID != trainerID {
		t.Fatalf("expected trainer_id=%s, got %s", trainerID, rule.TrainerID)
	}
}

func TestUpsertCommissionRule_EmptyTrainerID(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	_, err := svc.UpsertCommissionRule(context.Background(), "", 1500)
	if err == nil {
		t.Fatal("expected error for empty trainer id")
	}
	if !errors.Is(err, commission_rules.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpsertCommissionRule_InvalidBPS(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	_, err := svc.UpsertCommissionRule(context.Background(), validTrainerID(), 10001)
	if err == nil {
		t.Fatal("expected error for invalid bps")
	}
	if !errors.Is(err, commission_rules.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpsertCommissionRule_TrainerNotFound(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{err: repositories.ErrTrainerNotFound},
		defaultConfig(),
	)

	_, err := svc.UpsertCommissionRule(context.Background(), validTrainerID(), 1500)
	if err == nil {
		t.Fatal("expected error for missing trainer")
	}
	if !errors.Is(err, commission_rules.ErrTrainerNotFound) {
		t.Fatalf("expected ErrTrainerNotFound, got %v", err)
	}
}

func TestUpsertCommissionRule_MaxBPS(t *testing.T) {
	trainerID := validTrainerID()
	repo := &stubCommissionRuleRepository{}
	trainerRepo := &stubTrainerRepository{trainer: &models.Trainer{ID: trainerID}}

	svc := commission_rules.NewService(repo, trainerRepo, defaultConfig())

	rule, err := svc.UpsertCommissionRule(context.Background(), trainerID, 10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.CommissionBPS != 10000 {
		t.Fatalf("expected commission_bps=10000, got %d", rule.CommissionBPS)
	}
}

func TestUpsertCommissionRule_ZeroBPS(t *testing.T) {
	trainerID := validTrainerID()
	repo := &stubCommissionRuleRepository{}
	trainerRepo := &stubTrainerRepository{trainer: &models.Trainer{ID: trainerID}}

	svc := commission_rules.NewService(repo, trainerRepo, defaultConfig())

	rule, err := svc.UpsertCommissionRule(context.Background(), trainerID, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.CommissionBPS != 0 {
		t.Fatalf("expected commission_bps=0, got %d", rule.CommissionBPS)
	}
}

// --- DeleteCommissionRule ---

func TestDeleteCommissionRule_Success(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	err := svc.DeleteCommissionRule(context.Background(), validTrainerID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteCommissionRule_EmptyTrainerID(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	err := svc.DeleteCommissionRule(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty trainer id")
	}
	if !errors.Is(err, commission_rules.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDeleteCommissionRule_NotFound(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{err: repositories.ErrCommissionRuleNotFound},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	err := svc.DeleteCommissionRule(context.Background(), validTrainerID())
	if err == nil {
		t.Fatal("expected error for missing rule")
	}
	if !errors.Is(err, commission_rules.ErrCommissionRuleNotFound) {
		t.Fatalf("expected ErrCommissionRuleNotFound, got %v", err)
	}
}

// --- ResolveCommission ---

func TestResolveCommission_OverrideExists(t *testing.T) {
	trainerID := validTrainerID()
	expectedRule := &models.CommissionRule{
		ID:            "22222222-2222-2222-2222-222222222222",
		TrainerID:     trainerID,
		CommissionBPS: 1500,
	}

	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{rule: expectedRule},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	resolution, err := svc.ResolveCommission(context.Background(), trainerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution.CommissionBPS != 1500 {
		t.Fatalf("expected commission_bps=1500, got %d", resolution.CommissionBPS)
	}
	if !resolution.IsOverride {
		t.Fatal("expected is_override=true")
	}
}

func TestResolveCommission_NoOverride(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{err: repositories.ErrCommissionRuleNotFound},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	resolution, err := svc.ResolveCommission(context.Background(), validTrainerID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution.CommissionBPS != 2000 {
		t.Fatalf("expected default commission_bps=2000, got %d", resolution.CommissionBPS)
	}
	if resolution.IsOverride {
		t.Fatal("expected is_override=false")
	}
}

func TestResolveCommission_EmptyTrainerID(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	_, err := svc.ResolveCommission(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty trainer id")
	}
	if !errors.Is(err, commission_rules.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestResolveCommission_CustomDefault(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{err: repositories.ErrCommissionRuleNotFound},
		&stubTrainerRepository{},
		config.CommissionConfig{DefaultPlatformCommissionBPS: 3000},
	)

	resolution, err := svc.ResolveCommission(context.Background(), validTrainerID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution.CommissionBPS != 3000 {
		t.Fatalf("expected default commission_bps=3000, got %d", resolution.CommissionBPS)
	}
	if resolution.IsOverride {
		t.Fatal("expected is_override=false")
	}
}

// --- CalculateCommissionSplit ---

func TestCalculateCommissionSplit_StandardCase(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	resolution := commission_rules.CommissionResolution{CommissionBPS: 2000, IsOverride: false}
	calc := svc.CalculateCommissionSplit(10000, resolution)

	// 10000 * 2000 / 10000 = 2000
	if calc.PlatformAmount != 2000 {
		t.Fatalf("expected platform_amount=2000, got %d", calc.PlatformAmount)
	}
	if calc.TrainerAmount != 8000 {
		t.Fatalf("expected trainer_amount=8000, got %d", calc.TrainerAmount)
	}
}

func TestCalculateCommissionSplit_ZeroPrice(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	resolution := commission_rules.CommissionResolution{CommissionBPS: 2000, IsOverride: false}
	calc := svc.CalculateCommissionSplit(0, resolution)

	if calc.PlatformAmount != 0 {
		t.Fatalf("expected platform_amount=0, got %d", calc.PlatformAmount)
	}
	if calc.TrainerAmount != 0 {
		t.Fatalf("expected trainer_amount=0, got %d", calc.TrainerAmount)
	}
}

func TestCalculateCommissionSplit_RoundingBehavior(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	// 999 * 2000 / 10000 = 199.8 → floor = 199
	resolution := commission_rules.CommissionResolution{CommissionBPS: 2000, IsOverride: false}
	calc := svc.CalculateCommissionSplit(999, resolution)

	if calc.PlatformAmount != 199 {
		t.Fatalf("expected platform_amount=199, got %d", calc.PlatformAmount)
	}
	if calc.TrainerAmount != 800 {
		t.Fatalf("expected trainer_amount=800, got %d", calc.TrainerAmount)
	}
	// Verify the split sums to the original price
	if calc.PlatformAmount+calc.TrainerAmount != 999 {
		t.Fatalf("split does not sum to original price: %d + %d != 999", calc.PlatformAmount, calc.TrainerAmount)
	}
}

func TestCalculateCommissionSplit_OtherRoundingCases(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	cases := []struct {
		price            int64
		bps              uint32
		expectedPlatform int64
		expectedTrainer  int64
	}{
		{100, 2000, 20, 80},
		{1, 2000, 0, 1},
		{5000, 1500, 750, 4250},
		{10000, 10000, 10000, 0},
		{10000, 0, 0, 10000},
		{7777, 3333, 2592, 5185},
	}

	for _, tc := range cases {
		resolution := commission_rules.CommissionResolution{CommissionBPS: tc.bps, IsOverride: false}
		calc := svc.CalculateCommissionSplit(tc.price, resolution)

		if calc.PlatformAmount != tc.expectedPlatform {
			t.Errorf("price=%d bps=%d: expected platform_amount=%d, got %d",
				tc.price, tc.bps, tc.expectedPlatform, calc.PlatformAmount)
		}
		if calc.TrainerAmount != tc.expectedTrainer {
			t.Errorf("price=%d bps=%d: expected trainer_amount=%d, got %d",
				tc.price, tc.bps, tc.expectedTrainer, calc.TrainerAmount)
		}
		if calc.PlatformAmount+calc.TrainerAmount != tc.price {
			t.Errorf("price=%d bps=%d: split does not sum to price: %d + %d != %d",
				tc.price, tc.bps, calc.PlatformAmount, calc.TrainerAmount, tc.price)
		}
	}
}

func TestCalculateCommissionSplit_OverrideCase(t *testing.T) {
	svc := commission_rules.NewService(
		&stubCommissionRuleRepository{},
		&stubTrainerRepository{},
		defaultConfig(),
	)

	resolution := commission_rules.CommissionResolution{CommissionBPS: 5000, IsOverride: true}
	calc := svc.CalculateCommissionSplit(10000, resolution)

	// 10000 * 5000 / 10000 = 5000
	if calc.PlatformAmount != 5000 {
		t.Fatalf("expected platform_amount=5000, got %d", calc.PlatformAmount)
	}
	if calc.TrainerAmount != 5000 {
		t.Fatalf("expected trainer_amount=5000, got %d", calc.TrainerAmount)
	}
}
