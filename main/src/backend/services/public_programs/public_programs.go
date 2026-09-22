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
)

var (
	allowedSortFields = map[string]bool{
		"created_at": true,
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
	FindPublishedByIDWithStructure(ctx context.Context, programID string) (*models.Program, error)
	SearchPublished(ctx context.Context, filter repositories.PublicCatalogFilter, page, limit int) ([]models.Program, int64, error)
}

// ProgramFilter narrows the public program catalog. Every value is optional;
// empty values are ignored. ScopeGeneric restricts the catalog to
// platform-owned (trainer_id IS NULL) programs, which is the foundation of the
// generic training plans marketplace. sortBy is whitelisted to "created_at"
// and "name"; order is whitelisted to "asc" and "desc".
type ProgramFilter struct {
	Query            string
	ProgramType      string
	TrainingType     string
	Level            string
	FrequencyPerWeek int
	DurationMin      int
	DurationMax      int
	SortBy           string
	Order            string
	ScopeGeneric     bool
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
	PriceMinorUnits  int64
	Currency         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Set is the safe representation of one public prescription set.
type Set struct {
	SetNumber   int
	SetType     string
	Reps        *int
	WeightKg    *float64
	RIR         *int
	RPE         *float64
	RestSeconds *int
	Tempo       string
}

// Exercise is the safe catalog summary of the exercise assigned to a public
// workout. Internal fields are never exposed.
type Exercise struct {
	ID            string
	Name          string
	Description   string
	Instructions  string
	TargetMuscles string
	Equipment     string
	Difficulty    string
	VideoURL      string
	ImageURL      string
	Position      int
	Notes         string
	Sets          []Set
}

// Workout is the safe representation of one public workout slot.
type Workout struct {
	Position  int
	Exercises []Exercise
}

// Week is the safe representation of one public program week.
type Week struct {
	WeekNumber int
	Workouts   []Workout
}

// ProgramDetail is the safe representation of one published program with its
// complete client-safe structure.
type ProgramDetail struct {
	Program
	Weeks []Week
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
	GetPublishedProgram(ctx context.Context, programID string) (*ProgramDetail, error)
	// SearchPublishedPrograms returns published programs matching the optional
	// filters. An empty filter returns all published programs (equivalent to
	// ListPublishedPrograms). ScopeGeneric restricts the catalog to
	// platform-owned programs; sortBy and order control deterministic sorting
	// with safe defaults.
	SearchPublishedPrograms(ctx context.Context, filter ProgramFilter, page, limit int) (ListProgramsResult, error)
}

type service struct {
	programs ProgramRepository
}

func NewService(programs ProgramRepository) Service {
	return &service{programs: programs}
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

// GetPublishedProgram returns one published, non-deleted program with its full
// client-safe structure. A missing, draft or soft-deleted program maps to
// ErrProgramNotFound; there is no way to distinguish between the two.
func (s *service) GetPublishedProgram(ctx context.Context, programID string) (*ProgramDetail, error) {
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	model, err := s.programs.FindPublishedByIDWithStructure(ctx, programID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrProgramNotFound):
			return nil, ErrProgramNotFound
		default:
			return nil, fmt.Errorf("failed to find published program: %w", err)
		}
	}

	return toProgramDetail(model), nil
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

	filter.TrainingType = strings.TrimSpace(filter.TrainingType)
	if filter.TrainingType != "" && !allowedTrainingTypes[filter.TrainingType] {
		return ListProgramsResult{}, fmt.Errorf("%w: invalid training type", ErrInvalidInput)
	}

	filter.Level = strings.TrimSpace(filter.Level)
	if filter.Level != "" && !allowedLevels[filter.Level] {
		return ListProgramsResult{}, fmt.Errorf("%w: invalid audience level", ErrInvalidInput)
	}

	if filter.FrequencyPerWeek < 0 {
		return ListProgramsResult{}, fmt.Errorf("%w: frequency cannot be negative", ErrInvalidInput)
	}
	if filter.DurationMin < 0 || filter.DurationMax < 0 {
		return ListProgramsResult{}, fmt.Errorf("%w: duration cannot be negative", ErrInvalidInput)
	}

	if filter.SortBy != "" && !allowedSortFields[filter.SortBy] {
		filter.SortBy = ""
	}
	if filter.Order != "" && !allowedSortOrders[strings.ToLower(filter.Order)] {
		filter.Order = ""
	}

	models, total, err := s.programs.SearchPublished(ctx, repositories.PublicCatalogFilter{
		Query:            filter.Query,
		ProgramType:      filter.ProgramType,
		TrainingType:     filter.TrainingType,
		Level:            filter.Level,
		FrequencyPerWeek: filter.FrequencyPerWeek,
		DurationMin:      filter.DurationMin,
		DurationMax:      filter.DurationMax,
		SortBy:           filter.SortBy,
		Order:            filter.Order,
		ScopeGeneric:     filter.ScopeGeneric,
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
		PriceMinorUnits:  model.PriceMinorUnits,
		Currency:         model.Currency,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}
}

func toProgramDetail(model *models.Program) *ProgramDetail {
	detail := &ProgramDetail{
		Program: *toSafe(model),
		Weeks:   make([]Week, 0, len(model.Weeks)),
	}
	for i := range model.Weeks {
		week := &model.Weeks[i]
		detail.Weeks = append(detail.Weeks, Week{
			WeekNumber: week.WeekNumber,
			Workouts:   make([]Workout, 0, len(week.Workouts)),
		})
		for k := range week.Workouts {
			workout := &week.Workouts[k]
			safeWorkout := Workout{Position: workout.Position, Exercises: make([]Exercise, 0, len(workout.Exercises))}
			for j := range workout.Exercises {
				usage := &workout.Exercises[j]
				exercise := Exercise{
					Position: usage.Position,
					Notes:    usage.Notes,
					Sets:     make([]Set, 0, len(usage.Sets)),
				}
				if usage.Exercise != nil {
					exercise.ID = usage.Exercise.ID
					exercise.Name = usage.Exercise.Name
					exercise.Description = usage.Exercise.Description
					exercise.Instructions = usage.Exercise.Instructions
					exercise.TargetMuscles = usage.Exercise.TargetMuscles
					exercise.Equipment = usage.Exercise.Equipment
					exercise.Difficulty = usage.Exercise.Difficulty
					exercise.VideoURL = usage.Exercise.VideoURL
					exercise.ImageURL = usage.Exercise.ImageURL
				}
				for l := range usage.Sets {
					set := &usage.Sets[l]
					exercise.Sets = append(exercise.Sets, Set{
						SetNumber:   set.SetNumber,
						SetType:     set.SetType,
						Reps:        set.Reps,
						WeightKg:    set.WeightKg,
						RIR:         set.RIR,
						RPE:         set.RPE,
						RestSeconds: set.RestSeconds,
						Tempo:       set.Tempo,
					})
				}
				safeWorkout.Exercises = append(safeWorkout.Exercises, exercise)
			}
			detail.Weeks[len(detail.Weeks)-1].Workouts = append(detail.Weeks[len(detail.Weeks)-1].Workouts, safeWorkout)
		}
	}
	return detail
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
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}
