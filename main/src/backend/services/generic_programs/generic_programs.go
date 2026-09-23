package generic_programs

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
)

var (
	// ErrInvalidInput indicates the generic program input was malformed or
	// incomplete.
	ErrInvalidInput = errors.New("invalid generic program input")
	// ErrProgramNotFound indicates the generic program does not exist or is
	// soft-deleted.
	ErrProgramNotFound = errors.New("generic program not found")
	// ErrExerciseNotFound indicates a referenced catalogue exercise does not
	// exist or is soft-deleted in the global catalog and can therefore never be
	// assigned.
	ErrExerciseNotFound = errors.New("exercise not found")
	// ErrDuplicateExercise indicates the same catalogue exercise is assigned
	// more than once inside the same workout.
	ErrDuplicateExercise = errors.New("duplicate exercise in workout")
	// ErrProgramAlreadyPublished indicates the generic program is already in
	// the published state and cannot be published again.
	ErrProgramAlreadyPublished = errors.New("generic program already published")
)

const (
	// MaxPageSize caps the number of generic programs returned in a single page.
	MaxPageSize = admin_users.MaxPageSize
	// MaxNameLength caps the program name length, matching the database column.
	MaxNameLength = 255
	// MaxDescriptionLength caps the program description length.
	MaxDescriptionLength = 5000
	// MaxSearchLength caps the search query length.
	MaxSearchLength = 100
	// MaxWeeksPerProgram caps the number of weeks of a generic program.
	MaxWeeksPerProgram = 52
	// MaxWorkoutsPerWeek caps the number of workouts (training days) inside a
	// single week.
	MaxWorkoutsPerWeek = 14
	// MaxExercisesPerWorkout caps the number of workout exercises.
	MaxExercisesPerWorkout = 50
	// MaxSetsPerExercise caps the number of prescription sets per workout
	// exercise.
	MaxSetsPerExercise = 50
	// MaxTempoLength caps the tempo notation length.
	MaxTempoLength = 20
)

// LevelValues is the controlled audience vocabulary of generic programs.
// Values are stored verbatim and every filter value is validated against them.
var LevelValues = []string{"Beginner", "Intermediate", "Advanced"}

// TrainingTypeValues is the controlled marketplace vocabulary of generic
// programs. Values are stored verbatim and every filter value is validated
// against them; an empty value means the program does not advertise a specific
// training type.
var TrainingTypeValues = []string{"Hypertrophy", "Strength", "HYROX", "CrossFit", "Fat Loss", "At Home"}

// SetTypeValues is the controlled set-type vocabulary, matching the database
// CHECK constraint and the builder's prescription editor.
var SetTypeValues = []string{"warmup", "working", "drop", "backoff", "failure"}

// GenericProgramRepository is the data-access surface required by the generic
// program service. It is the platform-owned counterpart of the trainer program
// repository and never carries a trainer id.
type GenericProgramRepository interface {
	CreateFull(ctx context.Context, program *models.Program) error
	UpdateFull(ctx context.Context, programID string, program *models.Program) error
	FindByID(ctx context.Context, programID string) (*models.Program, error)
	Search(ctx context.Context, filter repositories.GenericProgramFilter, page, limit int) ([]models.Program, int64, error)
	SoftDelete(ctx context.Context, programID string) error
	Publish(ctx context.Context, programID string) error
}

// SetInput carries one prescription set of a workout exercise.
type SetInput struct {
	SetType     string
	Reps        *int
	WeightKg    *float64
	RIR         *int
	RPE         *float64
	RestSeconds *int
	Tempo       string
}

// ExerciseInput carries one exercise usage inside a workout, together with its
// per-assignment instructions, notes and prescription sets.
type ExerciseInput struct {
	ExerciseID   string
	Instructions string
	Notes        string
	Prescription []SetInput
}

// WorkoutInput carries one workout slot of a week with its exercise usages.
type WorkoutInput struct {
	Position  int
	Exercises []ExerciseInput
}

// WeekInput carries one week of the program with its workouts.
type WeekInput struct {
	WeekNumber int
	Workouts   []WorkoutInput
}

// ProgramInput is the full desired state of a generic program, used by both
// create and update. Level is empty when unset; DurationWeeks and
// FrequencyPerWeek are 0 when unset. String coercion of client identifiers is
// deliberately absent: the hierarchy is always addressed by its server-owned
// structural keys.
type ProgramInput struct {
	Name             string
	Description      string
	Type             string
	Status           string
	Level            string
	DurationWeeks    int
	FrequencyPerWeek int
	TrainingType     string
	PriceMinorUnits  int64
	Currency         string
	Weeks            []WeekInput
}

// ProgramFilter narrows the generic program list.
type ProgramFilter struct {
	Query            string
	Type             string
	Level            string
	DurationWeeks    int
	FrequencyPerWeek int
	TrainingType     string
}

// Set is the safe representation of one prescription set. Optional training
// fields remain nil when unset.
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

// Exercise is the safe catalog summary of the platform-owned exercise assigned
// to a workout.
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
}

// WorkoutExercise is the safe representation of one exercise usage with its
// prescription sets.
type WorkoutExercise struct {
	Position     int
	Instructions string
	Notes        string
	Exercise     Exercise
	Sets         []Set
}

// Workout is the safe representation of one workout slot in position order.
type Workout struct {
	Position  int
	Exercises []WorkoutExercise
}

// Week is the safe representation of one program week in week order.
type Week struct {
	WeekNumber int
	Workouts   []Workout
}

// Program is the safe metadata summary of a generic program. Platform ownership
// is implicit: the owning trainer is never exposed.
type Program struct {
	ID               string
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

// ProgramDetail is the full safe representation of a generic program including
// its complete structure.
type ProgramDetail struct {
	Program
	Weeks []Week
}

// ListProgramsResult carries one page of generic programs plus the pagination
// metadata needed to render the list.
type ListProgramsResult struct {
	Programs []Program
	Total    int64
	Page     int
	Limit    int
}

// Service implements the platform-owned generic program flow. Authorization
// (which administrator may operate) is enforced by the route middleware; the
// platform ownership itself is guaranteed because every operation is scoped to
// the generic (trainer_id IS NULL) state and never accepts a trainer id. This
// service never knows about HTTP, Gin or the admin context.
type Service interface {
	CreateProgram(ctx context.Context, input ProgramInput) (*ProgramDetail, error)
	ListPrograms(ctx context.Context, filter ProgramFilter, page, limit int) (ListProgramsResult, error)
	GetProgram(ctx context.Context, programID string) (*ProgramDetail, error)
	UpdateProgram(ctx context.Context, programID string, input ProgramInput) (*ProgramDetail, error)
	PublishProgram(ctx context.Context, programID string) (Program, error)
	DeleteProgram(ctx context.Context, programID string) error
}

type service struct {
	programs GenericProgramRepository
	pricing  config.PricingConfig
}

func NewService(programs GenericProgramRepository, pricing config.PricingConfig) Service {
	return &service{programs: programs, pricing: pricing}
}

// CreateProgram creates a new platform-owned generic program together with its
// full structure in a single transaction. A client-supplied trainer id is never
// accepted: the program is always created with trainer_id NULL.
func (s *service) CreateProgram(ctx context.Context, input ProgramInput) (*ProgramDetail, error) {
	if err := s.validateProgramInput(input); err != nil {
		return nil, err
	}

	program := buildProgramModel(input)
	if err := s.programs.CreateFull(ctx, program); err != nil {
		if errors.Is(err, repositories.ErrExerciseNotFound) {
			return nil, ErrExerciseNotFound
		}
		return nil, fmt.Errorf("failed to create generic program: %w", err)
	}
	return s.GetProgram(ctx, program.ID)
}

// ListPrograms returns one page of active generic programs narrowed by the
// filter, plus the pagination metadata.
func (s *service) ListPrograms(ctx context.Context, filter ProgramFilter, page, limit int) (ListProgramsResult, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return ListProgramsResult{}, err
	}
	filter, err = s.normalizeFilter(filter)
	if err != nil {
		return ListProgramsResult{}, err
	}

	models, total, err := s.programs.Search(ctx, repositories.GenericProgramFilter{
		Query:            filter.Query,
		Type:             filter.Type,
		Level:            filter.Level,
		DurationWeeks:    filter.DurationWeeks,
		FrequencyPerWeek: filter.FrequencyPerWeek,
	}, page, limit)
	if err != nil {
		return ListProgramsResult{}, fmt.Errorf("failed to list generic programs: %w", err)
	}

	programs := make([]Program, 0, len(models))
	for i := range models {
		programs = append(programs, newProgramSummary(&models[i]))
	}
	return ListProgramsResult{Programs: programs, Total: total, Page: page, Limit: limit}, nil
}

// GetProgram returns one active generic program with its complete structure. A
// missing or soft-deleted program maps to ErrProgramNotFound.
func (s *service) GetProgram(ctx context.Context, programID string) (*ProgramDetail, error) {
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}

	model, err := s.programs.FindByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrGenericProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, fmt.Errorf("failed to load generic program: %w", err)
	}
	return newProgramDetail(model), nil
}

// UpdateProgram reconciles an existing generic program with the full desired
// structure in a single transaction, preserving the ids of reused structural
// rows.
func (s *service) UpdateProgram(ctx context.Context, programID string, input ProgramInput) (*ProgramDetail, error) {
	if err := validateProgramID(programID); err != nil {
		return nil, err
	}
	if err := s.validateProgramInput(input); err != nil {
		return nil, err
	}

	program := buildProgramModel(input)
	if err := s.programs.UpdateFull(ctx, programID, program); err != nil {
		if errors.Is(err, repositories.ErrGenericProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		if errors.Is(err, repositories.ErrExerciseNotFound) {
			return nil, ErrExerciseNotFound
		}
		return nil, fmt.Errorf("failed to update generic program: %w", err)
	}
	return s.GetProgram(ctx, programID)
}

// PublishProgram transitions a draft generic program to published. The
// transition draft → published is the only allowed state change; publishing an
// already published program returns ErrProgramAlreadyPublished.
func (s *service) PublishProgram(ctx context.Context, programID string) (Program, error) {
	if err := validateProgramID(programID); err != nil {
		return Program{}, err
	}

	model, err := s.programs.FindByID(ctx, programID)
	if err != nil {
		if errors.Is(err, repositories.ErrGenericProgramNotFound) {
			return Program{}, ErrProgramNotFound
		}
		return Program{}, fmt.Errorf("failed to load generic program: %w", err)
	}
	if model.Status != models.ProgramStatusDraft {
		return Program{}, ErrProgramAlreadyPublished
	}

	// Publishing exposes the program as a purchasable marketplace entry, so
	// the commercial gate is re-checked at publish time even though create and
	// update already validated it. This guarantees a published purchasable
	// program can never carry an invalid price.
	if err := validatePriceForType(model.Type, model.PriceMinorUnits, model.Currency, s.pricing.MinProgramPriceMinorUnits); err != nil {
		return Program{}, err
	}

	err = s.programs.Publish(ctx, programID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrGenericProgramNotFound):
			// The repository returns ErrGenericProgramNotFound for missing,
			// soft-deleted and already-published programs. Our pre-check
			// already confirmed this is a draft; a concurrent change between
			// the pre-check and the publish is reported as a conflict.
			return Program{}, ErrProgramAlreadyPublished
		default:
			return Program{}, fmt.Errorf("failed to publish generic program: %w", err)
		}
	}

	model.Status = models.ProgramStatusPublished
	return newProgramSummary(model), nil
}

// DeleteProgram soft-deletes one active generic program. Only the program row
// is touched and it is never removed from the database.
func (s *service) DeleteProgram(ctx context.Context, programID string) error {
	if err := validateProgramID(programID); err != nil {
		return err
	}

	if err := s.programs.SoftDelete(ctx, programID); err != nil {
		if errors.Is(err, repositories.ErrGenericProgramNotFound) {
			return ErrProgramNotFound
		}
		return fmt.Errorf("failed to delete generic program: %w", err)
	}
	return nil
}

// validateProgramInput enforces the documented generic program invariants:
// required and bounded metadata, controlled vocabularies, contiguous structural
// ordering and the duplicate-exercise rule. Structural keys (week_number,
// position, set_number) are contiguous and start at 1.
func (s *service) validateProgramInput(input ProgramInput) error {
	if err := validateName(input.Name); err != nil {
		return err
	}
	if err := validateDescription(input.Description); err != nil {
		return err
	}
	if err := validateType(input.Type); err != nil {
		return err
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = models.ProgramStatusDraft
	}
	if err := validateStatus(status); err != nil {
		return err
	}

	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = string(models.ProgramCurrencyEUR)
	}
	if err := validateCurrency(currency); err != nil {
		return err
	}
	if err := validatePriceForType(input.Type, input.PriceMinorUnits, currency, s.pricing.MinProgramPriceMinorUnits); err != nil {
		return err
	}

	if err := validateLevel(input.Level); err != nil {
		return err
	}
	if err := validateTrainingType(input.TrainingType); err != nil {
		return err
	}
	if err := validateDuration(input.DurationWeeks); err != nil {
		return err
	}
	if err := validateFrequency(input.FrequencyPerWeek); err != nil {
		return err
	}

	if len(input.Weeks) < 1 {
		return fmt.Errorf("%w: a generic program requires at least one week", ErrInvalidInput)
	}
	if len(input.Weeks) > MaxWeeksPerProgram {
		return fmt.Errorf("%w: a generic program is capped at %d weeks", ErrInvalidInput, MaxWeeksPerProgram)
	}

	seenExercises := make(map[string]int)
	for i := range input.Weeks {
		week := &input.Weeks[i]
		if week.WeekNumber != i+1 {
			return fmt.Errorf("%w: week numbers must be contiguous starting at 1", ErrInvalidInput)
		}
		if len(week.Workouts) > MaxWorkoutsPerWeek {
			return fmt.Errorf("%w: a week is capped at %d workouts", ErrInvalidInput, MaxWorkoutsPerWeek)
		}
		for k := range week.Workouts {
			workout := &week.Workouts[k]
			if workout.Position != k+1 {
				return fmt.Errorf("%w: workout positions must be contiguous starting at 1", ErrInvalidInput)
			}
			if len(workout.Exercises) > MaxExercisesPerWorkout {
				return fmt.Errorf("%w: a workout is capped at %d exercises", ErrInvalidInput, MaxExercisesPerWorkout)
			}
			used := make(map[string]struct{}, len(workout.Exercises))
			for j := range workout.Exercises {
				exercise := &workout.Exercises[j]
				if err := validateExerciseID(exercise.ExerciseID); err != nil {
					return err
				}
				if _, exists := used[exercise.ExerciseID]; exists {
					return fmt.Errorf("%w: exercise %q assigned more than once in workout %d", ErrDuplicateExercise, exercise.ExerciseID, workout.Position)
				}
				used[exercise.ExerciseID] = struct{}{}
				seenExercises[exercise.ExerciseID]++
				if len(exercise.Prescription) > MaxSetsPerExercise {
					return fmt.Errorf("%w: an exercise is capped at %d sets", ErrInvalidInput, MaxSetsPerExercise)
				}
				for l := range exercise.Prescription {
					if err := validateSetInput(&exercise.Prescription[l], l+1); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// normalizeFilter trims the free-text filter and validates vocabulary values
// before any repository access.
func (s *service) normalizeFilter(filter ProgramFilter) (ProgramFilter, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	if len([]rune(filter.Query)) > MaxSearchLength {
		return ProgramFilter{}, fmt.Errorf("%w: search query exceeds the maximum length", ErrInvalidInput)
	}
	filter.Type = strings.TrimSpace(filter.Type)
	if filter.Type != "" {
		if err := validateType(filter.Type); err != nil {
			return ProgramFilter{}, err
		}
	}
	filter.Level = strings.TrimSpace(filter.Level)
	if filter.Level != "" {
		if err := validateLevel(filter.Level); err != nil {
			return ProgramFilter{}, err
		}
	}
	filter.TrainingType = strings.TrimSpace(filter.TrainingType)
	if filter.TrainingType != "" {
		if err := validateTrainingType(filter.TrainingType); err != nil {
			return ProgramFilter{}, err
		}
	}
	if filter.DurationWeeks < 0 || filter.FrequencyPerWeek < 0 {
		return ProgramFilter{}, fmt.Errorf("%w: filter values cannot be negative", ErrInvalidInput)
	}
	return filter, nil
}

// buildProgramModel maps the validated input into the model tree handed to the
// repository. Optional fields map to nil pointers so the nullable columns stay
// NULL when unset.
func buildProgramModel(input ProgramInput) *models.Program {
	var level *string
	if strings.TrimSpace(input.Level) != "" {
		v := strings.TrimSpace(input.Level)
		level = &v
	}
	var duration *int
	if input.DurationWeeks > 0 {
		v := input.DurationWeeks
		duration = &v
	}
	var frequency *int
	if input.FrequencyPerWeek > 0 {
		v := input.FrequencyPerWeek
		frequency = &v
	}
	var trainingType *string
	if strings.TrimSpace(input.TrainingType) != "" {
		v := strings.TrimSpace(input.TrainingType)
		trainingType = &v
	}

	program := &models.Program{
		Name:             strings.TrimSpace(input.Name),
		Description:      input.Description,
		Type:             input.Type,
		Status:           strings.TrimSpace(input.Status),
		Level:            level,
		DurationWeeks:    duration,
		FrequencyPerWeek: frequency,
		TrainingType:     trainingType,
		PriceMinorUnits:  input.PriceMinorUnits,
		Currency:         strings.TrimSpace(input.Currency),
	}
	if program.Status == "" {
		program.Status = models.ProgramStatusDraft
	}
	if program.Currency == "" {
		program.Currency = string(models.ProgramCurrencyEUR)
	}

	program.Weeks = make([]models.ProgramWeek, 0, len(input.Weeks))
	for i := range input.Weeks {
		week := &input.Weeks[i]
		modelWeek := models.ProgramWeek{WeekNumber: week.WeekNumber, Workouts: make([]models.ProgramWorkout, 0, len(week.Workouts))}
		for k := range week.Workouts {
			workout := &week.Workouts[k]
			modelWorkout := models.ProgramWorkout{Position: workout.Position, Exercises: make([]models.WorkoutExercise, 0, len(workout.Exercises))}
			for j := range workout.Exercises {
				exercise := &workout.Exercises[j]
				modelExercise := models.WorkoutExercise{
					ExerciseID:   exercise.ExerciseID,
					Position:     j + 1,
					Instructions: exercise.Instructions,
					Notes:        exercise.Notes,
					Sets:         make([]models.WorkoutExerciseSet, 0, len(exercise.Prescription)),
				}
				for l := range exercise.Prescription {
					set := &exercise.Prescription[l]
					modelExercise.Sets = append(modelExercise.Sets, models.WorkoutExerciseSet{
						SetNumber:   l + 1,
						SetType:     set.SetType,
						Reps:        set.Reps,
						WeightKg:    set.WeightKg,
						RIR:         set.RIR,
						RPE:         set.RPE,
						RestSeconds: set.RestSeconds,
						Tempo:       set.Tempo,
					})
				}
				modelWorkout.Exercises = append(modelWorkout.Exercises, modelExercise)
			}
			modelWeek.Workouts = append(modelWeek.Workouts, modelWorkout)
		}
		program.Weeks = append(program.Weeks, modelWeek)
	}
	return program
}

func newProgramSummary(model *models.Program) Program {
	return Program{
		ID:               model.ID,
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

func newProgramDetail(model *models.Program) *ProgramDetail {
	detail := &ProgramDetail{
		Program: newProgramSummary(model),
		Weeks:   make([]Week, 0, len(model.Weeks)),
	}
	for i := range model.Weeks {
		week := &model.Weeks[i]
		detail.Weeks = append(detail.Weeks, newWeek(week))
	}
	return detail
}

func newWeek(model *models.ProgramWeek) Week {
	week := Week{WeekNumber: model.WeekNumber, Workouts: make([]Workout, 0, len(model.Workouts))}
	for i := range model.Workouts {
		week.Workouts = append(week.Workouts, newWorkout(&model.Workouts[i]))
	}
	return week
}

func newWorkout(model *models.ProgramWorkout) Workout {
	workout := Workout{Position: model.Position, Exercises: make([]WorkoutExercise, 0, len(model.Exercises))}
	for i := range model.Exercises {
		workout.Exercises = append(workout.Exercises, newWorkoutExercise(&model.Exercises[i]))
	}
	return workout
}

func newWorkoutExercise(model *models.WorkoutExercise) WorkoutExercise {
	sets := make([]Set, 0, len(model.Sets))
	for i := range model.Sets {
		set := &model.Sets[i]
		sets = append(sets, Set{
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

	exercise := Exercise{ID: model.ExerciseID}
	if model.Exercise != nil {
		exercise = Exercise{
			ID:            model.Exercise.ID,
			Name:          model.Exercise.Name,
			Description:   model.Exercise.Description,
			Instructions:  model.Exercise.Instructions,
			TargetMuscles: model.Exercise.TargetMuscles,
			Equipment:     model.Exercise.Equipment,
			Difficulty:    model.Exercise.Difficulty,
			VideoURL:      model.Exercise.VideoURL,
			ImageURL:      model.Exercise.ImageURL,
		}
	}
	return WorkoutExercise{
		Position:     model.Position,
		Instructions: model.Instructions,
		Notes:        model.Notes,
		Exercise:     exercise,
		Sets:         sets,
	}
}

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

func validateProgramID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: program id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid program id", ErrInvalidInput)
	}
	return nil
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if len([]rune(strings.TrimSpace(name))) > MaxNameLength {
		return fmt.Errorf("%w: name exceeds the maximum length", ErrInvalidInput)
	}
	return nil
}

func validateDescription(description string) error {
	if len([]rune(description)) > MaxDescriptionLength {
		return fmt.Errorf("%w: description exceeds the maximum length", ErrInvalidInput)
	}
	return nil
}

func validateType(programType string) error {
	switch programType {
	case models.ProgramTypeFree, models.ProgramTypePremium, models.ProgramTypePersonalized:
		return nil
	default:
		return fmt.Errorf("%w: invalid program type", ErrInvalidInput)
	}
}

func validateStatus(status string) error {
	switch status {
	case models.ProgramStatusDraft, models.ProgramStatusPublished:
		return nil
	default:
		return fmt.Errorf("%w: invalid program status", ErrInvalidInput)
	}
}

func validateCurrency(currency string) error {
	switch models.ProgramCurrency(currency) {
	case models.ProgramCurrencyEUR:
		return nil
	default:
		return fmt.Errorf("%w: unsupported currency", ErrInvalidInput)
	}
}

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

func validateLevel(level string) error {
	if strings.TrimSpace(level) == "" {
		return nil
	}
	for _, candidate := range LevelValues {
		if level == candidate {
			return nil
		}
	}
	return fmt.Errorf("%w: invalid audience level", ErrInvalidInput)
}

func validateTrainingType(trainingType string) error {
	if strings.TrimSpace(trainingType) == "" {
		return nil
	}
	for _, candidate := range TrainingTypeValues {
		if trainingType == candidate {
			return nil
		}
	}
	return fmt.Errorf("%w: invalid training type", ErrInvalidInput)
}

func validateDuration(duration int) error {
	if duration == 0 {
		return nil
	}
	if duration < 0 || duration > MaxWeeksPerProgram {
		return fmt.Errorf("%w: duration_weeks must be between 1 and %d", ErrInvalidInput, MaxWeeksPerProgram)
	}
	return nil
}

func validateFrequency(frequency int) error {
	if frequency == 0 {
		return nil
	}
	if frequency < 0 || frequency > MaxWorkoutsPerWeek {
		return fmt.Errorf("%w: frequency_per_week must be between 1 and %d", ErrInvalidInput, MaxWorkoutsPerWeek)
	}
	return nil
}

func validateExerciseID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: exercise id is required", ErrInvalidInput)
	}
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%w: invalid exercise id", ErrInvalidInput)
	}
	return nil
}

func validateSetInput(set *SetInput, expectedNumber int) error {
	if set.SetType != "" {
		valid := false
		for _, candidate := range SetTypeValues {
			if set.SetType == candidate {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("%w: invalid set type", ErrInvalidInput)
		}
	}
	if set.Reps != nil && *set.Reps < 1 {
		return fmt.Errorf("%w: reps must be at least 1", ErrInvalidInput)
	}
	if set.WeightKg != nil && *set.WeightKg < 0 {
		return fmt.Errorf("%w: weight cannot be negative", ErrInvalidInput)
	}
	if set.RIR != nil && *set.RIR < 0 {
		return fmt.Errorf("%w: rir cannot be negative", ErrInvalidInput)
	}
	if set.RPE != nil && (*set.RPE < 1.0 || *set.RPE > 10.0) {
		return fmt.Errorf("%w: rpe must be between 1 and 10", ErrInvalidInput)
	}
	if set.RestSeconds != nil && *set.RestSeconds < 0 {
		return fmt.Errorf("%w: rest cannot be negative", ErrInvalidInput)
	}
	if len([]rune(set.Tempo)) > MaxTempoLength {
		return fmt.Errorf("%w: tempo exceeds the maximum length", ErrInvalidInput)
	}
	return nil
}
