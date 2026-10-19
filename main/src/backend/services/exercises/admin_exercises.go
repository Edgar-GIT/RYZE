package exercises

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

// ErrExerciseConflict indicates the catalogue entry cannot be created or
// updated because another active entry already uses the requested name.
var ErrExerciseConflict = errors.New("exercise conflict")

const (
	maxExerciseNameLength             = 255
	maxExerciseDescriptionLength      = 2000
	maxExerciseInstructionsLength     = 5000
	maxExerciseSteps                  = 50
	maxExerciseStepLength             = 500
	maxExerciseTargetLength           = 255
	maxExercisePrimaryMuscleLength    = 80
	maxExerciseSecondaryMusclesLength = 255
	maxExerciseEquipmentLength        = 255
	maxExerciseDifficultyLength       = 50
	maxExerciseMovementLength         = 60
	maxExerciseBodyPartLength         = 40
	maxExerciseMediaPathLength        = 500
	maxExerciseSourceMediaIDLength    = 40
	maxExerciseAttributionLength      = 255
)

// AdminExercise is the administrative representation of one catalogue entry.
// Unlike the public representation it includes provenance, publication status
// and the raw stored media paths, which administrators need to curate the
// catalogue.
type AdminExercise struct {
	ID                    string
	Name                  string
	Source                string
	SourceID              string
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
	SourceMediaID         string
	MediaAttribution      string
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// AdminExerciseListResult carries one page of administrative catalogue
// entries plus the pagination metadata needed to render the list.
type AdminExerciseListResult struct {
	Exercises []AdminExercise
	Total     int64
	Page      int
	Limit     int
}

// AdminCreateExerciseInput carries the editable fields of a new catalogue
// entry. Every field except the name is optional; entries start as drafts and
// must be published explicitly.
type AdminCreateExerciseInput struct {
	Name                  string
	Description           string
	Instructions          string
	InstructionSteps      []string
	TargetMuscles         string
	PrimaryMuscleGroup    string
	SecondaryMuscleGroups string
	Equipment             string
	Difficulty            string
	MovementCategory      string
	BodyPart              string
	ImageURL              string
	AnimationURL          string
	SourceMediaID         string
	MediaAttribution      string
}

// AdminUpdateExerciseInput carries the mutable fields of a catalogue entry. A
// nil field is left untouched; a provided field is written even when empty, so
// media references and classifications can be cleared.
type AdminUpdateExerciseInput struct {
	Description           *string
	Instructions          *string
	InstructionSteps      *[]string
	TargetMuscles         *string
	PrimaryMuscleGroup    *string
	SecondaryMuscleGroups *string
	Equipment             *string
	Difficulty            *string
	MovementCategory      *string
	BodyPart              *string
	ImageURL              *string
	AnimationURL          *string
	SourceMediaID         *string
	MediaAttribution      *string
	Status                *string
}

// ExerciseAdminRepository is the data-access surface required by the
// administration catalogue service.
type ExerciseAdminRepository interface {
	AdminList(ctx context.Context, filter repositories.AdminExerciseFilter, page, limit int) ([]models.Exercise, int64, error)
	FindAnyByID(ctx context.Context, exerciseID string) (*models.Exercise, error)
	FindByName(ctx context.Context, name string) (*models.Exercise, error)
	Create(ctx context.Context, exercise *models.Exercise) error
	Update(ctx context.Context, exerciseID string, update repositories.ExerciseUpdate) error
}

// AdminService exposes the catalogue administration surface. Callers must
// already be authenticated administrators; authorization is enforced by the
// route middleware before this service is reached.
type AdminService interface {
	AdminListExercises(ctx context.Context, filter repositories.AdminExerciseFilter, page, limit int) (AdminExerciseListResult, error)
	AdminGetExercise(ctx context.Context, exerciseID string) (*AdminExercise, error)
	AdminCreateExercise(ctx context.Context, input AdminCreateExerciseInput) (*AdminExercise, error)
	AdminUpdateExercise(ctx context.Context, exerciseID string, input AdminUpdateExerciseInput) (*AdminExercise, error)
}

type adminService struct {
	exercises ExerciseAdminRepository
}

func NewAdminService(exercises ExerciseAdminRepository) AdminService {
	return &adminService{exercises: exercises}
}

// AdminListExercises returns one page of catalogue entries including draft and
// archived ones.
func (s *adminService) AdminListExercises(ctx context.Context, filter repositories.AdminExerciseFilter, page, limit int) (AdminExerciseListResult, error) {
	filter = normalizeAdminFilter(filter)
	if err := validateAdminFilter(filter); err != nil {
		return AdminExerciseListResult{}, err
	}
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return AdminExerciseListResult{}, err
	}

	exerciseModels, total, err := s.exercises.AdminList(ctx, filter, page, limit)
	if err != nil {
		return AdminExerciseListResult{}, fmt.Errorf("failed to list exercises: %w", err)
	}

	return AdminExerciseListResult{
		Exercises: toAdminSafeList(exerciseModels),
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

// AdminGetExercise returns one catalogue entry regardless of publication
// status.
func (s *adminService) AdminGetExercise(ctx context.Context, exerciseID string) (*AdminExercise, error) {
	if err := validateExerciseID(exerciseID); err != nil {
		return nil, err
	}
	exerciseModel, err := s.exercises.FindAnyByID(ctx, exerciseID)
	if err != nil {
		return nil, mapAdminLookupError(err)
	}
	return toAdminSafe(exerciseModel), nil
}

// AdminCreateExercise creates one catalogue entry as a draft. The name must be
// unique among active entries; the uniqueness guarantee is the database's, so
// a concurrent create cannot slip past the pre-check.
func (s *adminService) AdminCreateExercise(ctx context.Context, input AdminCreateExerciseInput) (*AdminExercise, error) {
	input.Name = strings.TrimSpace(input.Name)
	if err := validateExerciseName(input.Name); err != nil {
		return nil, err
	}

	if _, err := s.exercises.FindByName(ctx, input.Name); err == nil {
		return nil, ErrExerciseConflict
	} else if !errors.Is(err, repositories.ErrExerciseNotFound) {
		return nil, fmt.Errorf("failed to check exercise name: %w", err)
	}

	stepsJSON, err := marshalSteps(input.InstructionSteps)
	if err != nil {
		return nil, err
	}

	bodyPart := strings.TrimSpace(input.BodyPart)
	exercise := &models.Exercise{
		Name:                  input.Name,
		Description:           strings.TrimSpace(input.Description),
		Instructions:          strings.TrimSpace(input.Instructions),
		InstructionSteps:      rawSteps(stepsJSON),
		TargetMuscles:         strings.TrimSpace(input.TargetMuscles),
		PrimaryMuscleGroup:    strings.TrimSpace(input.PrimaryMuscleGroup),
		SecondaryMuscleGroups: strings.TrimSpace(input.SecondaryMuscleGroups),
		Equipment:             strings.TrimSpace(input.Equipment),
		Difficulty:            strings.TrimSpace(input.Difficulty),
		MovementCategory:      strings.TrimSpace(input.MovementCategory),
		BodyPart:              optionalString(bodyPart),
		ImageURL:              strings.TrimSpace(input.ImageURL),
		AnimationURL:          strings.TrimSpace(input.AnimationURL),
		SourceMediaID:         strings.TrimSpace(input.SourceMediaID),
		MediaAttribution:      strings.TrimSpace(input.MediaAttribution),
		Status:                models.ExerciseStatusDraft,
	}
	if err := validateStoredFields(exerciseFieldValues{
		targetMuscles:         exercise.TargetMuscles,
		primaryMuscleGroup:    exercise.PrimaryMuscleGroup,
		secondaryMuscleGroups: exercise.SecondaryMuscleGroups,
		equipment:             exercise.Equipment,
		difficulty:            exercise.Difficulty,
		movementCategory:      exercise.MovementCategory,
		bodyPart:              bodyPart,
		imageURL:              exercise.ImageURL,
		animationURL:          exercise.AnimationURL,
		sourceMediaID:         exercise.SourceMediaID,
		mediaAttribution:      exercise.MediaAttribution,
		description:           exercise.Description,
		instructions:          exercise.Instructions,
	}); err != nil {
		return nil, err
	}

	if err := s.exercises.Create(ctx, exercise); err != nil {
		if errors.Is(err, repositories.ErrExerciseNameTaken) {
			return nil, ErrExerciseConflict
		}
		return nil, fmt.Errorf("failed to create exercise: %w", err)
	}
	return toAdminSafe(exercise), nil
}

// AdminUpdateExercise applies the provided fields to one catalogue entry and
// returns the updated state.
func (s *adminService) AdminUpdateExercise(ctx context.Context, exerciseID string, input AdminUpdateExerciseInput) (*AdminExercise, error) {
	if err := validateExerciseID(exerciseID); err != nil {
		return nil, err
	}
	if _, err := s.exercises.FindAnyByID(ctx, exerciseID); err != nil {
		return nil, mapAdminLookupError(err)
	}

	update, err := buildAdminUpdate(input)
	if err != nil {
		return nil, err
	}

	if err := s.exercises.Update(ctx, exerciseID, update); err != nil {
		return nil, mapAdminLookupError(err)
	}

	updated, err := s.exercises.FindAnyByID(ctx, exerciseID)
	if err != nil {
		return nil, mapAdminLookupError(err)
	}
	return toAdminSafe(updated), nil
}

// exerciseFieldValues groups the stored field values validated together so
// create and update cannot drift apart.
type exerciseFieldValues struct {
	description           string
	instructions          string
	targetMuscles         string
	primaryMuscleGroup    string
	secondaryMuscleGroups string
	equipment             string
	difficulty            string
	movementCategory      string
	bodyPart              string
	imageURL              string
	animationURL          string
	sourceMediaID         string
	mediaAttribution      string
	status                string
}

// buildAdminUpdate validates the update input and converts it into the
// repository update. Instruction steps are validated and serialized here.
func buildAdminUpdate(input AdminUpdateExerciseInput) (repositories.ExerciseUpdate, error) {
	update := repositories.ExerciseUpdate{}
	values := exerciseFieldValues{}
	statusProvided := input.Status != nil

	if input.Description != nil {
		values.description = strings.TrimSpace(*input.Description)
		update.Description = &values.description
	}
	if input.Instructions != nil {
		values.instructions = strings.TrimSpace(*input.Instructions)
		update.Instructions = &values.instructions
	}
	if input.InstructionSteps != nil {
		stepsJSON, err := marshalSteps(*input.InstructionSteps)
		if err != nil {
			return repositories.ExerciseUpdate{}, err
		}
		update.InstructionSteps = &stepsJSON
	}
	if input.TargetMuscles != nil {
		values.targetMuscles = strings.TrimSpace(*input.TargetMuscles)
		update.TargetMuscles = &values.targetMuscles
	}
	if input.PrimaryMuscleGroup != nil {
		values.primaryMuscleGroup = strings.TrimSpace(*input.PrimaryMuscleGroup)
		update.PrimaryMuscleGroup = &values.primaryMuscleGroup
	}
	if input.SecondaryMuscleGroups != nil {
		values.secondaryMuscleGroups = strings.TrimSpace(*input.SecondaryMuscleGroups)
		update.SecondaryMuscleGroups = &values.secondaryMuscleGroups
	}
	if input.Equipment != nil {
		values.equipment = strings.TrimSpace(*input.Equipment)
		update.Equipment = &values.equipment
	}
	if input.Difficulty != nil {
		values.difficulty = strings.TrimSpace(*input.Difficulty)
		update.Difficulty = &values.difficulty
	}
	if input.MovementCategory != nil {
		values.movementCategory = strings.TrimSpace(*input.MovementCategory)
		update.MovementCategory = &values.movementCategory
	}
	if input.BodyPart != nil {
		values.bodyPart = strings.TrimSpace(*input.BodyPart)
		update.BodyPart = &values.bodyPart
	}
	if input.ImageURL != nil {
		values.imageURL = strings.TrimSpace(*input.ImageURL)
		update.ImageURL = &values.imageURL
	}
	if input.AnimationURL != nil {
		values.animationURL = strings.TrimSpace(*input.AnimationURL)
		update.AnimationURL = &values.animationURL
	}
	if input.SourceMediaID != nil {
		values.sourceMediaID = strings.TrimSpace(*input.SourceMediaID)
		update.SourceMediaID = &values.sourceMediaID
	}
	if input.MediaAttribution != nil {
		values.mediaAttribution = strings.TrimSpace(*input.MediaAttribution)
		update.MediaAttribution = &values.mediaAttribution
	}
	if input.Status != nil {
		values.status = strings.TrimSpace(*input.Status)
		update.Status = &values.status
	}

	if err := validateStoredFields(values); err != nil {
		return repositories.ExerciseUpdate{}, err
	}
	if statusProvided && !contains(statusValues(), values.status) {
		return repositories.ExerciseUpdate{}, fmt.Errorf("%w: invalid status value", ErrInvalidInput)
	}
	return update, nil
}

// validateStoredFields validates every free-text catalogue field for length
// limits, controlled vocabularies and safe media paths. Empty values are
// always allowed: they clear the field.
func validateStoredFields(values exerciseFieldValues) error {
	if err := validateTextField("description", values.description, maxExerciseDescriptionLength); err != nil {
		return err
	}
	if err := validateTextField("instructions", values.instructions, maxExerciseInstructionsLength); err != nil {
		return err
	}
	if err := validateTextField("target_muscles", values.targetMuscles, maxExerciseTargetLength); err != nil {
		return err
	}
	if err := validateTextField("primary_muscle_group", values.primaryMuscleGroup, maxExercisePrimaryMuscleLength); err != nil {
		return err
	}
	if err := validateTextField("secondary_muscle_groups", values.secondaryMuscleGroups, maxExerciseSecondaryMusclesLength); err != nil {
		return err
	}
	if err := validateTextField("equipment", values.equipment, maxExerciseEquipmentLength); err != nil {
		return err
	}
	if err := validateTextField("difficulty", values.difficulty, maxExerciseDifficultyLength); err != nil {
		return err
	}
	if err := validateTextField("movement_category", values.movementCategory, maxExerciseMovementLength); err != nil {
		return err
	}
	if err := validateTextField("body_part", values.bodyPart, maxExerciseBodyPartLength); err != nil {
		return err
	}
	if err := validateMediaPath("image_url", values.imageURL); err != nil {
		return err
	}
	if err := validateMediaPath("animation_url", values.animationURL); err != nil {
		return err
	}
	if err := validateTextField("source_media_id", values.sourceMediaID, maxExerciseSourceMediaIDLength); err != nil {
		return err
	}
	if err := validateTextField("media_attribution", values.mediaAttribution, maxExerciseAttributionLength); err != nil {
		return err
	}

	if values.difficulty != "" && !contains(DifficultyValues, values.difficulty) {
		return fmt.Errorf("%w: invalid difficulty value", ErrInvalidInput)
	}
	if values.movementCategory != "" && !contains(MovementCategoryValues, values.movementCategory) {
		return fmt.Errorf("%w: invalid movement category value", ErrInvalidInput)
	}
	if values.bodyPart != "" && !contains(BodyPartValues, values.bodyPart) {
		return fmt.Errorf("%w: invalid body part value", ErrInvalidInput)
	}
	return nil
}

// validateExerciseName enforces the catalog name rules.
func validateExerciseName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: exercise name is required", ErrInvalidInput)
	}
	if len([]rune(name)) > maxExerciseNameLength {
		return fmt.Errorf("%w: exercise name exceeds the maximum length", ErrInvalidInput)
	}
	return nil
}

// validateTextField bounds one free-text field by rune count.
func validateTextField(field, value string, maxLength int) error {
	if len([]rune(value)) > maxLength {
		return fmt.Errorf("%w: %s exceeds the maximum length", ErrInvalidInput, field)
	}
	return nil
}

// validateMediaPath bounds a stored media reference and rejects anything that
// could escape the configured media base path once prefixed to it.
func validateMediaPath(field, value string) error {
	if len([]rune(value)) > maxExerciseMediaPathLength {
		return fmt.Errorf("%w: %s exceeds the maximum length", ErrInvalidInput, field)
	}
	if value == "" {
		return nil
	}
	if strings.Contains(value, "://") || strings.Contains(value, "..") || strings.HasPrefix(value, "/") {
		return fmt.Errorf("%w: %s must be a relative media path", ErrInvalidInput, field)
	}
	return nil
}

// validateSteps bounds the instruction step list.
func validateSteps(steps []string) error {
	if len(steps) > maxExerciseSteps {
		return fmt.Errorf("%w: too many instruction steps", ErrInvalidInput)
	}
	for _, step := range steps {
		trimmed := strings.TrimSpace(step)
		if trimmed == "" {
			return fmt.Errorf("%w: instruction steps cannot be empty", ErrInvalidInput)
		}
		if len([]rune(trimmed)) > maxExerciseStepLength {
			return fmt.Errorf("%w: instruction step exceeds the maximum length", ErrInvalidInput)
		}
	}
	return nil
}

// marshalSteps serializes a validated step list for storage. A nil or empty
// list is stored as NULL so the column stays absent rather than "[]".
func marshalSteps(steps []string) (string, error) {
	if len(steps) == 0 {
		return "", nil
	}
	trimmed := make([]string, 0, len(steps))
	for _, step := range steps {
		trimmed = append(trimmed, strings.TrimSpace(step))
	}
	if err := validateSteps(trimmed); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(trimmed)
	if err != nil {
		return "", fmt.Errorf("failed to encode instruction steps: %w", err)
	}
	return string(encoded), nil
}

// rawSteps converts a serialized step list into the stored JSON value. An
// absent list is stored as NULL, never as an invalid empty JSON document.
func rawSteps(stepsJSON string) json.RawMessage {
	if stepsJSON == "" {
		return nil
	}
	return json.RawMessage(stepsJSON)
}

// optionalString stores an empty value as NULL so nullable classification
// columns never hold an empty string the validity checks would reject.
func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// statusValues enumerates the publication lifecycle values.
func statusValues() []string {
	return []string{models.ExerciseStatusDraft, models.ExerciseStatusPublished, models.ExerciseStatusArchived}
}

// normalizeAdminFilter trims the administrative filter values.
func normalizeAdminFilter(filter repositories.AdminExerciseFilter) repositories.AdminExerciseFilter {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.BodyPart = strings.TrimSpace(filter.BodyPart)
	filter.Source = strings.TrimSpace(filter.Source)
	return filter
}

// validateAdminFilter bounds and vocabulary-checks the administrative filter.
func validateAdminFilter(filter repositories.AdminExerciseFilter) error {
	if len([]rune(filter.Query)) > MaxSearchLength {
		return fmt.Errorf("%w: query exceeds the maximum length", ErrInvalidInput)
	}
	if len([]rune(filter.Source)) > 40 {
		return fmt.Errorf("%w: source exceeds the maximum length", ErrInvalidInput)
	}
	if filter.Status != "" && !contains(statusValues(), filter.Status) {
		return fmt.Errorf("%w: invalid status filter value", ErrInvalidInput)
	}
	if filter.BodyPart != "" && !contains(BodyPartValues, filter.BodyPart) {
		return fmt.Errorf("%w: invalid body part filter value", ErrInvalidInput)
	}
	return nil
}

// mapAdminLookupError maps repository errors onto service errors.
func mapAdminLookupError(err error) error {
	switch {
	case errors.Is(err, repositories.ErrExerciseNotFound):
		return ErrExerciseNotFound
	default:
		return fmt.Errorf("failed to load exercise: %w", err)
	}
}

// toAdminSafe maps one model into the administrative representation.
func toAdminSafe(model *models.Exercise) *AdminExercise {
	sourceID := ""
	if model.SourceID != nil {
		sourceID = *model.SourceID
	}
	return &AdminExercise{
		ID:                    model.ID,
		Name:                  model.Name,
		Source:                model.Source,
		SourceID:              sourceID,
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
		ImageURL:              model.ImageURL,
		AnimationURL:          model.AnimationURL,
		SourceMediaID:         model.SourceMediaID,
		MediaAttribution:      model.MediaAttribution,
		Status:                model.Status,
		CreatedAt:             model.CreatedAt,
		UpdatedAt:             model.UpdatedAt,
	}
}

func toAdminSafeList(models []models.Exercise) []AdminExercise {
	list := make([]AdminExercise, 0, len(models))
	for i := range models {
		list = append(list, *toAdminSafe(&models[i]))
	}
	return list
}
