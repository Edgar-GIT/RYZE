// Package questionnaires coordinates the Premium Level 1 client intake: reading
// the requirement state for a program, storing a validated submission, and
// reporting whether a program may be checked out yet.
//
// The service never trusts client-supplied identity, ownership, program state or
// validation outcomes. The user id always comes from the authentication
// context, the program must be a published Premium Level 1 program, and the
// answers are re-validated here before anything is written.
package questionnaires

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/nutrition_questionnaire"
)

var (
	// ErrInvalidInput indicates the request was malformed: a bad identifier or a
	// questionnaire that failed validation.
	ErrInvalidInput = errors.New("invalid questionnaire input")
	// ErrProgramNotFound indicates the program does not exist, is soft-deleted,
	// is not published, or is not a Premium Level 1 program. All four cases are
	// reported identically so the questionnaire surface never reveals which
	// programs exist but are not questionnaire-driven.
	ErrProgramNotFound = errors.New("questionnaire program not found")
	// ErrIntakeRequired indicates the product cannot be checked out until the
	// authenticated buyer submits an intake. The purchase service surfaces it as
	// a precondition failure; it never reveals whether another account has one.
	ErrIntakeRequired = errors.New("nutrition intake required")
)

// ProgramReader resolves the program the questionnaire belongs to. It is the
// existing published-program read surface, so the product-type gate is evaluated
// against the same data the marketplace serves.
type ProgramReader interface {
	FindPublishedByID(ctx context.Context, programID string) (*models.Program, error)
}

// QuestionnaireRepository is the data-access surface the service needs.
type QuestionnaireRepository interface {
	UpsertForUserAndProgram(ctx context.Context, userID, programID string, answers []byte) (*models.NutritionQuestionnaire, error)
	FindByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionQuestionnaire, error)
	FindSummaryByUserAndProgram(ctx context.Context, userID, programID string) (*repositories.NutritionQuestionnaireSummary, error)
}

// Service is the questionnaire use-case surface consumed by the API layer.
type Service interface {
	// GetRequirement reports whether the program needs an intake and, when the
	// user already submitted one, which revision is stored. It is safe to call
	// for any authenticated user and never mutates state.
	GetRequirement(ctx context.Context, userID, programID string) (*Requirement, error)
	// Submit validates and stores an intake for the authenticated user.
	Submit(ctx context.Context, userID, programID string, answers nutrition_questionnaire.Answers) (*Requirement, error)
	// FindStored returns the validated intake that a nutrition generation run
	// must be derived from, together with the identity and revision of the stored
	// row. It fails with nutrition_questionnaire.ErrNotSubmitted when none exists.
	FindStored(ctx context.Context, userID, programID string) (*nutrition_questionnaire.StoredIntake, error)
	// Satisfied enforces the checkout precondition: a Premium Level 1 program
	// cannot be bought before the authenticated buyer submitted an intake. It is
	// part of this contract so the purchase service depends on a documented
	// behaviour rather than on the concrete implementation.
	Satisfied(ctx context.Context, userID, programID, productType string) error
}

// Requirement is the client-safe view of the intake state for one program. It
// deliberately carries no answer content: the client needs to know whether it
// must still answer, not to read its own health data back from the server.
type Requirement struct {
	ProgramID string `json:"program_id"`
	// Required is true for Premium Level 1 programs, which cannot be checked
	// out before an intake exists.
	Required bool `json:"required"`
	// Submitted is true once the authenticated user has a stored intake.
	Submitted bool `json:"submitted"`
	// Version is the stored intake revision, so the client can tell that a
	// resubmission supersedes a generated plan.
	Version int `json:"version"`
	// SchemaVersion is the intake contract version the stored intake was written
	// with.
	SchemaVersion int `json:"schema_version"`
	// SubmittedAt is when the current revision was accepted.
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	// MinSchemaVersion is the lowest contract version the backend still accepts,
	// so a client built against an older contract can refresh itself.
	MinSchemaVersion int `json:"min_schema_version"`
}

type service struct {
	programs       ProgramReader
	questionnaires QuestionnaireRepository
}

// NewService wires the published-program gate and the questionnaire repository.
func NewService(programs ProgramReader, questionnaires QuestionnaireRepository) Service {
	return &service{programs: programs, questionnaires: questionnaires}
}

// GetRequirement returns the intake state for a program. A program that does not
// require an intake still resolves, so the client can use one endpoint to decide
// whether to show the questionnaire step.
func (s *service) GetRequirement(ctx context.Context, userID, programID string) (*Requirement, error) {
	if err := validateIdentifiers(userID, programID); err != nil {
		return nil, err
	}

	program, err := s.programs.FindPublishedByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load program: %w", err)
	}

	requirement := &Requirement{
		ProgramID:        programID,
		Required:         program.ProductType == models.ProgramProductTypePremiumLevel1,
		SchemaVersion:    nutrition_questionnaire.SchemaVersion,
		MinSchemaVersion: nutrition_questionnaire.SchemaVersion,
	}

	summary, err := s.questionnaires.FindSummaryByUserAndProgram(ctx, userID, programID)
	switch {
	case errors.Is(err, repositories.ErrNutritionQuestionnaireNotFound):
		return requirement, nil
	case err != nil:
		return nil, fmt.Errorf("failed to load questionnaire summary: %w", err)
	}

	requirement.Submitted = true
	requirement.Version = summary.Version
	submittedAt := summary.SubmittedAt
	requirement.SubmittedAt = &submittedAt
	return requirement, nil
}

// Submit validates and stores the intake for the authenticated user. The whole
// submission is accepted or rejected as a unit: a partially valid intake is
// never persisted, and a resubmission supersedes the previous revision.
func (s *service) Submit(ctx context.Context, userID, programID string, answers nutrition_questionnaire.Answers) (*Requirement, error) {
	if err := validateIdentifiers(userID, programID); err != nil {
		return nil, err
	}

	program, err := s.programs.FindPublishedByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load program: %w", err)
	}
	if program.ProductType != models.ProgramProductTypePremiumLevel1 {
		return nil, ErrProgramNotFound
	}

	normalized, err := nutrition_questionnaire.Normalize(answers)
	if err != nil {
		var validation *nutrition_questionnaire.ValidationError
		if errors.As(err, &validation) {
			// The per-field reasons are returned as structured validation
			// detail. They name fields, never the values that failed.
			return nil, fmt.Errorf("%w: %w", ErrInvalidInput, &FieldValidationError{Fields: validation.FieldErrors()})
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	encoded, err := normalized.Marshal()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	if _, err := s.questionnaires.UpsertForUserAndProgram(ctx, userID, programID, encoded); err != nil {
		if errors.Is(err, repositories.ErrNutritionQuestionnaireAlreadyExists) {
			return nil, fmt.Errorf("%w: questionnaire is being updated concurrently", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to store nutrition questionnaire: %w", err)
	}

	return s.GetRequirement(ctx, userID, programID)
}

// FindStored returns the validated intake for generation, together with the
// identity and revision of the row it came from. It is the read the nutrition
// assignment service depends on, so a plan can never be derived from anything
// other than a stored, validated document.
func (s *service) FindStored(ctx context.Context, userID, programID string) (*nutrition_questionnaire.StoredIntake, error) {
	if err := validateIdentifiers(userID, programID); err != nil {
		return nil, err
	}

	questionnaire, err := s.questionnaires.FindByUserAndProgram(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrNutritionQuestionnaireNotFound) {
			return nil, nutrition_questionnaire.ErrNotSubmitted
		}
		return nil, fmt.Errorf("failed to load nutrition questionnaire: %w", err)
	}

	normalized, err := nutrition_questionnaire.Unmarshal(questionnaire.Answers)
	if err != nil {
		return nil, fmt.Errorf("failed to decode stored questionnaire: %w", err)
	}

	return &nutrition_questionnaire.StoredIntake{
		QuestionnaireID: questionnaire.ID,
		Version:         questionnaire.Version,
		Intake:          normalized,
	}, nil
}

// Satisfied implements the checkout precondition gate. It is evaluated by the
// purchase service with the authenticated buyer and the already-resolved product
// type, before any purchase row is created.
//
// A product type other than Premium Level 1 has no intake requirement, so the
// gate is transparent for every existing product.
func (s *service) Satisfied(ctx context.Context, userID, programID, productType string) error {
	if productType != models.ProgramProductTypePremiumLevel1 {
		return nil
	}
	if err := validateIdentifiers(userID, programID); err != nil {
		return err
	}
	if _, err := s.questionnaires.FindByUserAndProgram(ctx, userID, programID); err != nil {
		if errors.Is(err, repositories.ErrNutritionQuestionnaireNotFound) {
			return ErrIntakeRequired
		}
		return fmt.Errorf("failed to verify nutrition questionnaire: %w", err)
	}
	return nil
}

// FieldValidationError carries the per-field rejection reasons of a malformed
// submission. It is exported so the API layer can render field-level detail
// without depending on the questionnaire contract package.
type FieldValidationError struct {
	Fields map[string]string
}

func (e *FieldValidationError) Error() string {
	return "questionnaire validation failed"
}

// FieldReasons returns the per-field rejection reasons keyed by field name.
func (e *FieldValidationError) FieldReasons() map[string]string {
	return e.Fields
}

// validateIdentifiers rejects malformed identifiers before any query runs, so an
// invalid identifier can never reach the database. Both identifiers are parsed
// as UUIDs, which is the storage format of users and programs.
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
