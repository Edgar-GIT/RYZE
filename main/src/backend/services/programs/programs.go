package programs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
	"ryze/backend/services/program_details"
)

var (
	// ErrInvalidInput indicates the program input was malformed or incomplete.
	ErrInvalidInput = errors.New("invalid program input")
	// ErrProgramNotFound indicates the program does not exist, is soft-deleted
	// or is not owned by the trainer performing the operation.
	ErrProgramNotFound = errors.New("program not found")
	// ErrProgramAlreadyPublished indicates the program is already in the
	// published state and cannot be published again.
	ErrProgramAlreadyPublished = errors.New("program already published")
	// ErrProgramLocked indicates the program is attached to a Premium Level 2
	// request and can no longer be published, deleted or edited after delivery.
	ErrProgramLocked = errors.New("program locked by premium level 2 delivery")
	// ErrPriceFixed indicates a price other than the plan's fixed catalogue
	// price was requested. Prices follow the catalogue category only (owner
	// decision, 2026-11-14).
	ErrPriceFixed = errors.New("program price is fixed by its catalogue category")
)

const (
	// MaxPageSize caps the number of programs returned in a single page.
	MaxPageSize = admin_users.MaxPageSize
	// MaxNameLength caps the program name length, matching the database column.
	MaxNameLength = 255
	// MaxDescriptionLength caps the program description length.
	MaxDescriptionLength = 5000
)

// ProgramRepository is the data-access surface required by the programs
// service. Every operation is scoped to an explicit trainer id; the repository
// never obtains it from an HTTP context.
type ProgramRepository interface {
	Create(ctx context.Context, program *models.Program) error
	FindByIDAndTrainer(ctx context.Context, trainerID, programID string) (*models.Program, error)
	FindByID(ctx context.Context, programID string) (*models.Program, error)
	ListByTrainer(ctx context.Context, trainerID string, page, limit int) ([]models.Program, int64, error)
	Update(ctx context.Context, trainerID, programID string, updates map[string]any) error
	UpdatePricing(ctx context.Context, programID string, priceMinorUnits int64, currency string) error
	Publish(ctx context.Context, trainerID, programID string) error
	SoftDelete(ctx context.Context, trainerID, programID string) error
}

// Program is the safe representation of one program. It carries only the
// public program metadata and never exposes deletion markers or any internal
// data.
type Program struct {
	ID               string
	TrainerID        string
	Name             string
	Description      string
	Type             string
	Status           string
	PriceMinorUnits  int64
	Currency         string
	Level            *string
	DurationWeeks    *int
	FrequencyPerWeek *int
	TrainingType     *string
	CoverImage       *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CreateProgramInput carries the fields accepted when creating a program. The
// trainer id never comes from the input: it is always the authenticated
// trainer.
// CreateProgramInput carries a new trainer plan. PriceMinorUnits and Currency
// are ignored: the price follows the plan type's catalogue category.
type CreateProgramInput struct {
	Name             string
	Description      string
	Type             string
	Status           string
	PriceMinorUnits  int64
	Currency         string
	Level            string
	DurationWeeks    int
	FrequencyPerWeek int
	TrainingType     string
	CoverImage       string
}

// UpdateProgramInput carries the optional whitelisted fields accepted when
// updating a program. Nil values mean "leave unchanged"; at least one field
// must be provided.
// UpdateProgramInput carries a partial trainer plan update. PriceMinorUnits and
// Currency are ignored: changing the type re-applies its catalogue price.
type UpdateProgramInput struct {
	Name             *string
	Description      *string
	Type             *string
	Status           *string
	PriceMinorUnits  *int64
	Currency         *string
	Level            *string
	DurationWeeks    *int
	FrequencyPerWeek *int
	TrainingType     *string
	CoverImage       *string
}

// UpdatePricingInput carries the pricing fields accepted when updating a
// program's price. Both fields are required.
type UpdatePricingInput struct {
	PriceMinorUnits int64
	Currency        string
}

// ListProgramsResult carries one page of programs plus the pagination metadata
// needed to render the list.
type ListProgramsResult struct {
	Programs []Program
	Total    int64
	Page     int
	Limit    int
}

// Service implements the trainer-owned program flow. Authorization (which
// trainer may operate) is enforced by the route middleware; ownership is
// guaranteed because the trainer id always comes from the authenticated trainer
// context and is never accepted from the client. This service never knows about
// HTTP, Gin or the trainer context.
type Service interface {
	CreateProgram(ctx context.Context, trainerID string, input CreateProgramInput) (*Program, error)
	ListPrograms(ctx context.Context, trainerID string, page, limit int) (ListProgramsResult, error)
	GetProgram(ctx context.Context, trainerID, programID string) (*Program, error)
	UpdateProgram(ctx context.Context, trainerID, programID string, input UpdateProgramInput) (*Program, error)
	// PublishProgram transitions a draft program to published. The transition
	// draft → published is the only allowed transition; publishing an already
	// published program returns ErrProgramAlreadyPublished. A missing,
	// soft-deleted or foreign program maps to ErrProgramNotFound.
	PublishProgram(ctx context.Context, trainerID, programID string) (*Program, error)
	DeleteProgram(ctx context.Context, trainerID, programID string) error
	// UpdateProgramPricing updates the price of any active program. The caller
	// (trainer owner or authorized administrator) is responsible for
	// authorization before calling this method. Free programs must have a price
	// of 0; paid programs must meet the configured minimum.
	UpdateProgramPricing(ctx context.Context, programID string, input UpdatePricingInput) (*Program, error)
	// GetProgramByID returns one active program by its id without ownership
	// scoping. This is used by the admin pricing path.
	GetProgramByID(ctx context.Context, programID string) (*Program, error)
}

type premiumLevel2LinkReader interface {
	PremiumLevel2Link(ctx context.Context, programID string) (linked bool, delivered bool, err error)
}

type service struct {
	programs ProgramRepository
	pricing  config.PricingConfig
}

func NewService(programs ProgramRepository, pricing config.PricingConfig) Service {
	return &service{programs: programs, pricing: pricing}
}

// CreateProgram creates a new program owned by the authenticated trainer. The
// trainer id always comes from the caller (the authenticated trainer context)
// and is never accepted from the client. The status defaults to draft when not
// provided; publishing is only a state and carries no purchase, assignment or
// access semantics.
func (s *service) CreateProgram(ctx context.Context, trainerID string, input CreateProgramInput) (*Program, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateName(input.Name); err != nil {
		return nil, err
	}
	if err := validateDescription(input.Description); err != nil {
		return nil, err
	}
	if err := validateType(input.Type); err != nil {
		return nil, err
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = models.ProgramStatusDraft
	}
	if err := validateStatus(status); err != nil {
		return nil, err
	}

	// The price is never chosen by the trainer: it is the catalogue price of
	// the plan type (free, Level 1 price for premium, Level 2 price for
	// personalized).
	currency := string(models.ProgramCurrencyEUR)
	price := models.CatalogPriceMinorUnits(models.ProgramProductTypeGeneric, input.Type)
	if err := validatePriceForType(input.Type, price, currency, s.pricing.MinProgramPriceMinorUnits); err != nil {
		return nil, err
	}
	if err := program_details.ValidateCatalogMetadata(input.Level, input.TrainingType, input.DurationWeeks, input.FrequencyPerWeek); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if err := program_details.ValidateCoverKey(program_details.CoverFolder(models.ProgramProductTypeGeneric, input.Type), input.CoverImage); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	program := &models.Program{
		TrainerID:        trainerID,
		Name:             strings.TrimSpace(input.Name),
		Description:      input.Description,
		Type:             input.Type,
		Status:           status,
		PriceMinorUnits:  price,
		Currency:         currency,
		Level:            program_details.OptionalString(input.Level),
		DurationWeeks:    program_details.OptionalPositive(input.DurationWeeks),
		FrequencyPerWeek: program_details.OptionalPositive(input.FrequencyPerWeek),
		TrainingType:     program_details.OptionalString(input.TrainingType),
		CoverImage:       program_details.OptionalCover(input.CoverImage),
	}
	if err := s.programs.Create(ctx, program); err != nil {
		return nil, fmt.Errorf("failed to create program: %w", err)
	}

	return newProgram(program), nil
}

// ListPrograms returns one page of the authenticated trainer's active programs,
// ordered by creation time, plus the total count. The trainer id is always
// explicit; a client-supplied trainer_id can never change which programs are
// listed.
func (s *service) ListPrograms(ctx context.Context, trainerID string, page, limit int) (ListProgramsResult, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return ListProgramsResult{}, err
	}

	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListProgramsResult{}, err
	}

	models, total, err := s.programs.ListByTrainer(ctx, trainerID, page, limit)
	if err != nil {
		return ListProgramsResult{}, fmt.Errorf("failed to list programs: %w", err)
	}

	programs := make([]Program, 0, len(models))
	for i := range models {
		programs = append(programs, *newProgram(&models[i]))
	}
	return ListProgramsResult{Programs: programs, Total: total, Page: page, Limit: limit}, nil
}

// GetProgram returns one of the authenticated trainer's active programs. The
// query is scoped by both the owner trainer and the program id, so a program
// that is missing, soft-deleted or owned by another trainer is
// indistinguishable and never revealed.
func (s *service) GetProgram(ctx context.Context, trainerID, programID string) (*Program, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	model, err := s.programs.FindByIDAndTrainer(ctx, trainerID, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load program: %w", err)
	}

	program := newProgram(model)
	return program, nil
}

// UpdateProgram updates the whitelisted fields of one of the authenticated
// trainer's own programs. Ownership is verified before updating; trainer_id is
// immutable and can never be changed through this operation. At least one
// field must be provided.
func (s *service) UpdateProgram(ctx context.Context, trainerID, programID string, input UpdateProgramInput) (*Program, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	existing, err := s.programs.FindByIDAndTrainer(ctx, trainerID, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to verify program ownership: %w", err)
	}

	linked, delivered, err := s.premiumLevel2Link(ctx, programID)
	if err != nil {
		return nil, err
	}
	if delivered {
		return nil, ErrProgramLocked
	}
	if linked && input.Status != nil && *input.Status == models.ProgramStatusPublished {
		return nil, ErrProgramLocked
	}

	updates := make(map[string]any)
	if input.Name != nil {
		if err := validateName(*input.Name); err != nil {
			return nil, err
		}
		updates["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		if err := validateDescription(*input.Description); err != nil {
			return nil, err
		}
		updates["description"] = *input.Description
	}
	if input.Type != nil {
		if err := validateType(*input.Type); err != nil {
			return nil, err
		}
		updates["type"] = *input.Type
	}
	if input.Status != nil {
		if err := validateStatus(*input.Status); err != nil {
			return nil, err
		}
		updates["status"] = *input.Status
	}

	effectiveType := existing.Type
	if t, ok := updates["type"]; ok {
		effectiveType = t.(string)
	}

	level, trainingType := "", ""
	if existing.Level != nil {
		level = *existing.Level
	}
	if existing.TrainingType != nil {
		trainingType = *existing.TrainingType
	}
	duration, frequency := 0, 0
	if existing.DurationWeeks != nil {
		duration = *existing.DurationWeeks
	}
	if existing.FrequencyPerWeek != nil {
		frequency = *existing.FrequencyPerWeek
	}
	if input.Level != nil {
		level = *input.Level
		updates["level"] = program_details.OptionalString(level)
	}
	if input.TrainingType != nil {
		trainingType = *input.TrainingType
		updates["training_type"] = program_details.OptionalString(trainingType)
	}
	if input.DurationWeeks != nil {
		duration = *input.DurationWeeks
		updates["duration_weeks"] = program_details.OptionalPositive(duration)
	}
	if input.FrequencyPerWeek != nil {
		frequency = *input.FrequencyPerWeek
		updates["frequency_per_week"] = program_details.OptionalPositive(frequency)
	}
	if err := program_details.ValidateCatalogMetadata(level, trainingType, duration, frequency); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if input.CoverImage != nil {
		if err := program_details.ValidateCoverKey(program_details.CoverFolder(models.ProgramProductTypeGeneric, effectiveType), *input.CoverImage); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		updates["cover_image"] = program_details.OptionalCover(*input.CoverImage)
	}

	if len(updates) == 0 {
		return nil, fmt.Errorf("%w: at least one field must be provided", ErrInvalidInput)
	}

	// The price always follows the effective type's catalogue category; a
	// client-sent price or currency is ignored.
	if catalogPrice := models.CatalogPriceMinorUnits(models.ProgramProductTypeGeneric, effectiveType); catalogPrice != existing.PriceMinorUnits || existing.Currency != string(models.ProgramCurrencyEUR) {
		updates["price_minor_units"] = catalogPrice
		updates["currency"] = string(models.ProgramCurrencyEUR)
	}

	if err := s.programs.Update(ctx, trainerID, programID, updates); err != nil {
		return nil, fmt.Errorf("failed to update program: %w", err)
	}

	return s.GetProgram(ctx, trainerID, programID)
}

// PublishProgram transitions a draft program to published. The transition
// draft → published is the only allowed state change through this operation;
// publishing an already published program returns ErrProgramAlreadyPublished.
// A missing, soft-deleted or foreign program is indistinguishable and maps to
// ErrProgramNotFound. The client identity always comes from the authenticated
// trainer context.
func (s *service) PublishProgram(ctx context.Context, trainerID, programID string) (*Program, error) {
	if err := validateTrainerID(trainerID); err != nil {
		return nil, err
	}
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	linked, _, err := s.premiumLevel2Link(ctx, programID)
	if err != nil {
		return nil, err
	}
	if linked {
		return nil, ErrProgramLocked
	}

	// A plan saved before the catalogue prices existed is aligned with its
	// type's catalogue price before it becomes purchasable.
	if existing, findErr := s.programs.FindByIDAndTrainer(ctx, trainerID, programID); findErr == nil {
		if catalogPrice := models.CatalogPriceMinorUnits(models.ProgramProductTypeGeneric, existing.Type); existing.PriceMinorUnits != catalogPrice {
			if err := s.programs.Update(ctx, trainerID, programID, map[string]any{
				"price_minor_units": catalogPrice,
				"currency":          string(models.ProgramCurrencyEUR),
			}); err != nil {
				return nil, fmt.Errorf("failed to apply catalogue price: %w", err)
			}
		}
	}

	err = s.programs.Publish(ctx, trainerID, programID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrProgramNotFound):
			// The repository returns ErrProgramNotFound for missing,
			// soft-deleted, foreign and already-published programs. To
			// distinguish "already published" we reload the program: if it
			// exists and is owned by the trainer but has a non-draft status,
			// it was already published.
			existing, findErr := s.programs.FindByIDAndTrainer(ctx, trainerID, programID)
			if findErr != nil {
				return nil, ErrProgramNotFound
			}
			if existing.Status != models.ProgramStatusDraft {
				return nil, ErrProgramAlreadyPublished
			}
			return nil, ErrProgramNotFound
		default:
			return nil, fmt.Errorf("failed to publish program: %w", err)
		}
	}

	return s.GetProgram(ctx, trainerID, programID)
}

// DeleteProgram soft-deletes one of the authenticated trainer's own programs.
// Only the program row is touched and it is never removed from the database.
func (s *service) DeleteProgram(ctx context.Context, trainerID, programID string) error {
	if err := validateTrainerID(trainerID); err != nil {
		return err
	}
	if err := validateProgramID(programID); err != nil {
		return err
	}

	linked, _, err := s.premiumLevel2Link(ctx, programID)
	if err != nil {
		return err
	}
	if linked {
		return ErrProgramLocked
	}

	if err := s.programs.SoftDelete(ctx, trainerID, programID); err != nil {
		switch {
		case errors.Is(err, repositories.ErrProgramNotFound):
			return ErrProgramNotFound
		default:
			return fmt.Errorf("failed to delete program: %w", err)
		}
	}
	return nil
}

// UpdateProgramPricing updates the price of any active program. The caller
// (trainer owner or authorized administrator) is responsible for authorization
// before calling this method. Free programs must have a price of 0; paid
// programs must meet the configured minimum.
func (s *service) UpdateProgramPricing(ctx context.Context, programID string, input UpdatePricingInput) (*Program, error) {
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	program, err := s.programs.FindByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load program: %w", err)
	}

	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = program.Currency
	}
	if err := validateCurrency(currency); err != nil {
		return nil, err
	}
	// Prices are fixed by the catalogue category: the only accepted value is
	// the plan's catalogue price, which lets an administrator re-align a plan
	// saved before the catalogue prices existed, never choose a price.
	if input.PriceMinorUnits != models.CatalogPriceMinorUnits(program.ProductType, program.Type) {
		return nil, ErrPriceFixed
	}
	if err := validatePriceForType(program.Type, input.PriceMinorUnits, currency, s.pricing.MinProgramPriceMinorUnits); err != nil {
		return nil, err
	}

	if err := s.programs.UpdatePricing(ctx, programID, input.PriceMinorUnits, currency); err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to update program pricing: %w", err)
	}

	program.PriceMinorUnits = input.PriceMinorUnits
	program.Currency = currency
	return newProgram(program), nil
}

// GetProgramByID returns one active program by its id without ownership
// scoping. This is used by the admin pricing path.
func (s *service) GetProgramByID(ctx context.Context, programID string) (*Program, error) {
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	model, err := s.programs.FindByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load program: %w", err)
	}

	return newProgram(model), nil
}

func (s *service) premiumLevel2Link(ctx context.Context, programID string) (bool, bool, error) {
	reader, ok := s.programs.(premiumLevel2LinkReader)
	if !ok {
		return false, false, nil
	}
	linked, delivered, err := reader.PremiumLevel2Link(ctx, programID)
	if err != nil {
		return false, false, fmt.Errorf("failed to verify premium level 2 link: %w", err)
	}
	return linked, delivered, nil
}

func newProgram(model *models.Program) *Program {
	return &Program{
		ID:               model.ID,
		TrainerID:        model.TrainerID,
		Name:             model.Name,
		Description:      model.Description,
		Type:             model.Type,
		Status:           model.Status,
		PriceMinorUnits:  model.PriceMinorUnits,
		Currency:         model.Currency,
		Level:            model.Level,
		DurationWeeks:    model.DurationWeeks,
		FrequencyPerWeek: model.FrequencyPerWeek,
		TrainingType:     model.TrainingType,
		CoverImage:       model.CoverImage,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}

// normalizePagination validates the pagination parameters and clamps oversized
// limits to MaxPageSize.
func normalizePagination(page, limit int) (int, int, error) {
	if page < 1 {
		return 0, 0, fmt.Errorf("%w: page must be at least 1", ErrInvalidInput)
	}
	if limit < 1 {
		return 0, 0, fmt.Errorf("%w: limit must be at least 1", ErrInvalidInput)
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}
	return page, limit, nil
}

// validateProgramID rejects empty and malformed identifiers before any database
// access.
func validateProgramID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}

// validateTrainerID rejects empty and malformed identifiers before any database
// access.
func validateTrainerID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: trainer id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid trainer id", ErrInvalidInput)
	}
	return nil
}

// validateName rejects empty, blank and oversized program names.
func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if len([]rune(strings.TrimSpace(name))) > MaxNameLength {
		return fmt.Errorf("%w: name exceeds the maximum length", ErrInvalidInput)
	}
	return nil
}

// validateDescription rejects descriptions longer than the documented limit.
// An empty description is valid.
func validateDescription(description string) error {
	if len([]rune(description)) > MaxDescriptionLength {
		return fmt.Errorf("%w: description exceeds the maximum length", ErrInvalidInput)
	}
	return nil
}

// validateType rejects any program type outside the official product set.
func validateType(programType string) error {
	switch programType {
	case models.ProgramTypeFree, models.ProgramTypePremium, models.ProgramTypePersonalized:
		return nil
	default:
		return fmt.Errorf("%w: invalid program type", ErrInvalidInput)
	}
}

// validateStatus rejects any program status outside the official set.
func validateStatus(status string) error {
	switch status {
	case models.ProgramStatusDraft, models.ProgramStatusPublished:
		return nil
	default:
		return fmt.Errorf("%w: invalid program status", ErrInvalidInput)
	}
}

// validateCurrency rejects any currency code outside the supported set.
func validateCurrency(currency string) error {
	switch models.ProgramCurrency(currency) {
	case models.ProgramCurrencyEUR:
		return nil
	default:
		return fmt.Errorf("%w: unsupported currency", ErrInvalidInput)
	}
}

// validatePriceForType enforces the pricing rules: free programs must have a
// price of 0, while paid programs (premium/personalized) must meet the
// configured minimum.
func validatePriceForType(programType string, priceMinorUnits int64, currency string, minPrice int64) error {
	if err := validateCurrency(currency); err != nil {
		return err
	}

	switch programType {
	case models.ProgramTypeFree:
		if priceMinorUnits != 0 {
			return fmt.Errorf("%w: free programs must have a price of 0", ErrInvalidInput)
		}
	default:
		if priceMinorUnits < minPrice {
			return fmt.Errorf("%w: price must be at least %d minor units", ErrInvalidInput, minPrice)
		}
	}
	return nil
}
