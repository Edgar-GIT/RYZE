package nutrition_assignment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/nutrition_questionnaire"
)

var (
	// ErrInvalidInput indicates the request was malformed.
	ErrInvalidInput = errors.New("invalid nutrition assignment input")
	// ErrAssignmentNotFound indicates the authenticated user has no nutrition
	// assignment for the program. This covers both "not a Premium Level 1
	// program" and "no plan was generated", so the endpoint never reveals which
	// programs exist without a purchase.
	ErrAssignmentNotFound = errors.New("nutrition assignment not found")
)

// QuestionnaireReader returns the stored intake a plan is derived from, together
// with the identity and revision of the row it came from.
type QuestionnaireReader interface {
	FindStored(ctx context.Context, userID, programID string) (*nutrition_questionnaire.StoredIntake, error)
}

// ProgramReader resolves the program an assignment belongs to.
type ProgramReader interface {
	FindPublishedByID(ctx context.Context, programID string) (*models.Program, error)
}

// EntitlementReader is the authority for whether a plan is owed. An entitlement
// is only ever created inside the purchase completion transaction or by Test
// Mode, so gating on it is what guarantees a nutrition plan can never exist for a
// program the buyer has not actually acquired.
type EntitlementReader interface {
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.Entitlement, error)
}

// AssignmentRepository is the data-access surface the service needs.
type AssignmentRepository interface {
	EnsurePendingForUserAndProgram(ctx context.Context, userID, programID, questionnaireID string, questionnaireVersion int) (*models.NutritionAssignment, bool, error)
	FindByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionAssignment, error)
	MarkProcessing(ctx context.Context, userID, programID string) error
	MarkCompleted(ctx context.Context, userID, programID string, configuration []byte, questionnaireVersion int) error
	MarkFailed(ctx context.Context, userID, programID, reason string) error
}

// Service owns the nutrition assignment lifecycle: provisioning the placeholder
// a verified purchase creates, running the deterministic generation step, and
// exposing the owner-scoped status read.
type Service interface {
	// EnsureProvisioned creates the pending assignment a verified Premium
	// Level 1 purchase implies. It is idempotent: repeated deliveries of the
	// same purchase converge on one assignment.
	EnsureProvisioned(ctx context.Context, userID, programID string) error
	// Run performs the generation step. It is safe to call repeatedly: an
	// assignment that already holds a current plan is reported as-is, and a
	// previous failure is retried in place.
	Run(ctx context.Context, userID, programID string) (*Status, error)
	// GetStatus returns the owner-scoped assignment state. A plan is only
	// included once generation completed, and OutOfDate is set when the intake
	// was resubmitted after the plan was generated.
	GetStatus(ctx context.Context, userID, programID string) (*Status, error)
}

type service struct {
	programs       ProgramReader
	entitlements   EntitlementReader
	questionnaires QuestionnaireReader
	assignments    AssignmentRepository
	generator      Generator
}

// NewService wires the program gate, the intake reader, the assignment
// repository and the generator.
func NewService(programs ProgramReader, entitlements EntitlementReader, questionnaires QuestionnaireReader, assignments AssignmentRepository, generator Generator) Service {
	return &service{
		programs:       programs,
		entitlements:   entitlements,
		questionnaires: questionnaires,
		assignments:    assignments,
		generator:      generator,
	}
}

// Status is the owner-scoped nutrition state for one program. Plan is populated
// only for a completed assignment. Internal failure diagnostics are never
// exposed here.
type Status struct {
	ProgramID string `json:"program_id"`
	// Status mirrors the assignment state machine: pending, processing,
	// completed or failed.
	Status string `json:"status"`
	// Version is the assignment revision, incremented on every completed
	// generation.
	Version int `json:"version"`
	// QuestionnaireVersion is the intake revision the plan was derived from.
	QuestionnaireVersion int `json:"questionnaire_version"`
	// OutOfDate is true when the intake was resubmitted after this plan was
	// generated, so the client can prompt for a regeneration instead of being
	// served a stale plan.
	OutOfDate bool           `json:"out_of_date"`
	Plan      *GeneratedPlan `json:"plan,omitempty"`
}

// EnsureProvisioned records that a Premium Level 1 purchase happened and a plan
// is owed, creating the pending assignment when none exists.
//
// The entitlement is the authority: it is created atomically with a verified
// purchase, so requiring it here is what guarantees a plan can never be
// provisioned for a program the buyer has not acquired. Provisioning is
// idempotent, so replaying it for the same purchase converges on one assignment.
func (s *service) EnsureProvisioned(ctx context.Context, userID, programID string) error {
	if err := validateIdentifiers(userID, programID); err != nil {
		return err
	}

	program, err := s.programs.FindPublishedByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return ErrAssignmentNotFound
		}
		return fmt.Errorf("failed to load program: %w", err)
	}
	if program.ProductType != models.ProgramProductTypePremiumLevel1 {
		// A product outside the Premium Level 1 family has no nutrition
		// assignment. Returning success keeps the caller uniform without
		// inventing state for a product that does not need a plan.
		return nil
	}

	if err := s.requireEntitlement(ctx, userID, programID); err != nil {
		return err
	}

	stored, err := s.questionnaires.FindStored(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, nutrition_questionnaire.ErrNotSubmitted) {
			// The intake is a checkout precondition, so reaching this point
			// means it was bypassed. Refusing to provision keeps a plan from
			// ever being derived from nothing.
			return fmt.Errorf("%w: no submitted questionnaire", ErrInvalidInput)
		}
		return fmt.Errorf("failed to load submitted questionnaire: %w", err)
	}

	if _, _, err := s.assignments.EnsurePendingForUserAndProgram(
		ctx, userID, programID, stored.QuestionnaireID, stored.Version,
	); err != nil {
		return fmt.Errorf("failed to provision nutrition assignment: %w", err)
	}
	return nil
}

// requireEntitlement rejects a request from an account that does not own the
// program. A missing, soft-deleted or cross-user entitlement is reported
// identically, so this endpoint never reveals who bought a program.
func (s *service) requireEntitlement(ctx context.Context, userID, programID string) error {
	if _, err := s.entitlements.FindActiveByUserAndProgram(ctx, userID, programID); err != nil {
		if errors.Is(err, repositories.ErrEntitlementNotFound) {
			return ErrAssignmentNotFound
		}
		return fmt.Errorf("failed to verify program entitlement: %w", err)
	}
	return nil
}

// Run performs the deterministic generation step for the authenticated owner.
//
// The claim is taken before generation, so two concurrent runs cannot both
// generate: the loser is told the assignment is not retryable instead of
// producing a competing plan. The completion write re-asserts the intake
// revision under a row lock, so a plan can never claim to be current for an
// intake it did not read.
func (s *service) Run(ctx context.Context, userID, programID string) (*Status, error) {
	if err := validateIdentifiers(userID, programID); err != nil {
		return nil, err
	}

	// Generation is only ever reachable for a program the buyer actually owns,
	// and a missing assignment is provisioned on demand. That makes the plan a
	// derived artifact of the entitlement: it can always be rebuilt, and it can
	// never exist for an unowned program.
	if err := s.EnsureProvisioned(ctx, userID, programID); err != nil {
		return nil, err
	}

	assignment, err := s.assignments.FindByUserAndProgram(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrNutritionAssignmentNotFound) {
			return nil, ErrAssignmentNotFound
		}
		return nil, fmt.Errorf("failed to load nutrition assignment: %w", err)
	}

	switch assignment.Status {
	case models.NutritionAssignmentStatusCompleted:
		// Generation is deterministic, so an already-completed assignment needs
		// no work. The stored plan is returned rather than regenerated.
		return s.GetStatus(ctx, userID, programID)
	case models.NutritionAssignmentStatusProcessing:
		return nil, fmt.Errorf("%w: generation already in progress", ErrInvalidInput)
	}

	if err := s.assignments.MarkProcessing(ctx, userID, programID); err != nil {
		if errors.Is(err, repositories.ErrNutritionAssignmentNotRetryable) {
			return nil, fmt.Errorf("%w: assignment is not in a retryable state", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to claim nutrition assignment: %w", err)
	}

	plan, runErr := s.generate(ctx, userID, programID)
	if runErr != nil {
		// The recorded reason is a server-generated diagnostic. Questionnaire
		// content is never included, because a stored failure reason must never
		// carry a health answer.
		if markErr := s.assignments.MarkFailed(ctx, userID, programID, runErr.Error()); markErr != nil {
			return nil, fmt.Errorf("nutrition generation failed: %w", runErr)
		}
		return nil, runErr
	}

	encoded, err := plan.Marshal()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}
	if err := s.assignments.MarkCompleted(ctx, userID, programID, encoded, assignment.QuestionnaireVersion); err != nil {
		if errors.Is(err, repositories.ErrNutritionAssignmentNotRetryable) {
			return nil, fmt.Errorf("%w: assignment changed during generation", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to complete nutrition assignment: %w", err)
	}

	return s.GetStatus(ctx, userID, programID)
}

// GetStatus returns the owner-scoped assignment state, including the plan once
// generation completed. A stored plan whose intake revision is older than the
// current one is reported as out of date rather than served as current.
func (s *service) GetStatus(ctx context.Context, userID, programID string) (*Status, error) {
	if err := validateIdentifiers(userID, programID); err != nil {
		return nil, err
	}

	// Reading the status is entitlement-gated for the same reason generation is:
	// a plan is private to the account that bought it. Provisioning on read means
	// a client that lands on the plan straight after checkout sees a real pending
	// state instead of a bare 404.
	if err := s.EnsureProvisioned(ctx, userID, programID); err != nil {
		return nil, err
	}

	assignment, err := s.assignments.FindByUserAndProgram(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrNutritionAssignmentNotFound) {
			return nil, ErrAssignmentNotFound
		}
		return nil, fmt.Errorf("failed to load nutrition assignment: %w", err)
	}

	status := &Status{
		ProgramID:            programID,
		Status:               assignment.Status,
		Version:              assignment.Version,
		QuestionnaireVersion: assignment.QuestionnaireVersion,
	}

	// Comparing against the current intake revision is what makes a
	// resubmission visible instead of leaving a stale plan in place unnoticed.
	if stored, intakeErr := s.questionnaires.FindStored(ctx, userID, programID); intakeErr == nil {
		status.OutOfDate = stored.Version != assignment.QuestionnaireVersion
	}

	if assignment.Status == models.NutritionAssignmentStatusCompleted && len(assignment.Configuration) > 0 {
		if plan, planErr := Unmarshal(assignment.Configuration); planErr == nil {
			status.Plan = plan
		}
	}
	return status, nil
}

// generate loads the current intake and runs the generator. Generation is
// deterministic, so the same intake always yields the same plan and a retry
// after a failure cannot drift from the original intent.
func (s *service) generate(ctx context.Context, userID, programID string) (*GeneratedPlan, error) {
	stored, err := s.questionnaires.FindStored(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, nutrition_questionnaire.ErrNotSubmitted) {
			return nil, fmt.Errorf("%w: questionnaire disappeared during generation", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to load submitted questionnaire: %w", err)
	}

	plan, err := s.generator.Generate(stored.Intake)
	if err != nil {
		return nil, fmt.Errorf("nutrition generation failed: %w", err)
	}
	return plan, nil
}

// validateIdentifiers rejects malformed identifiers before any query runs, so an
// invalid identifier can never reach the database. Both are parsed as UUIDs,
// which is the storage format of users and programs.
func validateIdentifiers(userID, programID string) error {
	if userID == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(userID); err != nil {
		return fmt.Errorf("%w: invalid user id", ErrInvalidInput)
	}
	if programID == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(programID); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}
