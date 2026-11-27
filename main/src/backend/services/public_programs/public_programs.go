package public_programs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
)

var (
	// ErrInvalidInput indicates the program query was malformed.
	ErrInvalidInput = errors.New("invalid program input")
	// ErrProgramNotFound indicates the program does not exist, is
	// soft-deleted, or is not published.
	ErrProgramNotFound = errors.New("program not found")
)

const (
	// MaxPageSize caps the number of programs returned in a single page.
	MaxPageSize = admin_users.MaxPageSize

	// MaxSearchQueryLength caps the maximum number of characters in a search
	// query to prevent excessive LIKE pattern matching.
	MaxSearchQueryLength = 200
	// MaxFilterValues bounds every repeated filter dimension so one request
	// cannot build an arbitrarily large SQL IN clause.
	MaxFilterValues = 20
)

// DurationRange is one inclusive duration filter in weeks. Multiple ranges are
// combined with OR.
type DurationRange = repositories.DurationRange

var (
	allowedSortFields = map[string]bool{
		"created_at": true,
		"updated_at": true,
		"name":       true,
	}
	allowedSortOrders = map[string]bool{
		"asc":  true,
		"desc": true,
	}
	allowedProgramTypes = map[string]bool{
		models.ProgramTypeFree:         true,
		models.ProgramTypePremium:      true,
		models.ProgramTypePersonalized: true,
	}
	allowedTrainingTypes = map[string]bool{
		"Hypertrophy": true,
		"Strength":    true,
		"HYROX":       true,
		"CrossFit":    true,
		"Fat Loss":    true,
		"At Home":     true,
	}
	allowedLevels = map[string]bool{
		"Beginner":     true,
		"Intermediate": true,
		"Advanced":     true,
	}
)

// ProgramRepository is the read-only data-access surface required by the
// public programs service. The catalog is global: there is no ownership
// scoping on this entity and no write operation is exposed.
type ProgramRepository interface {
	ListPublished(ctx context.Context, page, limit int) ([]models.Program, int64, error)
	FindPublishedByID(ctx context.Context, programID string) (*models.Program, error)
	SearchPublished(ctx context.Context, filter repositories.PublicCatalogFilter, page, limit int) ([]models.Program, int64, error)
}

// ProgramFilter narrows the public program catalog. Every value is optional;
// empty values are ignored. Slice values are OR within one dimension and AND
// across dimensions. ScopeGeneric restricts the catalog to platform-owned
// (trainer_id IS NULL) programs, which is the foundation of the generic
// training plans marketplace. sortBy is whitelisted to "created_at",
// "updated_at" and "name"; order is whitelisted to "asc" and "desc".
type ProgramFilter struct {
	Query              string
	ProgramType        string
	TrainingTypes      []string
	Levels             []string
	Frequencies        []int
	DurationMin        int
	DurationMax        int
	DurationRanges     []DurationRange
	SortBy             string
	Order              string
	ScopeGeneric       bool
	ScopePremiumLevel1 bool
	ScopePremiumLevel2 bool
	// Audience ("female" or "male") keeps the plans for everyone plus the
	// plans designed for that audience.
	Audience string
}

// Program is the safe representation of one published program. It carries
// only the public product metadata and never exposes deletion markers,
// draft programs, or any internal data.
type Program struct {
	ID               string
	TrainerID        string
	Name             string
	Description      string
	Type             string
	Status           string
	Level            *string
	DurationWeeks    *int
	FrequencyPerWeek *int
	TrainingType     *string
	CoverImage       *string
	Audience         *string
	// Archived: off the market; still open to the people who have it.
	Archived        bool
	PriceMinorUnits int64
	Currency        string
	// ProductType is the product family the program belongs to. It is public
	// product metadata: it tells the client which marketplace and which
	// onboarding flow the program belongs to.
	ProductType string
	// Objective is the curated Premium Level 1 goal. It is public catalogue
	// metadata and is nil for generic and legacy packages.
	Objective *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListProgramsResult carries one page of published programs plus the
// pagination metadata needed to render the list.
type ListProgramsResult struct {
	Programs []Program
	Total    int64
	Page     int
	Limit    int
}

// Service exposes the public, read-only program catalog. The catalog is
// global: every published program is visible to every caller, there is no
// ownership scoping, and no write operation is exposed.
type Service interface {
	ListPublishedPrograms(ctx context.Context, page, limit int) (ListProgramsResult, error)
	GetPublishedProgram(ctx context.Context, programID string) (*Program, error)
	// SearchPublishedPrograms returns published programs matching the optional
	// filters. An empty filter returns all published programs (equivalent to
	// ListPublishedPrograms). ScopeGeneric restricts the catalog to
	// platform-owned programs; sortBy and order control deterministic sorting
	// with safe defaults.
	SearchPublishedPrograms(ctx context.Context, filter ProgramFilter, page, limit int) (ListProgramsResult, error)
	// GetPublishedProgramOutline returns the shape of a published FREE
	// programme (owner request, 2026-11-20): its weeks and training days with
	// each day's title, focus and number of exercises, never the exercises.
	// Paid programmes keep their structure private (ErrProgramNotFound).
	GetPublishedProgramOutline(ctx context.Context, programID string) (*Outline, error)
}

// OutlineReader reads a programme's structure shape.
type OutlineReader interface {
	Outline(ctx context.Context, programID string) ([]repositories.ProgramOutlineDay, error)
}

// Outline is the public shape of a free programme.
type Outline struct {
	ProgramID      string
	Weeks          []OutlineWeek
	TotalDays      int
	TotalExercises int
}

// OutlineWeek groups a week's training days.
type OutlineWeek struct {
	WeekNumber int
	Days       []OutlineDay
}

// OutlineDay is one training day: its order, title, focus and exercise count.
type OutlineDay struct {
	Position      int
	Title         string
	Focus         *string
	ExerciseCount int
}

type service struct {
	programs ProgramRepository
	outlines OutlineReader
}

// Option configures an optional collaborator.
type Option func(*service)

// WithOutlines enables GetPublishedProgramOutline.
func WithOutlines(outlines OutlineReader) Option {
	return func(s *service) { s.outlines = outlines }
}

func NewService(programs ProgramRepository, opts ...Option) Service {
	s := &service{programs: programs}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *service) GetPublishedProgramOutline(ctx context.Context, programID string) (*Outline, error) {
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}
	model, err := s.programs.FindPublishedByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to find published program: %w", err)
	}
	// Only free programmes show their shape before access; a paid one answers
	// exactly like an unknown programme.
	if model.Type != models.ProgramTypeFree || s.outlines == nil {
		return nil, ErrProgramNotFound
	}
	days, err := s.outlines.Outline(ctx, model.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to read program outline: %w", err)
	}
	outline := &Outline{ProgramID: model.ID, Weeks: []OutlineWeek{}}
	for _, day := range days {
		if len(outline.Weeks) == 0 || outline.Weeks[len(outline.Weeks)-1].WeekNumber != day.WeekNumber {
			outline.Weeks = append(outline.Weeks, OutlineWeek{WeekNumber: day.WeekNumber})
		}
		week := &outline.Weeks[len(outline.Weeks)-1]
		week.Days = append(week.Days, OutlineDay{Position: day.Position, Title: day.Name, Focus: day.Focus, ExerciseCount: day.ExerciseCount})
		outline.TotalDays++
		outline.TotalExercises += day.ExerciseCount
	}
	return outline, nil
}

// ListPublishedPrograms returns one page of published programs ordered by
// creation time (newest first), plus the total count.
func (s *service) ListPublishedPrograms(ctx context.Context, page, limit int) (ListProgramsResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListProgramsResult{}, err
	}

	models, total, err := s.programs.ListPublished(ctx, page, limit)
	if err != nil {
		return ListProgramsResult{}, fmt.Errorf("failed to list published programs: %w", err)
	}

	programs := make([]Program, 0, len(models))
	for i := range models {
		programs = append(programs, *toSafe(&models[i]))
	}
	return ListProgramsResult{Programs: programs, Total: total, Page: page, Limit: limit}, nil
}

// GetPublishedProgram returns the public metadata of one published,
// non-deleted program. Programme structure is deliberately not part of the
// public response: week, workout, exercise and set content is only available
// through the authenticated access grant. A missing, draft or soft-deleted
// program maps to ErrProgramNotFound; there is no way to distinguish the two.
func (s *service) GetPublishedProgram(ctx context.Context, programID string) (*Program, error) {
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	model, err := s.programs.FindPublishedByID(ctx, programID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrProgramNotFound):
			return nil, ErrProgramNotFound
		default:
			return nil, fmt.Errorf("failed to find published program: %w", err)
		}
	}

	return toSafe(model), nil
}

// SearchPublishedPrograms returns published programs matching the optional
// catalog filters. The query is validated for length; the type, level and
// training-type values and the sort/order values are whitelisted. Repository
// errors are mapped to safe domain errors without exposing internal details.
func (s *service) SearchPublishedPrograms(ctx context.Context, filter ProgramFilter, page, limit int) (ListProgramsResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListProgramsResult{}, err
	}

	filter.Query = strings.TrimSpace(filter.Query)
	if len([]rune(filter.Query)) > MaxSearchQueryLength {
		return ListProgramsResult{}, fmt.Errorf("%w: search query exceeds maximum length", ErrInvalidInput)
	}

	filter.ProgramType = strings.TrimSpace(filter.ProgramType)
	if filter.ProgramType != "" && !allowedProgramTypes[filter.ProgramType] {
		return ListProgramsResult{}, fmt.Errorf("%w: invalid program type", ErrInvalidInput)
	}

	trainingTypes, err := normalizeAllowedValues(filter.TrainingTypes, allowedTrainingTypes)
	if err != nil {
		return ListProgramsResult{}, fmt.Errorf("%w: invalid training type", ErrInvalidInput)
	}

	levels, err := normalizeAllowedValues(filter.Levels, allowedLevels)
	if err != nil {
		return ListProgramsResult{}, fmt.Errorf("%w: invalid audience level", ErrInvalidInput)
	}

	frequencies, err := normalizeFrequencies(filter.Frequencies)
	if err != nil {
		return ListProgramsResult{}, err
	}

	if filter.DurationMin < 0 || filter.DurationMax < 0 {
		return ListProgramsResult{}, fmt.Errorf("%w: duration cannot be negative", ErrInvalidInput)
	}
	durationRanges, err := normalizeDurationRanges(filter.DurationRanges)
	if err != nil {
		return ListProgramsResult{}, err
	}

	filter.SortBy = strings.ToLower(strings.TrimSpace(filter.SortBy))
	if filter.SortBy != "" && !allowedSortFields[filter.SortBy] {
		filter.SortBy = ""
	}
	filter.Order = strings.ToLower(strings.TrimSpace(filter.Order))
	if filter.Order != "" && !allowedSortOrders[filter.Order] {
		filter.Order = ""
	}

	models, total, err := s.programs.SearchPublished(ctx, repositories.PublicCatalogFilter{
		Query:              filter.Query,
		ProgramType:        filter.ProgramType,
		TrainingTypes:      trainingTypes,
		Levels:             levels,
		Frequencies:        frequencies,
		DurationMin:        filter.DurationMin,
		DurationMax:        filter.DurationMax,
		DurationRanges:     durationRanges,
		SortBy:             filter.SortBy,
		Order:              filter.Order,
		ScopeGeneric:       filter.ScopeGeneric,
		ScopePremiumLevel1: filter.ScopePremiumLevel1,
		ScopePremiumLevel2: filter.ScopePremiumLevel2,
		Audience:           filter.Audience,
	}, page, limit)
	if err != nil {
		return ListProgramsResult{}, fmt.Errorf("failed to search published programs: %w", err)
	}

	programs := make([]Program, 0, len(models))
	for i := range models {
		programs = append(programs, *toSafe(&models[i]))
	}
	return ListProgramsResult{Programs: programs, Total: total, Page: page, Limit: limit}, nil
}

func toSafe(model *models.Program) *Program {
	return &Program{
		ID:               model.ID,
		TrainerID:        model.TrainerID,
		Name:             model.Name,
		Description:      model.Description,
		Type:             model.Type,
		Status:           model.Status,
		Level:            model.Level,
		DurationWeeks:    model.DurationWeeks,
		FrequencyPerWeek: model.FrequencyPerWeek,
		TrainingType:     model.TrainingType,
		CoverImage:       model.CoverImage,
		Audience:         model.Audience,
		Archived:         model.ArchivedAt != nil,
		PriceMinorUnits:  model.PriceMinorUnits,
		ProductType:      model.ProductType,
		Objective:        model.Objective,
		Currency:         model.Currency,
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

// normalizeAllowedValues trims, deduplicates and whitelists one filter
// dimension while preserving the caller's order.
func normalizeAllowedValues(values []string, allowed map[string]bool) ([]string, error) {
	if len(values) > MaxFilterValues {
		return nil, fmt.Errorf("%w: too many filter values", ErrInvalidInput)
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		if !allowed[value] {
			return nil, fmt.Errorf("%w: unsupported filter value", ErrInvalidInput)
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	return normalized, nil
}

// normalizeFrequencies validates the days-per-week dimension. The public
// catalogue contract allows exactly 1 to 7 days per week.
func normalizeFrequencies(values []int) ([]int, error) {
	if len(values) > MaxFilterValues {
		return nil, fmt.Errorf("%w: too many frequency values", ErrInvalidInput)
	}
	normalized := make([]int, 0, len(values))
	seen := make(map[int]bool, len(values))
	for _, value := range values {
		if value < 1 || value > 7 {
			return nil, fmt.Errorf("%w: frequency must be between 1 and 7", ErrInvalidInput)
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	return normalized, nil
}

// normalizeDurationRanges validates inclusive duration ranges. Ranges are ORed
// so the marketplace can express disjoint options.
func normalizeDurationRanges(ranges []DurationRange) ([]DurationRange, error) {
	if len(ranges) > MaxFilterValues {
		return nil, fmt.Errorf("%w: too many duration ranges", ErrInvalidInput)
	}
	normalized := make([]DurationRange, 0, len(ranges))
	seen := make(map[DurationRange]bool, len(ranges))
	for _, durationRange := range ranges {
		if durationRange.MinWeeks < 1 || durationRange.MaxWeeks < durationRange.MinWeeks {
			return nil, fmt.Errorf("%w: duration range must be positive and ordered", ErrInvalidInput)
		}
		if seen[durationRange] {
			continue
		}
		seen[durationRange] = true
		normalized = append(normalized, durationRange)
	}
	return normalized, nil
}

// validateProgramID rejects empty and malformed identifiers before any database
// access.
func validateProgramID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}
