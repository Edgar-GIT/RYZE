package exercises

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/admin_users"
)

var (
	// ErrInvalidInput indicates the exercise query was malformed.
	ErrInvalidInput = errors.New("invalid exercise input")
	// ErrExerciseNotFound indicates the exercise does not exist or is
	// soft-deleted.
	ErrExerciseNotFound = errors.New("exercise not found")
)

const (
	// MaxPageSize caps the number of exercises returned in a single page.
	MaxPageSize = admin_users.MaxPageSize
	// MaxSearchLength caps the length of the exercise search query.
	MaxSearchLength = 100
	// MaxFilterLength caps the length of any free-text browse filter value.
	MaxFilterLength = 255
)

// Difficulty vocabulary of the catalog. Values are stored verbatim and every
// filter value is validated against them.
var DifficultyValues = []string{"Beginner", "Intermediate", "Advanced"}

// MovementCategoryValues is the controlled vocabulary of the catalog's
// movement categories. Values are stored verbatim and every filter value is
// validated against them.
var MovementCategoryValues = []string{"Compound", "Isolation", "Core", "Cardio", "Plyometric", "Mobility", "Stretching"}

// BodyPartValues is the controlled vocabulary of the catalogue's body-part
// classification. It matches the vocabulary stored on imported catalogue
// records; entries without a classification never match a body-part filter.
var BodyPartValues = []string{"back", "cardio", "chest", "lower arms", "lower legs", "neck", "shoulders", "upper arms", "upper legs", "waist"}

// ExerciseRepository is the read-only data-access surface required by the
// exercises service. Writing the catalog is intentionally not part of it: the
// catalog is platform-owned and is populated outside this service.
type ExerciseRepository interface {
	FindByID(ctx context.Context, exerciseID string) (*models.Exercise, error)
	List(ctx context.Context, page, limit int) ([]models.Exercise, int64, error)
	Search(ctx context.Context, query string, page, limit int) ([]models.Exercise, int64, error)
	Library(ctx context.Context, filter repositories.ExerciseSearchFilter, page, limit int) ([]models.Exercise, int64, error)
	ListAlternatives(ctx context.Context, exerciseID string) ([]repositories.ExerciseAlternativeLink, error)
}

// Exercise is the safe representation of one exercise catalog entry. It
// carries only the public descriptive metadata and never exposes deletion
// markers, source identity or any internal data. Media fields are populated
// only when media exposure is enabled; otherwise they stay empty and
// MediaAvailable is false so clients can render an explicit unavailable state.
type Exercise struct {
	ID                    string
	Name                  string
	BodyPart              string
	Description           string
	Instructions          string
	InstructionSteps      []string
	TargetMuscles         string
	PrimaryMuscleGroup    string
	SecondaryMuscleGroups string
	Equipment             string
	Difficulty            string
	MovementCategory      string
	VideoURL              string
	ImageURL              string
	AnimationURL          string
	MediaAttribution      string
	MediaAvailable        bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ExerciseAlternative is the safe representation of one directed alternative
// link. AlternativeName is resolved from the global catalog at read time.
type ExerciseAlternative struct {
	ID                    string
	ExerciseID            string
	AlternativeExerciseID string
	AlternativeName       string
}

// ExerciseDetail is one catalog entry plus its curated alternatives.
type ExerciseDetail struct {
	Exercise     Exercise
	Alternatives []ExerciseAlternative
}

// ListExercisesResult carries one page of exercises plus the pagination
// metadata needed to render the list.
type ListExercisesResult struct {
	Exercises []Exercise
	Total     int64
	Page      int
	Limit     int
}

// Service exposes the public, read-only exercise catalog. The catalog is
// platform-owned: there is no trainer or admin ownership on this entity, the
// same data is served to every caller, and no write operation is exposed.
type Service interface {
	ListExercises(ctx context.Context, page, limit int) (ListExercisesResult, error)
	GetExercise(ctx context.Context, exerciseID string) (*ExerciseDetail, error)
	SearchExercises(ctx context.Context, query string, page, limit int) (ListExercisesResult, error)
	BrowseExercises(ctx context.Context, filter repositories.ExerciseSearchFilter, page, limit int) (ListExercisesResult, error)
}

type service struct {
	exercises ExerciseRepository
	media     config.ExerciseMediaConfig
}

// NewService builds the public catalogue service. media controls whether the
// catalogue exposes its media references: when disabled (the default) every
// media field stays empty and clients render an unavailable state, because the
// bundled dataset media licensing must be confirmed before public delivery.
func NewService(exercises ExerciseRepository, media config.ExerciseMediaConfig) Service {
	return &service{exercises: exercises, media: media}
}

// ListExercises returns one page of the catalog ordered alphabetically by name.
func (s *service) ListExercises(ctx context.Context, page, limit int) (ListExercisesResult, error) {
	return s.BrowseExercises(ctx, repositories.ExerciseSearchFilter{}, page, limit)
}

// GetExercise returns one active catalog entry together with its curated
// alternatives. Soft-deleted exercises are indistinguishable from missing ones.
func (s *service) GetExercise(ctx context.Context, exerciseID string) (*ExerciseDetail, error) {
	if err := validateExerciseID(exerciseID); err != nil {
		return nil, err
	}

	exerciseModel, err := s.exercises.FindByID(ctx, exerciseID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrExerciseNotFound):
			return nil, ErrExerciseNotFound
		default:
			return nil, fmt.Errorf("failed to find exercise: %w", err)
		}
	}

	alternativeLinks, err := s.exercises.ListAlternatives(ctx, exerciseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list exercise alternatives: %w", err)
	}

	return &ExerciseDetail{
		Exercise:     *s.toSafe(exerciseModel),
		Alternatives: toSafeAlternativeList(alternativeLinks),
	}, nil
}

// SearchExercises returns one page of catalog entries whose name contains the
// query, case-insensitively. It is kept as a convenience for callers that only
// match on free text and delegates to the full browse surface.
func (s *service) SearchExercises(ctx context.Context, query string, page, limit int) (ListExercisesResult, error) {
	if err := validateSearchQuery(query); err != nil {
		return ListExercisesResult{}, err
	}
	return s.BrowseExercises(ctx, repositories.ExerciseSearchFilter{Query: strings.TrimSpace(query)}, page, limit)
}

// BrowseExercises returns one page of catalog entries narrowed by the filter.
// Empty filter fields are ignored; vocabulary and length are validated here
// before any repository access.
func (s *service) BrowseExercises(ctx context.Context, filter repositories.ExerciseSearchFilter, page, limit int) (ListExercisesResult, error) {
	filter = normalizeFilter(filter)
	if err := validateFilter(filter); err != nil {
		return ListExercisesResult{}, err
	}
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListExercisesResult{}, err
	}

	exerciseModels, total, err := s.exercises.Library(ctx, filter, page, limit)
	if err != nil {
		return ListExercisesResult{}, fmt.Errorf("failed to browse exercises: %w", err)
	}

	return ListExercisesResult{
		Exercises: s.toSafeList(exerciseModels),
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

func (s *service) toSafe(model *models.Exercise) *Exercise {
	safe := &Exercise{
		ID:                    model.ID,
		Name:                  model.Name,
		BodyPart:              stringValue(model.BodyPart),
		Description:           model.Description,
		Instructions:          model.Instructions,
		InstructionSteps:      parseInstructionSteps(model.InstructionSteps),
		TargetMuscles:         model.TargetMuscles,
		PrimaryMuscleGroup:    model.PrimaryMuscleGroup,
		SecondaryMuscleGroups: model.SecondaryMuscleGroups,
		Equipment:             model.Equipment,
		Difficulty:            model.Difficulty,
		MovementCategory:      model.MovementCategory,
		VideoURL:              model.VideoURL,
		CreatedAt:             model.CreatedAt,
		UpdatedAt:             model.UpdatedAt,
	}
	s.applyMedia(safe, model)
	return safe
}

func (s *service) toSafeList(models []models.Exercise) []Exercise {
	list := make([]Exercise, 0, len(models))
	for i := range models {
		list = append(list, *s.toSafe(&models[i]))
	}
	return list
}

// applyMedia exposes the stored media references only when media exposure is
// enabled. The stored values are relative paths; the configured base path is
// prefixed here so clients never have to know the storage layout.
func (s *service) applyMedia(safe *Exercise, model *models.Exercise) {
	if !s.media.Enabled {
		return
	}
	if model.ImageURL != "" {
		safe.ImageURL = joinMediaPath(s.media.BasePath, model.ImageURL)
	}
	if model.AnimationURL != "" {
		safe.AnimationURL = joinMediaPath(s.media.BasePath, model.AnimationURL)
	}
	safe.MediaAttribution = model.MediaAttribution
	safe.MediaAvailable = safe.ImageURL != "" || safe.AnimationURL != ""
}

// joinMediaPath prefixes a stored relative media path with the configured base
// path. Both sides are normalized so a path can never produce a double slash.
func joinMediaPath(basePath, mediaPath string) string {
	return strings.TrimSuffix(basePath, "/") + "/" + strings.TrimPrefix(mediaPath, "/")
}

// stringValue dereferences an optional stored string, treating NULL as absent.
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// parseInstructionSteps decodes the stored JSON array of steps. A value that
// cannot be decoded is treated as absent so malformed stored data can never
// break a read response.
func parseInstructionSteps(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var steps []string
	if err := json.Unmarshal(raw, &steps); err != nil {
		return nil
	}
	return steps
}

// toSafeAlternativeList maps the repository links into safe values. Names are
// already resolved by the repository, so no extra lookup is needed here.
func toSafeAlternativeList(links []repositories.ExerciseAlternativeLink) []ExerciseAlternative {
	list := make([]ExerciseAlternative, 0, len(links))
	for i := range links {
		link := &links[i]
		list = append(list, ExerciseAlternative{
			ID:                    link.ID,
			ExerciseID:            link.ExerciseID,
			AlternativeExerciseID: link.AlternativeExerciseID,
			AlternativeName:       link.AlternativeName,
		})
	}
	return list
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

// validateExerciseID rejects empty and malformed identifiers before any
// database access.
func validateExerciseID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: exercise id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid exercise id", ErrInvalidInput)
	}
	return nil
}

// validateSearchQuery rejects empty, blank and oversized search queries.
func validateSearchQuery(query string) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("%w: search query is required", ErrInvalidInput)
	}
	if len([]rune(strings.TrimSpace(query))) > MaxSearchLength {
		return fmt.Errorf("%w: search query exceeds the maximum length", ErrInvalidInput)
	}
	return nil
}

// validateFilter rejects oversized free-text values and any difficulty,
// movement-category or body-part value outside the documented catalog
// vocabulary. Name queries keep the legacy search cap; the remaining filters
// use the general one.
func validateFilter(filter repositories.ExerciseSearchFilter) error {
	limits := map[string]int{
		"query":      MaxSearchLength,
		"muscle":     MaxFilterLength,
		"equipment":  MaxFilterLength,
		"difficulty": MaxFilterLength,
		"category":   MaxFilterLength,
		"body_part":  MaxFilterLength,
	}
	for label, value := range map[string]string{
		"query":      filter.Query,
		"muscle":     filter.Muscle,
		"equipment":  filter.Equipment,
		"difficulty": filter.Difficulty,
		"category":   filter.Category,
		"body_part":  filter.BodyPart,
	} {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if len([]rune(strings.TrimSpace(value))) > limits[label] {
			return fmt.Errorf("%w: filter %q exceeds the maximum length", ErrInvalidInput, label)
		}
	}
	if filter.Difficulty != "" && !contains(DifficultyValues, filter.Difficulty) {
		return fmt.Errorf("%w: invalid difficulty filter value", ErrInvalidInput)
	}
	if filter.Category != "" && !contains(MovementCategoryValues, filter.Category) {
		return fmt.Errorf("%w: invalid movement category filter value", ErrInvalidInput)
	}
	if filter.BodyPart != "" && !contains(BodyPartValues, filter.BodyPart) {
		return fmt.Errorf("%w: invalid body part filter value", ErrInvalidInput)
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// normalizeFilter trims every free-text filter value so leading and trailing
// whitespace can never alter the lookup.
func normalizeFilter(filter repositories.ExerciseSearchFilter) repositories.ExerciseSearchFilter {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Muscle = strings.TrimSpace(filter.Muscle)
	filter.Equipment = strings.TrimSpace(filter.Equipment)
	filter.Difficulty = strings.TrimSpace(filter.Difficulty)
	filter.Category = strings.TrimSpace(filter.Category)
	filter.BodyPart = strings.TrimSpace(filter.BodyPart)
	return filter
}
