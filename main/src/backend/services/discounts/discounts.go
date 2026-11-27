// Package discounts manages discount codes (owner request, 2026-11-24):
// ADMIN_1 or ADMIN_2 create a code with a percentage off and how long it can
// be used (at most 30 days, or a number of hours when it is 24 hours or
// less); a buyer enters it at checkout and the server applies it to the
// pending purchase before the payment opens.
package discounts

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

const (
	// MaxDuration is the longest a code can be used.
	MaxDuration = 30 * 24 * time.Hour
	// MaxPercentOff keeps every paid plan paid.
	MaxPercentOff = 90
)

var (
	ErrInvalidInput  = errors.New("invalid discount code")
	ErrCodeTaken     = errors.New("discount code already exists")
	ErrNotFound      = errors.New("discount code not found")
	ErrExpired       = errors.New("discount code expired or disabled")
	ErrNotApplicable = errors.New("discount code does not apply to this plan")
	ErrUsedUp        = errors.New("discount code has no uses left")
	ErrAlreadyUsed   = errors.New("discount code already used by this person")
	ErrLocked        = errors.New("the payment already opened; the price no longer changes")
)

var codePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{2,31}$`)

var scopes = map[string]bool{
	models.DiscountAppliesToAll: true, models.DiscountAppliesToPremiumLevel1: true,
	models.DiscountAppliesToPremiumLevel2: true, models.DiscountAppliesToTrainerPlans: true,
}

// NormalizeCode upper-cases and trims a code.
func NormalizeCode(code string) string { return strings.ToUpper(strings.TrimSpace(code)) }

// CreateInput is a new code. DurationHours is how long it can be used from
// now: 1 to 720 hours (30 days).
type CreateInput struct {
	Code          string
	PercentOff    int
	AppliesTo     string
	DurationHours int
	MaxUses       *int
}

type Service struct {
	repo repositories.DiscountCodeRepository
	now  func() time.Time
}

func New(repo repositories.DiscountCodeRepository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Create(ctx context.Context, adminID string, input CreateInput) (*models.DiscountCode, error) {
	code := NormalizeCode(input.Code)
	appliesTo := strings.TrimSpace(input.AppliesTo)
	if appliesTo == "" {
		appliesTo = models.DiscountAppliesToAll
	}
	duration := time.Duration(input.DurationHours) * time.Hour
	if !codePattern.MatchString(code) || input.PercentOff < 1 || input.PercentOff > MaxPercentOff || !scopes[appliesTo] ||
		duration <= 0 || duration > MaxDuration || (input.MaxUses != nil && *input.MaxUses < 1) || strings.TrimSpace(adminID) == "" {
		return nil, ErrInvalidInput
	}
	now := s.now()
	created := &models.DiscountCode{
		Code: code, PercentOff: input.PercentOff, AppliesTo: appliesTo,
		StartsAt: now, ExpiresAt: now.Add(duration), MaxUses: input.MaxUses, CreatedBy: adminID,
	}
	if err := s.repo.Create(ctx, created); err != nil {
		if errors.Is(err, repositories.ErrDiscountCodeTaken) {
			return nil, ErrCodeTaken
		}
		return nil, err
	}
	return created, nil
}

func (s *Service) List(ctx context.Context) ([]repositories.DiscountCodeView, error) {
	return s.repo.List(ctx)
}

func (s *Service) Disable(ctx context.Context, id string) error {
	if err := s.repo.Disable(ctx, id, s.now()); err != nil {
		if errors.Is(err, repositories.ErrDiscountCodeNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// applies reports whether the code covers the plan.
func applies(code *models.DiscountCode, target *repositories.DiscountTarget) bool {
	switch code.AppliesTo {
	case models.DiscountAppliesToPremiumLevel1:
		return target.ProductType == models.ProgramProductTypePremiumLevel1
	case models.DiscountAppliesToPremiumLevel2:
		return target.ProductType == models.ProgramProductTypePremiumLevel2
	case models.DiscountAppliesToTrainerPlans:
		return target.TrainerID != nil && *target.TrainerID != ""
	}
	return true
}

// Check validates a code for a person and a plan.
func (s *Service) Check(ctx context.Context, userID, rawCode string, target *repositories.DiscountTarget) (*models.DiscountCode, error) {
	code, err := s.repo.FindByCode(ctx, NormalizeCode(rawCode))
	if err != nil {
		if errors.Is(err, repositories.ErrDiscountCodeNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	now := s.now()
	if code.DisabledAt != nil || now.Before(code.StartsAt) || !now.Before(code.ExpiresAt) {
		return nil, ErrExpired
	}
	if !applies(code, target) {
		return nil, ErrNotApplicable
	}
	if code.MaxUses != nil {
		uses, err := s.repo.CountUses(ctx, code.ID)
		if err != nil {
			return nil, err
		}
		if uses >= int64(*code.MaxUses) {
			return nil, ErrUsedUp
		}
	}
	used, err := s.repo.UsedBy(ctx, code.ID, userID)
	if err != nil {
		return nil, err
	}
	if used {
		return nil, ErrAlreadyUsed
	}
	return code, nil
}

// Preview is what a code takes off a price.
type Preview struct {
	Code          string
	PercentOff    int
	DiscountMinor int64
	TotalMinor    int64
	ExpiresAt     time.Time
}

// PreviewFor checks a code against a plan before buying it.
func (s *Service) PreviewFor(ctx context.Context, userID, rawCode string, target *repositories.DiscountTarget, priceMinor int64) (*Preview, error) {
	code, err := s.Check(ctx, userID, rawCode, target)
	if err != nil {
		return nil, err
	}
	discount := priceMinor * int64(code.PercentOff) / 100
	return &Preview{Code: code.Code, PercentOff: code.PercentOff, DiscountMinor: discount, TotalMinor: priceMinor - discount, ExpiresAt: code.ExpiresAt}, nil
}

// ApplyToPurchase sets the buyer's code on their pending purchase ("" removes
// it). It is called when the payment is about to open.
func (s *Service) ApplyToPurchase(ctx context.Context, userID, purchaseID, rawCode string) error {
	var code *models.DiscountCode
	if strings.TrimSpace(rawCode) != "" {
		target, err := s.repo.Target(ctx, purchaseID, userID)
		if err != nil {
			return err
		}
		code, err = s.Check(ctx, userID, rawCode, target)
		if err != nil {
			return err
		}
	}
	if _, err := s.repo.Apply(ctx, purchaseID, userID, code); err != nil {
		if errors.Is(err, repositories.ErrDiscountLocked) {
			return ErrLocked
		}
		return err
	}
	return nil
}
