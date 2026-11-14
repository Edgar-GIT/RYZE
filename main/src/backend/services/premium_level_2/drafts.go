package premium_level_2

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// The Premium Level 2 questionnaire is answered before payment (owner request,
// 2026-11-14): the answers are kept as a per-offer draft, a purchase needs a
// complete draft, and the completed purchase turns the draft into the
// request's first intake.

var (
	// ErrOfferNotFound indicates the offer is not a published Premium Level 2
	// offer.
	ErrOfferNotFound = errors.New("premium level 2 offer not found")
	// ErrIntakeDraftRequired indicates a purchase was attempted before the
	// questionnaire was complete.
	ErrIntakeDraftRequired = errors.New("the premium level 2 questionnaire must be completed before payment")
)

// DraftRepository stores the pre-purchase questionnaires.
type DraftRepository interface {
	FindDraft(ctx context.Context, userID, programID string) (*models.PremiumLevel2IntakeDraft, error)
	SaveDraft(ctx context.Context, draft *models.PremiumLevel2IntakeDraft) error
	DeleteDraft(ctx context.Context, userID, programID string) error
}

// DraftStatus tells the client what is still missing.
type DraftStatus struct {
	Complete bool
	Missing  []string
	Invalid  map[string]string
}

// DraftService is the pre-purchase questionnaire surface.
type DraftService interface {
	GetIntakeDraft(ctx context.Context, userID, programID string) (*models.PremiumLevel2IntakeDraft, DraftStatus, error)
	SaveIntakeDraft(ctx context.Context, userID, programID string, answers []byte) (*models.PremiumLevel2IntakeDraft, DraftStatus, error)
	// Satisfied is the checkout prerequisite: a Premium Level 2 offer can only
	// be bought with a complete questionnaire.
	Satisfied(ctx context.Context, userID, programID, productType, nutritionMode string) error
}

// WithDrafts enables the pre-purchase questionnaire.
func WithDrafts(drafts DraftRepository) ServiceOption {
	return func(s *service) { s.drafts = drafts }
}

// Drafts exposes the pre-purchase questionnaire surface of a service built
// with WithDrafts.
func Drafts(svc Service) (DraftService, bool) {
	concrete, ok := svc.(*service)
	if !ok || concrete.drafts == nil {
		return nil, false
	}
	return concrete, true
}

func (s *service) publishedOffer(ctx context.Context, programID string) error {
	if strings.TrimSpace(programID) == "" {
		return ErrOfferNotFound
	}
	program, err := s.programs.FindByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return ErrOfferNotFound
		}
		return fmt.Errorf("failed to load premium level 2 offer: %w", err)
	}
	if program.ProductType != models.ProgramProductTypePremiumLevel2 || program.Status != models.ProgramStatusPublished {
		return ErrOfferNotFound
	}
	return nil
}

func statusOf(draft *models.PremiumLevel2IntakeDraft) DraftStatus {
	if draft == nil {
		_, invalid, missing := NormalizePartialAnswers(nil)
		return DraftStatus{Missing: missing, Invalid: invalid}
	}
	_, invalid, missing := NormalizePartialAnswers(draft.Answers)
	return DraftStatus{Complete: len(invalid) == 0 && len(missing) == 0, Missing: missing, Invalid: invalid}
}

func (s *service) GetIntakeDraft(ctx context.Context, userID, programID string) (*models.PremiumLevel2IntakeDraft, DraftStatus, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, DraftStatus{}, ErrRequestNotFound
	}
	if err := s.publishedOffer(ctx, programID); err != nil {
		return nil, DraftStatus{}, err
	}
	draft, err := s.drafts.FindDraft(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrPremiumLevel2DraftNotFound) {
			return nil, statusOf(nil), nil
		}
		return nil, DraftStatus{}, err
	}
	return draft, statusOf(draft), nil
}

func (s *service) SaveIntakeDraft(ctx context.Context, userID, programID string, answers []byte) (*models.PremiumLevel2IntakeDraft, DraftStatus, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, DraftStatus{}, ErrRequestNotFound
	}
	if err := s.publishedOffer(ctx, programID); err != nil {
		return nil, DraftStatus{}, err
	}
	normalized, invalid, missing := NormalizePartialAnswers(answers)
	status := DraftStatus{Complete: len(invalid) == 0 && len(missing) == 0, Missing: missing, Invalid: invalid}
	// Only valid answers are ever stored; invalid ones come back to the client.
	draft := &models.PremiumLevel2IntakeDraft{
		UserID:    userID,
		ProgramID: programID,
		Answers:   normalized,
		Complete:  status.Complete,
	}
	if err := s.drafts.SaveDraft(ctx, draft); err != nil {
		return nil, DraftStatus{}, err
	}
	saved, err := s.drafts.FindDraft(ctx, userID, programID)
	if err != nil {
		return nil, DraftStatus{}, err
	}
	return saved, status, nil
}

func (s *service) Satisfied(ctx context.Context, userID, programID, productType, _ string) error {
	if productType != models.ProgramProductTypePremiumLevel2 || s.drafts == nil {
		return nil
	}
	draft, err := s.drafts.FindDraft(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrPremiumLevel2DraftNotFound) {
			return ErrIntakeDraftRequired
		}
		return fmt.Errorf("failed to verify premium level 2 questionnaire: %w", err)
	}
	if !statusOf(draft).Complete {
		return ErrIntakeDraftRequired
	}
	return nil
}

// submitDraft turns a complete pre-purchase questionnaire into the first
// intake of the request created by the purchase. It is idempotent: a request
// that already has an intake is left as it is.
func (s *service) submitDraft(ctx context.Context, request *models.PremiumLevel2Request) {
	if s.drafts == nil || request == nil || request.Status != models.PremiumLevel2StatusAwaitingIntake {
		return
	}
	draft, err := s.drafts.FindDraft(ctx, request.UserID, request.ProgramID)
	if err != nil {
		return
	}
	normalized, fields := NormalizeAnswers(draft.Answers)
	if len(fields) > 0 {
		return
	}
	if _, intake, err := s.requests.CreateIntake(ctx, request.ID, normalized); err == nil {
		_ = s.drafts.DeleteDraft(ctx, request.UserID, request.ProgramID)
		s.notifyIntakeReceived(ctx, request.ID, intake)
	}
}
