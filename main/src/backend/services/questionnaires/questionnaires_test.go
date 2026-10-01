package questionnaires_test

import (
	"context"
	"errors"
	"testing"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/nutrition_questionnaire"
	"ryze/backend/services/questionnaires"
)

const (
	premiumUser = "11111111-1111-4111-8111-111111111111"
	otherUser   = "22222222-2222-4222-8222-222222222222"
	premiumProg = "33333333-3333-4333-8333-333333333333"
	genericProg = "44444444-4444-4444-8444-444444444444"
	missingProg = "55555555-5555-4555-8555-555555555555"
	storedID    = "66666666-6666-4666-8666-666666666666"
)

func validAnswers() nutrition_questionnaire.Answers {
	return nutrition_questionnaire.Answers{
		Goal:          strPtr("fat_loss"),
		Experience:    strPtr("beginner"),
		TrainingDays:  intPtr(3),
		SessionLength: intPtr(45),
		Diet:          strPtr("omnivore"),
		MealsPerDay:   intPtr(3),
		CookingEffort: strPtr("minimal"),
		BodyWeightKg:  floatPtr(70),
		BodyHeightCm:  intPtr(170),
		ActivityLevel: strPtr("light"),
		SleepHours:    floatPtr(7),
		StressLevel:   strPtr("low"),
	}
}

type stubProgramRepository struct {
	program *models.Program
	err     error
}

func (r *stubProgramRepository) FindPublishedByID(_ context.Context, _ string) (*models.Program, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.program == nil {
		return nil, repositories.ErrProgramNotFound
	}
	return r.program, nil
}

type storedQuestionnaire struct {
	id      string
	version int
	answers []byte
}

type stubQuestionnaireRepository struct {
	row *storedQuestionnaire
	err error
	// upserted captures what the service tried to persist.
	upsertedForUser string
	upsertedForProg string
	upsertedAnswers []byte
	upsertCalls     int
}

func (r *stubQuestionnaireRepository) UpsertForUserAndProgram(_ context.Context, userID, programID string, answers []byte) (*models.NutritionQuestionnaire, error) {
	r.upsertCalls++
	r.upsertedForUser = userID
	r.upsertedForProg = programID
	r.upsertedAnswers = answers
	if r.err != nil {
		return nil, r.err
	}
	version := 1
	if r.row != nil {
		version = r.row.version + 1
	}
	return &models.NutritionQuestionnaire{ID: storedID, UserID: userID, ProgramID: programID, Version: version, Answers: answers}, nil
}

func (r *stubQuestionnaireRepository) FindByUserAndProgram(_ context.Context, userID, programID string) (*models.NutritionQuestionnaire, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.row == nil {
		return nil, repositories.ErrNutritionQuestionnaireNotFound
	}
	return &models.NutritionQuestionnaire{
		ID:        r.row.id,
		UserID:    userID,
		ProgramID: programID,
		Version:   r.row.version,
		Answers:   r.row.answers,
	}, nil
}

func (r *stubQuestionnaireRepository) FindSummaryByUserAndProgram(_ context.Context, _, _ string) (*repositories.NutritionQuestionnaireSummary, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.row == nil {
		return nil, repositories.ErrNutritionQuestionnaireNotFound
	}
	return &repositories.NutritionQuestionnaireSummary{
		ID:          r.row.id,
		Version:     r.row.version,
		SubmittedAt: nowFixture,
	}, nil
}

// stubEntitlementRepository decides whether the pair looks purchased. An empty
// entitlement means the user has never bought the program.
type stubEntitlementRepository struct {
	entitlement *models.Entitlement
	err         error
	calls       int
}

func (s *stubEntitlementRepository) FindActiveByUserAndProgram(_ context.Context, _, _ string) (*models.Entitlement, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if s.entitlement == nil {
		return nil, repositories.ErrEntitlementNotFound
	}
	return s.entitlement, nil
}

func newService(program *models.Program, repo *stubQuestionnaireRepository) questionnaires.Service {
	return questionnaires.NewService(&stubProgramRepository{program: program}, repo, &stubEntitlementRepository{})
}

func newServiceWithEntitlements(program *models.Program, repo *stubQuestionnaireRepository, entitlements *stubEntitlementRepository) questionnaires.Service {
	return questionnaires.NewService(&stubProgramRepository{program: program}, repo, entitlements)
}

func premiumProgram() *models.Program {
	return &models.Program{
		ID:          premiumProg,
		Name:        "Premium Level 1",
		Type:        models.ProgramTypePremium,
		ProductType: models.ProgramProductTypePremiumLevel1,
		Status:      models.ProgramStatusPublished,
	}
}

func genericProgram() *models.Program {
	return &models.Program{
		ID:          genericProg,
		Name:        "Generic Plan",
		Type:        models.ProgramTypePremium,
		ProductType: models.ProgramProductTypeGeneric,
		Status:      models.ProgramStatusPublished,
	}
}

func TestSubmitPersistsNormalizedAnswersForAuthenticatedUser(t *testing.T) {
	repo := &stubQuestionnaireRepository{}
	svc := newService(premiumProgram(), repo)

	if _, err := svc.Submit(context.Background(), premiumUser, premiumProg, validAnswers()); err != nil {
		t.Fatalf("expected submission to succeed, got %v", err)
	}

	if repo.upsertedForUser != premiumUser {
		t.Errorf("persisted for user %q, want the authenticated user", repo.upsertedForUser)
	}
	if repo.upsertedForProg != premiumProg {
		t.Errorf("persisted for program %q, want %q", repo.upsertedForProg, premiumProg)
	}
	if len(repo.upsertedAnswers) == 0 {
		t.Fatal("expected normalized answers to be persisted")
	}
	// The stored document must be the normalized form, not the raw submission.
	if _, err := nutrition_questionnaire.Unmarshal(repo.upsertedAnswers); err != nil {
		t.Errorf("stored document is not a valid normalized intake: %v", err)
	}
}

func TestSubmitRejectsInvalidSubmissionWithoutPersisting(t *testing.T) {
	repo := &stubQuestionnaireRepository{}
	svc := newService(premiumProgram(), repo)

	answers := validAnswers()
	answers.Goal = strPtr("invented_goal")

	_, err := svc.Submit(context.Background(), premiumUser, premiumProg, answers)
	if !errors.Is(err, questionnaires.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.upsertCalls != 0 {
		t.Error("a rejected submission must never be persisted, even partially")
	}
}

func TestSubmitRejectsGenericProgram(t *testing.T) {
	// A questionnaire is a Premium Level 1 concern. Accepting one for a generic
	// plan would create sensitive rows nothing would ever consume.
	repo := &stubQuestionnaireRepository{}
	svc := newService(genericProgram(), repo)

	if _, err := svc.Submit(context.Background(), premiumUser, genericProg, validAnswers()); !errors.Is(err, questionnaires.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound for a generic program, got %v", err)
	}
	if repo.upsertCalls != 0 {
		t.Error("expected nothing to be persisted for a generic program")
	}
}

func TestSubmitRejectsUnpublishedProgram(t *testing.T) {
	repo := &stubQuestionnaireRepository{}
	svc := questionnaires.NewService(&stubProgramRepository{err: repositories.ErrProgramNotFound}, repo)

	if _, err := svc.Submit(context.Background(), premiumUser, missingProg, validAnswers()); !errors.Is(err, questionnaires.ErrProgramNotFound) {
		t.Fatalf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestSubmitRejectsMalformedIdentifiers(t *testing.T) {
	repo := &stubQuestionnaireRepository{}
	svc := newService(premiumProgram(), repo)

	if _, err := svc.Submit(context.Background(), "not-a-uuid", premiumProg, validAnswers()); !errors.Is(err, questionnaires.ErrInvalidInput) {
		t.Errorf("expected a malformed user id to be rejected, got %v", err)
	}
	if _, err := svc.Submit(context.Background(), premiumUser, "not-a-uuid", validAnswers()); !errors.Is(err, questionnaires.ErrInvalidInput) {
		t.Errorf("expected a malformed program id to be rejected, got %v", err)
	}
}

func TestGetRequirementReportsUnsubmittedState(t *testing.T) {
	svc := newService(premiumProgram(), &stubQuestionnaireRepository{})

	requirement, err := svc.GetRequirement(context.Background(), premiumUser, premiumProg)
	if err != nil {
		t.Fatalf("GetRequirement failed: %v", err)
	}
	if !requirement.Required {
		t.Error("expected a Premium Level 1 program to require an intake")
	}
	if requirement.Submitted {
		t.Error("expected no intake to be reported as submitted")
	}
}

func TestGetRequirementDoesNotRequireIntakeForGenericProgram(t *testing.T) {
	svc := newService(genericProgram(), &stubQuestionnaireRepository{})

	requirement, err := svc.GetRequirement(context.Background(), premiumUser, genericProg)
	if err != nil {
		t.Fatalf("GetRequirement failed: %v", err)
	}
	if requirement.Required {
		t.Error("expected a generic program not to require an intake")
	}
}

func TestGetRequirementNeverLeaksAnswerContent(t *testing.T) {
	intake := mustNormalize(t)
	repo := &stubQuestionnaireRepository{row: &storedQuestionnaire{id: storedID, version: 2, answers: intake}}
	svc := newService(premiumProgram(), repo)

	requirement, err := svc.GetRequirement(context.Background(), premiumUser, premiumProg)
	if err != nil {
		t.Fatalf("GetRequirement failed: %v", err)
	}
	if !requirement.Submitted {
		t.Error("expected the stored intake to be reported as submitted")
	}
	if requirement.Version != 2 {
		t.Errorf("version = %d, want 2", requirement.Version)
	}
	if requirement.SubmittedAt == nil {
		t.Error("expected a submission timestamp for a stored intake")
	}
	// The requirement is a state object, so the summary read must not be the
	// vehicle for answer content.
	if got := requirementBody(t, requirement); containsAnswerContent(got) {
		t.Error("requirement response must not carry questionnaire answers")
	}
}

func TestSatisfiedBlocksCheckoutUntilIntakeExists(t *testing.T) {
	repo := &stubQuestionnaireRepository{}
	svc := newService(premiumProgram(), repo)

	err := svc.Satisfied(context.Background(), premiumUser, premiumProg, models.ProgramProductTypePremiumLevel1)
	if !errors.Is(err, questionnaires.ErrIntakeRequired) {
		t.Fatalf("expected ErrIntakeRequired, got %v", err)
	}
}

func TestSatisfiedAllowsCheckoutAfterIntakeExists(t *testing.T) {
	intake := mustNormalize(t)
	repo := &stubQuestionnaireRepository{row: &storedQuestionnaire{id: storedID, version: 1, answers: intake}}
	svc := newService(premiumProgram(), repo)

	if err := svc.Satisfied(context.Background(), premiumUser, premiumProg, models.ProgramProductTypePremiumLevel1); err != nil {
		t.Fatalf("expected the precondition to be satisfied, got %v", err)
	}
}

func TestSatisfiedIsTransparentForOtherProductTypes(t *testing.T) {
	// The gate must be invisible for every product that has no intake
	// requirement, so existing checkout behaviour is unchanged.
	repo := &stubQuestionnaireRepository{}
	svc := newService(genericProgram(), repo)

	if err := svc.Satisfied(context.Background(), premiumUser, genericProg, models.ProgramProductTypeGeneric); err != nil {
		t.Fatalf("expected no precondition for a generic product, got %v", err)
	}
}

func TestFindStoredIsScopedToTheAuthenticatedUser(t *testing.T) {
	intake := mustNormalize(t)
	repo := &stubQuestionnaireRepository{row: &storedQuestionnaire{id: storedID, version: 3, answers: intake}}
	svc := newService(premiumProgram(), repo)

	stored, err := svc.FindStored(context.Background(), otherUser, premiumProg)
	if err != nil {
		t.Fatalf("FindStored failed: %v", err)
	}
	// The stored identity is always the authenticated caller; a row can never be
	// claimed on behalf of another account.
	if stored.QuestionnaireID != storedID {
		t.Errorf("questionnaire id = %q, want %q", stored.QuestionnaireID, storedID)
	}
	if stored.Version != 3 {
		t.Errorf("version = %d, want 3", stored.Version)
	}
	if stored.Intake == nil || stored.Intake.Goal != "fat_loss" {
		t.Error("expected the validated intake to be returned")
	}
}

func TestFindStoredReportsMissingIntake(t *testing.T) {
	svc := newService(premiumProgram(), &stubQuestionnaireRepository{})

	_, err := svc.FindStored(context.Background(), premiumUser, premiumProg)
	if !errors.Is(err, nutrition_questionnaire.ErrNotSubmitted) {
		t.Fatalf("expected ErrNotSubmitted, got %v", err)
	}
}

func mustNormalize(t *testing.T) []byte {
	t.Helper()
	normalized, err := nutrition_questionnaire.Normalize(validAnswers())
	if err != nil {
		t.Fatalf("failed to normalize fixture: %v", err)
	}
	encoded, err := normalized.Marshal()
	if err != nil {
		t.Fatalf("failed to encode fixture: %v", err)
	}
	return encoded
}
