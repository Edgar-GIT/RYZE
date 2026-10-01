package nutrition_assignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/nutrition_questionnaire"
)

var (
	// ErrInvalidInput indicates the request was malformed or the plan is not in a
	// state the request could move forward.
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

// AssignmentRepository is the assignment lifecycle data-access surface. Writing a
// plan is deliberately absent: completing an assignment means persisting a plan
// version, and that is a single operation owned by PlanRepository so the
// supersede-and-insert cannot be split.
type AssignmentRepository interface {
	EnsurePendingForUserAndProgram(ctx context.Context, userID, programID, questionnaireID string, questionnaireVersion int) (*models.NutritionAssignment, bool, error)
	FindByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionAssignment, error)
	// MarkProcessing claims the assignment for generation at a specific intake
	// revision, and only if the assignment is still at expectedVersion. It is a
	// compare-and-swap so two runs cannot both generate for the same revision.
	MarkProcessing(ctx context.Context, userID, programID, questionnaireID string, questionnaireVersion, expectedVersion int) error
	MarkFailed(ctx context.Context, userID, programID, reason string) error
}

// PlanRepository persists and reads generated plan versions.
type PlanRepository interface {
	Complete(ctx context.Context, userID, programID string, plan *models.NutritionPlan) error
	FindActiveByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionPlan, error)
	FindActiveByAssignment(ctx context.Context, assignmentID string) (*models.NutritionPlan, error)
}

// Service owns the nutrition assignment lifecycle: provisioning the placeholder
// a verified purchase creates, running the deterministic generation step, and
// exposing the owner-scoped status read.
type Service interface {
	// EnsureProvisioned creates the pending assignment a verified Premium
	// Level 1 purchase implies. It is idempotent: repeated deliveries of the
	// same purchase converge on one assignment.
	EnsureProvisioned(ctx context.Context, userID, programID string) error
	// FulfillPurchase provisions and generates the plan a completed Premium
	// Level 1 purchase owes. It is a no-op for any other product family, and it
	// is idempotent so a repeated purchase delivery is safe. It exists so the
	// purchase path never has to know how a plan is built.
	FulfillPurchase(ctx context.Context, userID, programID string) error
	// Run performs the generation step. It is safe to call repeatedly: an
	// assignment that already holds a current plan is reported as-is, and a
	// previous failure is retried in place.
	Run(ctx context.Context, userID, programID string) (*Status, error)
	// GetStatus returns the owner-scoped assignment state. A plan is only
	// included once generation completed.
	GetStatus(ctx context.Context, userID, programID string) (*Status, error)
}

type service struct {
	programs       ProgramReader
	entitlements   EntitlementReader
	questionnaires QuestionnaireReader
	assignments    AssignmentRepository
	plans          PlanRepository
	generator      Generator
}

// NewService wires the program gate, the intake reader, the assignment and plan
// repositories and the generator.
func NewService(programs ProgramReader, entitlements EntitlementReader, questionnaires QuestionnaireReader, assignments AssignmentRepository, plans PlanRepository, generator Generator) Service {
	return &service{
		programs:       programs,
		entitlements:   entitlements,
		questionnaires: questionnaires,
		assignments:    assignments,
		plans:          plans,
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
	// generation. It matches the version of the active plan.
	Version int `json:"version"`
	// QuestionnaireVersion is the intake revision the active plan was derived
	// from.
	QuestionnaireVersion int `json:"questionnaire_version"`
	// OutOfDate is true when the intake moved on after the active plan was
	// generated, so the client can explain why a refresh is needed instead of
	// silently serving a plan that no longer matches the intake.
	OutOfDate bool `json:"out_of_date"`
	// Plan is always present on the wire, null until generation completes, so a
	// client does not have to distinguish an absent field from an empty plan.
	Plan *Plan `json:"plan"`
}

// Plan is the client-facing nutrition plan.
//
// It is a projection of the stored normalised plan, not a storage type: the
// service builds it here so the client contract is explicit about what may leave
// the server. Internal provenance such as the catalog code a food was resolved
// from is deliberately absent — it exists for auditing and has no use on the
// client.
type Plan struct {
	Version             int         `json:"version"`
	EngineVersion       int         `json:"engine_version"`
	Fingerprint         string      `json:"fingerprint"`
	DietaryPattern      string      `json:"dietary_pattern"`
	MaintenanceCalories int         `json:"maintenance_calories"`
	TargetCalories      int         `json:"target_calories"`
	Daily               Macros      `json:"daily"`
	ProteinPercent      int         `json:"protein_percent"`
	CarbohydratePercent int         `json:"carbohydrate_percent"`
	FatPercent          int         `json:"fat_percent"`
	MealsPerDay         int         `json:"meals_per_day"`
	SnacksPerDay        int         `json:"snacks_per_day"`
	Meals               []Meal      `json:"meals"`
	Exclusions          []Exclusion `json:"exclusions"`
	Hydration           Hydration   `json:"hydration"`
	Cautions            []string    `json:"cautions"`
	PrepGuidance        string      `json:"prep_guidance"`
	Summary             string      `json:"summary"`
}

// Meal is one eating occasion of a plan.
type Meal struct {
	Position       int        `json:"position"`
	Label          string     `json:"label"`
	Kind           string     `json:"kind"`
	PercentOfDaily int        `json:"percent_of_daily"`
	Macros         Macros     `json:"macros"`
	Notes          string     `json:"notes,omitempty"`
	Items          []MealItem `json:"items"`
}

// MealItem is one food in a meal, with the quantity the client should use.
type MealItem struct {
	Position         int     `json:"position"`
	FoodName         string  `json:"food_name"`
	Category         string  `json:"category"`
	Quantity         float64 `json:"quantity"`
	Unit             string  `json:"unit"`
	Macros           Macros  `json:"macros"`
	SubstitutionNote string  `json:"substitution_note,omitempty"`
}

// Exclusion is one recorded dietary exclusion, so a client can see what the plan
// took into account rather than having to infer it from what is missing.
type Exclusion struct {
	ReasonCode string `json:"reason_code"`
	Token      string `json:"token"`
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

	_, inFamily, err := s.premiumLevel1Program(ctx, programID)
	if err != nil || !inFamily {
		return err
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

// FulfillPurchase delivers the plan a completed Premium Level 1 purchase owes.
//
// It is idempotent, so a webhook, a capture callback and a Test Mode completion
// can all call it for the same purchase and converge on one plan. A product
// outside the family is a no-op rather than an error: a completed Generic purchase
// owes nothing here, and reporting a failure would make every such sale look
// broken.
func (s *service) FulfillPurchase(ctx context.Context, userID, programID string) error {
	if err := validateIdentifiers(userID, programID); err != nil {
		return err
	}

	_, inFamily, err := s.premiumLevel1Program(ctx, programID)
	if err != nil || !inFamily {
		return err
	}

	if _, err := s.Run(ctx, userID, programID); err != nil {
		return fmt.Errorf("failed to fulfill nutrition plan: %w", err)
	}
	return nil
}

// premiumLevel1Program reports whether a published program belongs to the Premium
// Level 1 family, which is the only family that owes a nutrition plan.
//
// Every entry point resolves the family through here rather than repeating the
// check, because a family condition that exists in only some of the places that
// need it is how a tier becomes reachable without its prerequisites enforced.
func (s *service) premiumLevel1Program(ctx context.Context, programID string) (*models.Program, bool, error) {
	program, err := s.programs.FindPublishedByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, false, ErrAssignmentNotFound
		}
		return nil, false, fmt.Errorf("failed to load program: %w", err)
	}
	if program.ProductType != models.ProgramProductTypePremiumLevel1 {
		return program, false, nil
	}
	return program, true, nil
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
// The claim is taken before generation and pins the intake revision being
// generated, so two concurrent runs cannot both generate: the loser is told the
// assignment is not retryable instead of producing a competing plan. Persisting
// the plan re-asserts that same revision under a row lock, so a plan can never
// claim to be current for an intake it did not read.
//
// An assignment that already holds a current plan is reported as-is, which makes
// a repeated request idempotent. When the intake has moved on since the active
// plan was built, an explicit run supersedes it with a new version; the read path
// never does this on its own.
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

	// The intake is read once and used for both the claim and the generation, so
	// the revision the plan is built from cannot move between the two.
	stored, err := s.loadIntake(ctx, userID, programID)
	if err != nil {
		return nil, err
	}

	active, err := s.plans.FindActiveByUserAndProgram(ctx, userID, programID)
	if err != nil && !errors.Is(err, repositories.ErrNutritionPlanNotFound) {
		return nil, fmt.Errorf("failed to load nutrition plan: %w", err)
	}

	switch {
	case assignment.Status == models.NutritionAssignmentStatusCompleted &&
		active != nil &&
		active.QuestionnaireVersion == stored.Version:
		// Generation is deterministic, so an already-completed assignment whose
		// active plan was built from the current intake needs no work. This is
		// what makes a retried request idempotent.
		return s.GetStatus(ctx, userID, programID)
	case assignment.Status == models.NutritionAssignmentStatusProcessing:
		return nil, fmt.Errorf("%w: generation already in progress", ErrInvalidInput)
	}

	// Claiming also covers the repair case: a completed assignment whose plan
	// cannot be read, and a completed assignment whose intake moved on, are both
	// claimed again. The expected version makes the claim a compare-and-swap, so
	// a run that re-reads an assignment another run already advanced is rejected
	// rather than generating a second time for the same revision.
	if err := s.assignments.MarkProcessing(
		ctx, userID, programID, stored.QuestionnaireID, stored.Version, assignment.Version,
	); err != nil {
		if errors.Is(err, repositories.ErrNutritionAssignmentNotRetryable) {
			return nil, fmt.Errorf("%w: assignment is not in a retryable state", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to claim nutrition assignment: %w", err)
	}

	generated, runErr := s.generator.Generate(stored.Intake)
	if runErr != nil {
		// The recorded reason is a server-generated diagnostic. Questionnaire
		// content is never included, because a stored failure reason must never
		// carry a health answer.
		runErr = fmt.Errorf("nutrition generation failed: %w", runErr)
		if markErr := s.assignments.MarkFailed(ctx, userID, programID, runErr.Error()); markErr != nil {
			return nil, fmt.Errorf("nutrition generation failed: %w", runErr)
		}
		return nil, runErr
	}

	plan, err := toStoredPlan(generated, assignment, stored)
	if err != nil {
		if markErr := s.assignments.MarkFailed(ctx, userID, programID, err.Error()); markErr != nil {
			return nil, fmt.Errorf("failed to build nutrition plan: %w", err)
		}
		return nil, err
	}

	if err := s.plans.Complete(ctx, userID, programID, plan); err != nil {
		if errors.Is(err, repositories.ErrNutritionAssignmentNotRetryable) {
			return nil, fmt.Errorf("%w: assignment changed during generation", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to complete nutrition assignment: %w", err)
	}

	return s.GetStatus(ctx, userID, programID)
}

// loadIntake reads the stored intake and maps the not-submitted case onto the
// service input error, because checkout requires the intake before a purchase and
// reaching generation without one means the precondition was bypassed.
func (s *service) loadIntake(ctx context.Context, userID, programID string) (*nutrition_questionnaire.StoredIntake, error) {
	stored, err := s.questionnaires.FindStored(ctx, userID, programID)
	if err != nil {
		if errors.Is(err, nutrition_questionnaire.ErrNotSubmitted) {
			return nil, fmt.Errorf("%w: no submitted questionnaire", ErrInvalidInput)
		}
		return nil, fmt.Errorf("failed to load submitted questionnaire: %w", err)
	}
	return stored, nil
}

// GetStatus returns the owner-scoped assignment state, including the active plan
// once generation completed. A plan whose intake revision is older than the
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

	// The plan is projected from the normalised tables, which are the only
	// source of truth. An assignment with no readable plan is never reported as
	// delivered, even if its status column says completed.
	plan, planErr := s.plans.FindActiveByUserAndProgram(ctx, userID, programID)
	switch {
	case planErr == nil:
		if projection, projectionErr := toClientPlan(plan); projectionErr == nil {
			status.Plan = projection
		} else {
			status.Status = models.NutritionAssignmentStatusFailed
		}
	case errors.Is(planErr, repositories.ErrNutritionPlanNotFound):
		if status.Status == models.NutritionAssignmentStatusCompleted {
			// The completion is real but its plan is unreadable. Reporting the
			// assignment as pending would be a lie, and reporting it as delivered
			// would show nothing, so it is surfaced as failed and repairable.
			status.Status = models.NutritionAssignmentStatusFailed
		}
	default:
		return nil, fmt.Errorf("failed to load nutrition plan: %w", planErr)
	}

	return status, nil
}

// toStoredPlan projects a generated plan onto the persisted normalised domain.
// The identity and version columns are filled by the repository inside the
// completing transaction, because only it holds the assignment lock and therefore
// the only truth about which version this is.
func toStoredPlan(generated *GeneratedPlan, assignment *models.NutritionAssignment, stored *nutrition_questionnaire.StoredIntake) (*models.NutritionPlan, error) {
	cautions, err := json.Marshal(generated.Cautions)
	if err != nil {
		return nil, fmt.Errorf("failed to encode nutrition cautions: %w", err)
	}

	plan := &models.NutritionPlan{
		AssignmentID:         assignment.ID,
		UserID:               assignment.UserID,
		ProgramID:            assignment.ProgramID,
		QuestionnaireID:      stored.QuestionnaireID,
		QuestionnaireVersion: stored.Version,
		EngineVersion:        generated.EngineVersion,
		Fingerprint:          generated.Fingerprint,
		DietaryPattern:       generated.DietaryPattern,
		MaintenanceCalories:  generated.MaintenanceCalories,
		TargetCalories:       generated.TargetCalories,
		DailyProteinGrams:    generated.Daily.ProteinGrams,
		DailyCarbsGrams:      generated.Daily.CarbsGrams,
		DailyFatGrams:        generated.Daily.FatGrams,
		DailyFiberGrams:      generated.Daily.FiberGrams,
		ProteinPercent:       generated.ProteinPercent,
		CarbohydratePercent:  generated.CarbohydratePercent,
		FatPercent:           generated.FatPercent,
		MealsPerDay:          generated.MealsPerDay,
		SnacksPerDay:         generated.SnacksPerDay,
		HydrationLitres:      generated.Hydration.DailyLitres,
		HydrationNote:        generated.Hydration.Note,
		Summary:              generated.Summary,
		PrepGuidance:         generated.PrepGuidance,
		Cautions:             cautions,
		Meals:                make([]models.NutritionPlanMeal, 0, len(generated.Meals)),
		Exclusions:           make([]models.NutritionPlanExclusion, 0, len(generated.Exclusions)),
	}

	// The plan identifier is assigned before any child is built so every child
	// row can reference it before the parent exists.
	plan.ID = uuid.NewString()

	for _, meal := range generated.Meals {
		// The meals and foods are not GORM associations — they are plain slices
		// the repository writes explicitly — so their identifiers are assigned
		// here rather than by a create hook.
		storedMeal := models.NutritionPlanMeal{
			ID:                uuid.NewString(),
			PlanID:            plan.ID,
			Position:          meal.Position,
			Label:             meal.Label,
			Kind:              meal.Kind,
			PercentOfDaily:    meal.PercentOfDaily,
			Calories:          meal.Macros.Calories,
			ProteinGrams:      meal.Macros.ProteinGrams,
			CarbohydrateGrams: meal.Macros.CarbsGrams,
			FatGrams:          meal.Macros.FatGrams,
			Items:             make([]models.NutritionPlanMealItem, 0, len(meal.Items)),
		}
		if meal.Notes != "" {
			notes := meal.Notes
			storedMeal.Notes = &notes
		}

		for _, item := range meal.Items {
			storedItem := models.NutritionPlanMealItem{
				ID:                uuid.NewString(),
				PlanID:            plan.ID,
				MealID:            storedMeal.ID,
				Position:          item.Position,
				CatalogCode:       item.CatalogCode,
				FoodName:          item.FoodName,
				Category:          string(item.Category),
				Quantity:          item.Quantity,
				Unit:              item.Unit,
				Calories:          item.Macros.Calories,
				ProteinGrams:      item.Macros.ProteinGrams,
				CarbohydrateGrams: item.Macros.CarbsGrams,
				FatGrams:          item.Macros.FatGrams,
				FiberGrams:        item.Macros.FiberGrams,
			}
			if item.SubstitutionNote != "" {
				note := item.SubstitutionNote
				storedItem.SubstitutionNote = &note
			}
			storedMeal.Items = append(storedMeal.Items, storedItem)
		}

		plan.Meals = append(plan.Meals, storedMeal)
	}

	for _, exclusion := range generated.Exclusions {
		plan.Exclusions = append(plan.Exclusions, models.NutritionPlanExclusion{
			ID:         uuid.NewString(),
			PlanID:     plan.ID,
			ReasonCode: exclusion.ReasonCode,
			Token:      exclusion.Token,
		})
	}

	return plan, nil
}

// toClientPlan projects the stored plan onto the client contract.
//
// Cautions are decoded here rather than being returned as raw JSON, so a
// malformed stored value degrades to no cautions instead of surfacing as invalid
// JSON on the wire.
func toClientPlan(stored *models.NutritionPlan) (*Plan, error) {
	if stored == nil {
		return nil, ErrAssignmentNotFound
	}

	cautions := []string{}
	if len(stored.Cautions) > 0 {
		if err := json.Unmarshal(stored.Cautions, &cautions); err != nil {
			return nil, fmt.Errorf("failed to decode nutrition cautions: %w", err)
		}
	}

	plan := &Plan{
		Version:             stored.Version,
		EngineVersion:       stored.EngineVersion,
		Fingerprint:         stored.Fingerprint,
		DietaryPattern:      stored.DietaryPattern,
		MaintenanceCalories: stored.MaintenanceCalories,
		TargetCalories:      stored.TargetCalories,
		Daily: Macros{
			ProteinGrams: stored.DailyProteinGrams,
			CarbsGrams:   stored.DailyCarbsGrams,
			FatGrams:     stored.DailyFatGrams,
			FiberGrams:   stored.DailyFiberGrams,
		},
		ProteinPercent:      stored.ProteinPercent,
		CarbohydratePercent: stored.CarbohydratePercent,
		FatPercent:          stored.FatPercent,
		MealsPerDay:         stored.MealsPerDay,
		SnacksPerDay:        stored.SnacksPerDay,
		Hydration: Hydration{
			DailyLitres: stored.HydrationLitres,
			Note:        stored.HydrationNote,
		},
		Summary:      stored.Summary,
		PrepGuidance: stored.PrepGuidance,
		Cautions:     cautions,
		Meals:        make([]Meal, 0, len(stored.Meals)),
		Exclusions:   make([]Exclusion, 0, len(stored.Exclusions)),
	}

	for _, meal := range stored.Meals {
		projected := Meal{
			Position:       meal.Position,
			Label:          meal.Label,
			Kind:           meal.Kind,
			PercentOfDaily: meal.PercentOfDaily,
			Macros: Macros{
				Calories:     meal.Calories,
				ProteinGrams: meal.ProteinGrams,
				CarbsGrams:   meal.CarbohydrateGrams,
				FatGrams:     meal.FatGrams,
			},
			Items: make([]MealItem, 0, len(meal.Items)),
		}
		if meal.Notes != nil {
			projected.Notes = *meal.Notes
		}

		for _, item := range meal.Items {
			projectedItem := MealItem{
				Position: item.Position,
				FoodName: item.FoodName,
				Category: item.Category,
				Quantity: item.Quantity,
				Unit:     item.Unit,
				Macros: Macros{
					Calories:     item.Calories,
					ProteinGrams: item.ProteinGrams,
					CarbsGrams:   item.CarbohydrateGrams,
					FatGrams:     item.FatGrams,
					FiberGrams:   item.FiberGrams,
				},
			}
			if item.SubstitutionNote != nil {
				projectedItem.SubstitutionNote = *item.SubstitutionNote
			}
			projected.Items = append(projected.Items, projectedItem)
		}

		plan.Meals = append(plan.Meals, projected)
	}

	for _, exclusion := range stored.Exclusions {
		plan.Exclusions = append(plan.Exclusions, Exclusion{
			ReasonCode: exclusion.ReasonCode,
			Token:      exclusion.Token,
		})
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
