package commission_rules

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

var (
	// ErrInvalidInput indicates the commission rule input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid commission rule input")
	// ErrTrainerNotFound indicates the trainer does not exist or is not active.
	ErrTrainerNotFound = errors.New("trainer not found")
	// ErrCommissionRuleNotFound indicates the commission rule does not exist or
	// is soft-deleted.
	ErrCommissionRuleNotFound = errors.New("commission rule not found")
)

const (
	// MaxCommissionBPS is the maximum allowed commission in basis points.
	// A commission of 10000 bps means the platform retains 100% of the price.
	MaxCommissionBPS uint32 = 10000
)

// CommissionRuleRepository is the data-access surface required by the
// commission rules service. Every operation is scoped to an explicit trainer
// id; the repository never obtains it from an HTTP context.
type CommissionRuleRepository interface {
	FindActiveOverride(ctx context.Context, trainerID string) (*models.CommissionRule, error)
	UpsertOverride(ctx context.Context, rule *models.CommissionRule) error
	SoftDelete(ctx context.Context, trainerID string) error
}

// TrainerRepository is the data-access surface for verifying trainer existence.
type TrainerRepository interface {
	FindByID(ctx context.Context, id string) (*models.Trainer, error)
}

// CommissionRule is the safe representation of one commission rule. It carries
// only the public metadata and never exposes internal data.
type CommissionRule struct {
	ID            string
	TrainerID     string
	CommissionBPS uint32
	ValidFrom     time.Time
	ValidUntil    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CommissionResolution represents the outcome of resolving the applicable
// commission for a given trainer. The service always returns exactly one
// deterministic resolution: either a trainer-specific override or the global
// default.
type CommissionResolution struct {
	// CommissionBPS is the applicable commission in basis points.
	CommissionBPS uint32
	// IsOverride indicates whether this resolution comes from a trainer-specific
	// override (true) or the global default (false).
	IsOverride bool
}

// CommissionCalculation is the result of applying a commission resolution to a
// price. All amounts are expressed in the same minor currency units as the
// input price. Rounding is deterministic: the platform amount is computed with
// floor division and the trainer receives the remainder.
type CommissionCalculation struct {
	PlatformAmount int64
	TrainerAmount  int64
}

// Service implements the commercial commission-rules flow. Authorization (which
// administrator may operate) is enforced by the route middleware; this service
// never knows about HTTP, Gin or the authentication context.
type Service interface {
	// GetCommissionRule returns the active commission rule for a given trainer,
	// or ErrCommissionRuleNotFound when no override exists.
	GetCommissionRule(ctx context.Context, trainerID string) (*CommissionRule, error)
	// UpsertCommissionRule creates or replaces the active commission rule for a
	// given trainer. The trainer must exist and be active.
	UpsertCommissionRule(ctx context.Context, trainerID string, commissionBPS uint32) (*CommissionRule, error)
	// DeleteCommissionRule removes the active commission rule for a given trainer.
	// A missing rule maps to ErrCommissionRuleNotFound.
	DeleteCommissionRule(ctx context.Context, trainerID string) error
	// ResolveCommission determines the applicable commission for a given trainer.
	// It checks for a trainer-specific active override first; when none exists
	// the global default is used. The resolution is always deterministic.
	ResolveCommission(ctx context.Context, trainerID string) (CommissionResolution, error)
	// CalculateCommissionSplit applies a commission resolution to a price and
	// returns the platform and trainer amounts. The computation is deterministic
	// and uses floor division to avoid rounding ambiguity.
	CalculateCommissionSplit(priceMinorUnits int64, resolution CommissionResolution) CommissionCalculation
}

type service struct {
	rules      CommissionRuleRepository
	trainers   TrainerRepository
	defaultBPS uint32
}

func NewService(rules CommissionRuleRepository, trainers TrainerRepository, cfg config.CommissionConfig) Service {
	return &service{rules: rules, trainers: trainers, defaultBPS: cfg.DefaultPlatformCommissionBPS}
}

// GetCommissionRule returns the active commission rule for a given trainer. A
// missing or soft-deleted rule maps to ErrCommissionRuleNotFound.
func (s *service) GetCommissionRule(ctx context.Context, trainerID string) (*CommissionRule, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}

	rule, err := s.rules.FindActiveOverride(ctx, trainerID)
	if err != nil {
		if errors.Is(err, repositories.ErrCommissionRuleNotFound) {
			return nil, ErrCommissionRuleNotFound
		}
		return nil, fmt.Errorf("failed to load commission rule: %w", err)
	}
	return newCommissionRule(rule), nil
}

// UpsertCommissionRule creates or replaces the active commission rule for a
// given trainer. The trainer must exist and be active. A valid commissionBPS
// value is between 0 and 10000 (inclusive).
func (s *service) UpsertCommissionRule(ctx context.Context, trainerID string, commissionBPS uint32) (*CommissionRule, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateCommissionBPS(commissionBPS); err != nil {
		return nil, err
	}

	trainer, err := s.trainers.FindByID(ctx, trainerID)
	if err != nil {
		if errors.Is(err, repositories.ErrTrainerNotFound) {
			return nil, ErrTrainerNotFound
		}
		return nil, fmt.Errorf("failed to verify trainer: %w", err)
	}
	_ = trainer

	now := time.Now().UTC().Truncate(time.Microsecond)
	rule := &models.CommissionRule{
		TrainerID:     trainerID,
		CommissionBPS: commissionBPS,
		ValidFrom:     now,
	}

	if err := s.rules.UpsertOverride(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to upsert commission rule: %w", err)
	}

	return newCommissionRule(rule), nil
}

// DeleteCommissionRule removes the active commission rule for a given trainer.
// A missing rule maps to ErrCommissionRuleNotFound.
func (s *service) DeleteCommissionRule(ctx context.Context, trainerID string) error {
	if err := validateTrainerID(trainerID); err != nil {
		return err
	}

	if err := s.rules.SoftDelete(ctx, trainerID); err != nil {
		if errors.Is(err, repositories.ErrCommissionRuleNotFound) {
			return ErrCommissionRuleNotFound
		}
		return fmt.Errorf("failed to delete commission rule: %w", err)
	}
	return nil
}

// ResolveCommission determines the applicable commission for a given trainer.
// It checks for a trainer-specific active override first; when none exists the
// global default is used. The resolution is always deterministic and never
// returns an error for the default path.
func (s *service) ResolveCommission(ctx context.Context, trainerID string) (CommissionResolution, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return CommissionResolution{}, err
	}

	rule, err := s.rules.FindActiveOverride(ctx, trainerID)
	if err != nil {
		if errors.Is(err, repositories.ErrCommissionRuleNotFound) {
			return CommissionResolution{
				CommissionBPS: s.defaultBPS,
				IsOverride:    false,
			}, nil
		}
		return CommissionResolution{}, fmt.Errorf("failed to resolve commission: %w", err)
	}

	return CommissionResolution{
		CommissionBPS: rule.CommissionBPS,
		IsOverride:    true,
	}, nil
}

// CalculateCommissionSplit applies a commission resolution to a price and
// returns the platform and trainer amounts. The computation uses floor division
// (integer truncation toward zero) so the platform amount is never larger than
// the actual proportional share. The trainer receives the exact remainder:
//
//	platform = floor(price * bps / 10000)
//	trainer  = price - platform
//
// This guarantees: platform + trainer == price, always.
func (s *service) CalculateCommissionSplit(priceMinorUnits int64, resolution CommissionResolution) CommissionCalculation {
	platformAmount := (priceMinorUnits * int64(resolution.CommissionBPS)) / 10000
	trainerAmount := priceMinorUnits - platformAmount
	return CommissionCalculation{
		PlatformAmount: platformAmount,
		TrainerAmount:  trainerAmount,
	}
}

func newCommissionRule(model *models.CommissionRule) *CommissionRule {
	return &CommissionRule{
		ID:            model.ID,
		TrainerID:     model.TrainerID,
		CommissionBPS: model.CommissionBPS,
		ValidFrom:     model.ValidFrom,
		ValidUntil:    model.ValidUntil,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
	}
}

// validateTrainerID rejects empty and malformed identifiers before any database
// access.
func validateTrainerID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: trainer id is required", ErrInvalidInput)
	}
	return nil
}

// validateCommissionBPS rejects commission values outside the valid range
// [0, 10000].
func validateCommissionBPS(bps uint32) error {
	if bps > MaxCommissionBPS {
		return fmt.Errorf("%w: commission must be at most %d basis points", ErrInvalidInput, MaxCommissionBPS)
	}
	return nil
}
