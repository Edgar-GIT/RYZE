package nutrition_assignment_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/nutrition_assignment"
	"ryze/backend/services/nutrition_questionnaire"
)

const (
	ownerUser    = "11111111-1111-4111-8111-111111111111"
	otherUser    = "22222222-2222-4222-8222-222222222222"
	premiumProg  = "33333333-3333-4333-8333-333333333333"
	genericProg  = "44444444-4444-4444-8444-444444444444"
	questionnID  = "55555555-5555-4555-8555-555555555555"
	assignmentID = "66666666-6666-4666-8666-666666666666"
)

func validIntake(t *testing.T) []byte {
	t.Helper()

	normalized, err := nutrition_questionnaire.Normalize(nutrition_questionnaire.Answers{
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
	})
	if err != nil {
		t.Fatalf("failed to normalize fixture: %v", err)
	}
	encoded, err := normalized.Marshal()
	if err != nil {
		t.Fatalf("failed to encode fixture: %v", err)
	}
	return encoded
}

type stubPrograms struct {
	program *models.Program
	err     error
}

func (r *stubPrograms) FindPublishedByID(_ context.Context, _ string) (*models.Program, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.program == nil {
		return nil, repositories.ErrProgramNotFound
	}
	return r.program, nil
}

type stubEntitlements struct {
	// owned maps user id to whether the account holds an active entitlement.
	owned map[string]bool
}

func (r *stubEntitlements) FindActiveByUserAndProgram(_ context.Context, userID, _ string) (*models.Entitlement, error) {
	if r.owned[userID] {
		return &models.Entitlement{ID: "entitlement", UserID: userID}, nil
	}
	return nil, repositories.ErrEntitlementNotFound
}

type stubQuestionnaires struct {
	stored *nutrition_questionnaire.StoredIntake
	err    error
}

func (r *stubQuestionnaires) FindStored(_ context.Context, _, _ string) (*nutrition_questionnaire.StoredIntake, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.stored == nil {
		return nil, nutrition_questionnaire.ErrNotSubmitted
	}
	return r.stored, nil
}

type stubAssignments struct {
	assignment *models.NutritionAssignment
	// ensureCalls counts provisioning attempts, so idempotency is observable.
	ensureCalls int
	markedProc  int
	markedDone  int
	markedFail  int
	// failOnComplete simulates a generation run that cannot be committed.
	failOnComplete error
}

func (r *stubAssignments) EnsurePendingForUserAndProgram(_ context.Context, userID, programID, questionnaireID string, version int) (*models.NutritionAssignment, bool, error) {
	r.ensureCalls++
	if r.assignment != nil {
		return r.assignment, false, nil
	}
	created := &models.NutritionAssignment{
		ID:                   assignmentID,
		UserID:               userID,
		ProgramID:            programID,
		QuestionnaireID:      questionnaireID,
		QuestionnaireVersion: version,
		Status:               models.NutritionAssignmentStatusPending,
		Version:              1,
	}
	r.assignment = created
	return created, true, nil
}

func (r *stubAssignments) FindByUserAndProgram(_ context.Context, _, _ string) (*models.NutritionAssignment, error) {
	if r.assignment == nil {
		return nil, repositories.ErrNutritionAssignmentNotFound
	}
	return r.assignment, nil
}

func (r *stubAssignments) MarkProcessing(context.Context, string, string) error {
	r.markedProc++
	if r.assignment == nil {
		return repositories.ErrNutritionAssignmentNotFound
	}
	if r.assignment.Status != models.NutritionAssignmentStatusPending &&
		r.assignment.Status != models.NutritionAssignmentStatusFailed {
		return repositories.ErrNutritionAssignmentNotRetryable
	}
	r.assignment.Status = models.NutritionAssignmentStatusProcessing
	return nil
}

func (r *stubAssignments) MarkCompleted(_ context.Context, _, _ string, configuration []byte, questionnaireVersion int) error {
	if r.failOnComplete != nil {
		return r.failOnComplete
	}
	r.markedDone++
	if r.assignment == nil {
		return repositories.ErrNutritionAssignmentNotFound
	}
	if r.assignment.QuestionnaireVersion != questionnaireVersion {
		return repositories.ErrNutritionAssignmentNotRetryable
	}
	if r.assignment.Status != models.NutritionAssignmentStatusProcessing {
		return repositories.ErrNutritionAssignmentNotRetryable
	}
	r.assignment.Status = models.NutritionAssignmentStatusCompleted
	r.assignment.Configuration = configuration
	r.assignment.Version++
	return nil
}

func (r *stubAssignments) MarkFailed(_ context.Context, _, _, _ string) error {
	r.markedFail++
	if r.assignment == nil {
		return repositories.ErrNutritionAssignmentNotFound
	}
	r.assignment.Status = models.NutritionAssignmentStatusFailed
	return nil
}

type failingGenerator struct{}

func (failingGenerator) Generate(*nutrition_questionnaire.Normalized) (*nutrition_assignment.GeneratedPlan, error) {
	return nil, errors.New("generator unavailable")
}

type harness struct {
	svc          nutrition_assignment.Service
	programs     *stubPrograms
	entitlements *stubEntitlements
	questionn    *stubQuestionnaires
	assignments  *stubAssignments
}

func newHarness(t *testing.T, program *models.Program, owned bool, generator nutrition_assignment.Generator) *harness {
	t.Helper()

	programs := &stubPrograms{program: program}
	entitlements := &stubEntitlements{owned: map[string]bool{}}
	if owned {
		entitlements.owned[ownerUser] = true
	}
	questionn := &stubQuestionnaires{stored: &nutrition_questionnaire.StoredIntake{
		QuestionnaireID: questionnID,
		Version:         1,
		Intake:          mustIntake(t, validIntake(t)),
	}}
	assignments := &stubAssignments{}
	if generator == nil {
		generator = nutrition_assignment.NewDeterministicGenerator()
	}

	return &harness{
		svc:          nutrition_assignment.NewService(programs, entitlements, questionn, assignments, generator),
		programs:     programs,
		entitlements: entitlements,
		questionn:    questionn,
		assignments:  assignments,
	}
}

func premiumProgram() *models.Program {
	return &models.Program{
		ID:          premiumProg,
		Type:        models.ProgramTypePremium,
		ProductType: models.ProgramProductTypePremiumLevel1,
		Status:      models.ProgramStatusPublished,
	}
}

func genericProgram() *models.Program {
	return &models.Program{
		ID:          genericProg,
		Type:        models.ProgramTypePremium,
		ProductType: models.ProgramProductTypeGeneric,
		Status:      models.ProgramStatusPublished,
	}
}

func TestRunGeneratesAndCompletesThePlan(t *testing.T) {
	h := newHarness(t, premiumProgram(), true, nil)

	status, err := h.svc.Run(context.Background(), ownerUser, premiumProg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if status.Status != models.NutritionAssignmentStatusCompleted {
		t.Errorf("status = %q, want completed", status.Status)
	}
	if status.Plan == nil {
		t.Fatal("expected a generated plan for a completed assignment")
	}
	if h.assignments.markedProc != 1 || h.assignments.markedDone != 1 {
		t.Errorf("expected exactly one claim and one completion, got %d and %d", h.assignments.markedProc, h.assignments.markedDone)
	}
}

func TestRunIsIdempotentAfterCompletion(t *testing.T) {
	h := newHarness(t, premiumProgram(), true, nil)

	first, err := h.svc.Run(context.Background(), ownerUser, premiumProg)
	if err != nil {
		t.Fatalf("first Run failed: %v", err)
	}
	second, err := h.svc.Run(context.Background(), ownerUser, premiumProg)
	if err != nil {
		t.Fatalf("second Run failed: %v", err)
	}

	// A repeated run must converge on the same plan rather than regenerating a
	// competing one.
	if first.Plan.Fingerprint != second.Plan.Fingerprint {
		t.Error("expected a repeated run to return the same plan")
	}
	if h.assignments.markedProc != 1 {
		t.Errorf("expected generation to be claimed once, got %d claims", h.assignments.markedProc)
	}
}

func TestRunRefusesWithoutAnEntitlement(t *testing.T) {
	// A plan must never be generated for a program the account has not bought.
	h := newHarness(t, premiumProgram(), false, nil)

	if _, err := h.svc.Run(context.Background(), ownerUser, premiumProg); !errors.Is(err, nutrition_assignment.ErrAssignmentNotFound) {
		t.Fatalf("expected ErrAssignmentNotFound without an entitlement, got %v", err)
	}
	if h.assignments.ensureCalls != 0 {
		t.Error("expected no assignment to be provisioned without an entitlement")
	}
}

func TestGetStatusRefusesCrossAccountAccess(t *testing.T) {
	// The entitlement is the gate, so a different account reading the status of
	// the same program learns nothing.
	h := newHarness(t, premiumProgram(), true, nil)
	h.entitlements.owned[ownerUser] = true

	if _, err := h.svc.GetStatus(context.Background(), otherUser, premiumProg); !errors.Is(err, nutrition_assignment.ErrAssignmentNotFound) {
		t.Fatalf("expected a cross-account read to be refused, got %v", err)
	}
}

func TestRunRetriesAFailedGenerationInPlace(t *testing.T) {
	// A failed run is recovered by retrying the same assignment, never by
	// creating a second plan.
	h := newHarness(t, premiumProgram(), true, failingGenerator{})

	if _, err := h.svc.Run(context.Background(), ownerUser, premiumProg); err == nil {
		t.Fatal("expected a generator failure to surface")
	}
	if h.assignments.markedFail != 1 {
		t.Errorf("expected the failure to be recorded once, got %d", h.assignments.markedFail)
	}
	if h.assignments.assignment.Status != models.NutritionAssignmentStatusFailed {
		t.Errorf("status = %q, want failed", h.assignments.assignment.Status)
	}

	// Swapping in a working generator and retrying completes the same row.
	h.svc = nutrition_assignment.NewService(
		h.programs, h.entitlements, h.questionn, h.assignments,
		nutrition_assignment.NewDeterministicGenerator(),
	)
	status, err := h.svc.Run(context.Background(), ownerUser, premiumProg)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if status.Status != models.NutritionAssignmentStatusCompleted {
		t.Errorf("status = %q, want completed after a retry", status.Status)
	}
	if h.assignments.assignment.ID != assignmentID {
		t.Error("expected the retry to reuse the existing assignment row")
	}
}

func TestRunRefusesWhileGenerationIsInFlight(t *testing.T) {
	// The claim is taken before generation, so a concurrent second run cannot
	// produce a competing plan.
	h := newHarness(t, premiumProgram(), true, nil)
	h.assignments.assignment = &models.NutritionAssignment{
		ID:                   assignmentID,
		UserID:               ownerUser,
		ProgramID:            premiumProg,
		QuestionnaireVersion: 1,
		Status:               models.NutritionAssignmentStatusProcessing,
		Version:              1,
	}

	if _, err := h.svc.Run(context.Background(), ownerUser, premiumProg); !errors.Is(err, nutrition_assignment.ErrInvalidInput) {
		t.Fatalf("expected an in-flight assignment to be refused, got %v", err)
	}
	if h.assignments.markedProc != 0 {
		t.Error("expected no generation to be claimed while one is in flight")
	}
}

func TestGetStatusFlagsAnOutdatedPlan(t *testing.T) {
	// A questionnaire resubmission after generation must be visible instead of
	// silently serving a stale plan.
	h := newHarness(t, premiumProgram(), true, nil)

	if _, err := h.svc.Run(context.Background(), ownerUser, premiumProg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	h.questionn.stored = &nutrition_questionnaire.StoredIntake{
		QuestionnaireID: questionnID,
		Version:         2,
		Intake:          mustIntake(t, validIntake(t)),
	}

	status, err := h.svc.GetStatus(context.Background(), ownerUser, premiumProg)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if !status.OutOfDate {
		t.Error("expected a plan derived from a superseded intake to be reported as out of date")
	}
	if status.QuestionnaireVersion != 1 {
		t.Errorf("questionnaire version = %d, want the version the plan was built from", status.QuestionnaireVersion)
	}
}

func TestEnsureProvisionedIgnoresNonPremiumProducts(t *testing.T) {
	// A product outside the family has no plan, so nothing is created and no
	// error is raised for the caller that triggered it.
	h := newHarness(t, genericProgram(), true, nil)

	if err := h.svc.EnsureProvisioned(context.Background(), ownerUser, genericProg); err != nil {
		t.Fatalf("expected a non-premium product to be ignored, got %v", err)
	}
	if h.assignments.ensureCalls != 0 {
		t.Error("expected no assignment for a non-premium product")
	}
}

func TestEnsureProvisionedRefusesWithoutAnIntake(t *testing.T) {
	// The intake is a checkout precondition, so a plan can never be derived
	// from nothing even if the precondition is somehow bypassed.
	h := newHarness(t, premiumProgram(), true, nil)
	h.questionn.stored = nil

	err := h.svc.EnsureProvisioned(context.Background(), ownerUser, premiumProg)
	if !errors.Is(err, nutrition_assignment.ErrInvalidInput) {
		t.Fatalf("expected provisioning without an intake to be refused, got %v", err)
	}
	if h.assignments.ensureCalls != 0 {
		t.Error("expected no assignment to be provisioned without an intake")
	}
}

func TestGetStatusProvisionsOnDemandAfterCheckout(t *testing.T) {
	// Landing on the plan straight after checkout must surface a real pending
	// state rather than a bare not-found.
	h := newHarness(t, premiumProgram(), true, nil)

	status, err := h.svc.GetStatus(context.Background(), ownerUser, premiumProg)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.Status != models.NutritionAssignmentStatusPending {
		t.Errorf("status = %q, want pending", status.Status)
	}
	if status.Plan != nil {
		t.Error("expected no plan before generation has run")
	}
}

func TestStatusNeverExposesInternalDiagnostics(t *testing.T) {
	// A stored failure reason is admin-only and must not reach the client.
	h := newHarness(t, premiumProgram(), true, nil)
	reason := "nutrition generation failed: generator unavailable"
	h.assignments.assignment = &models.NutritionAssignment{
		ID:                   assignmentID,
		UserID:               ownerUser,
		ProgramID:            premiumProg,
		QuestionnaireVersion: 1,
		Status:               models.NutritionAssignmentStatusFailed,
		Version:              1,
		FailureReason:        &reason,
	}

	status, err := h.svc.GetStatus(context.Background(), ownerUser, premiumProg)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(encoded), "generator unavailable") {
		t.Error("status response must not carry the internal failure reason")
	}
}

func mustIntake(t *testing.T, raw []byte) *nutrition_questionnaire.Normalized {
	t.Helper()
	normalized, err := nutrition_questionnaire.Unmarshal(raw)
	if err != nil {
		t.Fatalf("failed to decode fixture: %v", err)
	}
	return normalized
}

func strPtr(v string) *string     { return &v }
func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }
