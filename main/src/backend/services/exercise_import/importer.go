// Package exercise_import imports external exercise datasets into the global
// exercise catalogue. The import is idempotent: every record carries a source
// identifier, re-imports enrich existing entries instead of duplicating them,
// and entries that already exist with the same normalized name are linked to
// the source record instead of being inserted again. No field that already has
// a value is ever overwritten, so RYZE-authored content survives a re-import
// untouched.
package exercise_import

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/exercises"
)

// SourceGymVisualDataset is the source identifier under which the bundled
// Gym visual exercise dataset is imported.
const SourceGymVisualDataset = "gymvisual"

const (
	maxRecords          = 20000
	maxNameLength       = 255
	maxEquipmentLength  = 255
	maxMuscleLength     = 80
	maxTargetLength     = 255
	maxInstructionsLen  = 5000
	maxSteps            = 50
	maxStepLength       = 500
	maxMediaPathLength  = 500
	maxMediaIDLength    = 40
	maxAttributionLen   = 255
	mediaImagesPrefix   = "images/"
	mediaVideosPrefix   = "videos/"
	secondarySeparator  = ", "
	maxSecondaryMuscles = 20
	maxSecondaryLength  = 255
)

var (
	sourceIDPattern = regexp.MustCompile(`^[0-9]{4}$`)
	mediaIDPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	imagePattern    = regexp.MustCompile(`\.(jpg|jpeg|png)$`)
	gifPattern      = regexp.MustCompile(`\.gif$`)
)

// DatasetRecord is one record of the external dataset. Only the English
// instructions are imported; the remaining languages stay in the source file
// for a future localization pass.
type DatasetRecord struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	BodyPart     string `json:"body_part"`
	Equipment    string `json:"equipment"`
	Instructions struct {
		En string `json:"en"`
	} `json:"instructions"`
	InstructionSteps struct {
		En []string `json:"en"`
	} `json:"instruction_steps"`
	MuscleGroup      string   `json:"muscle_group"`
	SecondaryMuscles []string `json:"secondary_muscles"`
	Target           string   `json:"target"`
	MediaID          string   `json:"media_id"`
	Image            string   `json:"image"`
	GifURL           string   `json:"gif_url"`
	Attribution      string   `json:"attribution"`
}

// Report summarizes one import run.
type Report struct {
	// Total is the number of records read from the source file.
	Total int
	// Unique is the number of records left after merging duplicate names
	// within the dataset itself.
	Unique int
	// Created counts newly inserted catalogue entries.
	Created int
	// MatchedBySource counts records whose source identifier already had a
	// catalogue entry.
	MatchedBySource int
	// MatchedByName counts records linked to an existing catalogue entry by
	// normalized name.
	MatchedByName int
	// Updated counts entries that received at least one previously missing
	// field.
	Updated int
	// SkippedDuplicateSourceIDs lists dataset ids skipped because an earlier
	// record in the same file already used the source id. Source identity is
	// stricter than a display name, so it is reported separately and checked
	// first.
	SkippedDuplicateSourceIDs []string
	// SkippedDuplicateNames lists dataset ids skipped because another record
	// with the same normalized name was already accepted.
	SkippedDuplicateNames []string
	// Conflicts lists dataset ids that could not be created because another
	// active catalogue entry already uses the same name outside the import
	// linkage. They are left untouched.
	Conflicts []string
}

// ExerciseRepository is the data-access surface required by the importer. The
// caller runs the import inside a transaction so a failed run leaves no
// partial catalogue behind.
type ExerciseRepository interface {
	ListActive(ctx context.Context) ([]models.Exercise, error)
	Create(ctx context.Context, exercise *models.Exercise) error
	Update(ctx context.Context, exerciseID string, update repositories.ExerciseUpdate) error
}

// Importer imports dataset records into the catalogue.
type Importer struct {
	exercises ExerciseRepository
}

func NewImporter(exercises ExerciseRepository) *Importer {
	return &Importer{exercises: exercises}
}

// Import reads a dataset JSON array and applies it to the catalogue. It is
// idempotent: running it twice with the same source produces no new entries
// and no field changes on the second run.
func (i *Importer) Import(ctx context.Context, reader io.Reader) (Report, error) {
	records, err := decodeRecords(reader)
	if err != nil {
		return Report{}, err
	}

	report := Report{Total: len(records)}
	if len(records) == 0 {
		return report, nil
	}

	for index := range records {
		if err := validateRecord(&records[index]); err != nil {
			return Report{}, fmt.Errorf("record %q: %w", records[index].ID, err)
		}
	}

	unique, skippedSourceIDs, skippedNames, err := mergeDuplicateRecords(records)
	if err != nil {
		return Report{}, err
	}
	report.Unique = len(unique)
	report.SkippedDuplicateSourceIDs = skippedSourceIDs
	report.SkippedDuplicateNames = skippedNames

	existing, err := i.exercises.ListActive(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("failed to load existing catalogue: %w", err)
	}

	bySource := make(map[string]*models.Exercise, len(existing))
	byName := make(map[string]*models.Exercise, len(existing))
	for index := range existing {
		entry := &existing[index]
		if entry.SourceID != nil {
			bySource[sourceKey(entry.Source, *entry.SourceID)] = entry
		}
		byName[normalizeName(entry.Name)] = entry
	}

	for index := range unique {
		record := &unique[index]

		if entry, found := bySource[sourceKey(SourceGymVisualDataset, record.ID)]; found {
			report.MatchedBySource++
			updated, err := i.enrich(ctx, entry, record)
			if err != nil {
				return Report{}, err
			}
			if updated {
				report.Updated++
			}
			continue
		}

		if entry, found := byName[normalizeName(record.Name)]; found && canLinkToSource(entry) {
			report.MatchedByName++
			updated, err := i.enrich(ctx, entry, record)
			if err != nil {
				return Report{}, err
			}
			if updated {
				report.Updated++
			}
			continue
		}

		created, err := i.create(ctx, record)
		if err != nil {
			return Report{}, err
		}
		if !created {
			report.Conflicts = append(report.Conflicts, record.ID)
			continue
		}
		report.Created++
	}

	return report, nil
}

// enrich writes every record field that the existing entry does not have yet,
// including the source linkage. Fields that already carry a value are never
// overwritten.
func (i *Importer) enrich(ctx context.Context, entry *models.Exercise, record *DatasetRecord) (bool, error) {
	update := repositories.ExerciseUpdate{}
	changed := false

	setString := func(target **string, value string) {
		if value == "" {
			return
		}
		*target = &value
		changed = true
	}

	if entry.Source == "" {
		source := SourceGymVisualDataset
		update.Source = &source
	}
	if entry.SourceID == nil {
		sourceID := record.ID
		update.SourceID = &sourceID
	}
	if entry.BodyPart == nil {
		setString(&update.BodyPart, record.BodyPart)
	}
	if entry.Instructions == "" {
		setString(&update.Instructions, strings.TrimSpace(record.Instructions.En))
	}
	if len(entry.InstructionSteps) == 0 {
		stepsJSON, err := encodeSteps(record.InstructionSteps.En)
		if err != nil {
			return false, fmt.Errorf("record %q: %w", record.ID, err)
		}
		setString(&update.InstructionSteps, stepsJSON)
	}
	if entry.TargetMuscles == "" {
		setString(&update.TargetMuscles, strings.TrimSpace(record.Target))
	}
	if entry.PrimaryMuscleGroup == "" {
		setString(&update.PrimaryMuscleGroup, strings.TrimSpace(record.MuscleGroup))
	}
	if entry.SecondaryMuscleGroups == "" {
		setString(&update.SecondaryMuscleGroups, strings.Join(trimAll(record.SecondaryMuscles), secondarySeparator))
	}
	if entry.Equipment == "" {
		setString(&update.Equipment, strings.TrimSpace(record.Equipment))
	}
	if entry.ImageURL == "" {
		setString(&update.ImageURL, record.Image)
	}
	if entry.AnimationURL == "" {
		setString(&update.AnimationURL, record.GifURL)
	}
	if entry.SourceMediaID == "" {
		setString(&update.SourceMediaID, record.MediaID)
	}
	if entry.MediaAttribution == "" {
		setString(&update.MediaAttribution, strings.TrimSpace(record.Attribution))
	}

	if !changed {
		return false, nil
	}
	if err := i.exercises.Update(ctx, entry.ID, update); err != nil {
		return false, fmt.Errorf("failed to enrich exercise %q: %w", record.ID, err)
	}
	return true, nil
}

// create inserts one record as a published catalogue entry. It reports false
// when the database rejects the name because another unlinked entry already
// uses it.
func (i *Importer) create(ctx context.Context, record *DatasetRecord) (bool, error) {
	stepsJSON, err := encodeSteps(record.InstructionSteps.En)
	if err != nil {
		return false, fmt.Errorf("record %q: %w", record.ID, err)
	}
	sourceID := record.ID
	bodyPart := record.BodyPart
	exercise := &models.Exercise{
		Name:                  record.Name,
		Source:                SourceGymVisualDataset,
		SourceID:              &sourceID,
		BodyPart:              &bodyPart,
		Instructions:          strings.TrimSpace(record.Instructions.En),
		InstructionSteps:      json.RawMessage(stepsJSON),
		TargetMuscles:         strings.TrimSpace(record.Target),
		PrimaryMuscleGroup:    strings.TrimSpace(record.MuscleGroup),
		SecondaryMuscleGroups: strings.Join(trimAll(record.SecondaryMuscles), secondarySeparator),
		Equipment:             strings.TrimSpace(record.Equipment),
		ImageURL:              record.Image,
		AnimationURL:          record.GifURL,
		SourceMediaID:         record.MediaID,
		MediaAttribution:      strings.TrimSpace(record.Attribution),
		Status:                models.ExerciseStatusPublished,
	}
	if err := i.exercises.Create(ctx, exercise); err != nil {
		if errors.Is(err, repositories.ErrExerciseSourceTaken) {
			return false, fmt.Errorf("source id %q is already linked to another exercise", record.ID)
		}
		if errors.Is(err, repositories.ErrExerciseNameTaken) {
			return false, nil
		}
		return false, fmt.Errorf("failed to create exercise %q: %w", record.ID, err)
	}
	return true, nil
}

// decodeRecords streams the dataset JSON array so the whole source file never
// has to be held in memory as raw bytes.
func decodeRecords(reader io.Reader) ([]DatasetRecord, error) {
	decoder := json.NewDecoder(reader)

	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("dataset is not valid JSON: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '[' {
		return nil, errors.New("dataset must be a JSON array of exercise records")
	}

	records := make([]DatasetRecord, 0, 1024)
	for decoder.More() {
		if len(records) >= maxRecords {
			return nil, fmt.Errorf("dataset exceeds the maximum of %d records", maxRecords)
		}
		var record DatasetRecord
		if err := decoder.Decode(&record); err != nil {
			return nil, fmt.Errorf("failed to decode dataset record: %w", err)
		}
		records = append(records, record)
	}

	token, err = decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("dataset is not a well-formed JSON array: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != ']' {
		return nil, errors.New("dataset must be a well-formed JSON array of exercise records")
	}

	// A second JSON value after the array (or trailing garbage that is not
	// whitespace) means the file is not exactly one dataset. Trailing
	// whitespace is accepted because the decoder reaches io.EOF.
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("dataset must contain exactly one JSON array")
		}
		return nil, fmt.Errorf("dataset has trailing data: %w", err)
	}
	return records, nil
}

// mergeDuplicateRecords folds repeated source ids and normalized names into one
// deterministic record stream. Source ids are checked first because they are
// the stable dataset identity; a repeated id with a different name is an
// unresolvable dataset conflict and aborts the import rather than silently
// discarding content. Name duplicates keep the first accepted record.
func mergeDuplicateRecords(records []DatasetRecord) ([]DatasetRecord, []string, []string, error) {
	unique := make([]DatasetRecord, 0, len(records))
	skippedSourceIDs := []string{}
	skippedNames := []string{}
	seenSourceIDs := make(map[string]string, len(records))
	seenNames := make(map[string]struct{}, len(records))

	for index := range records {
		record := &records[index]
		key := normalizeName(record.Name)

		if previousName, found := seenSourceIDs[record.ID]; found {
			if previousName != key {
				return nil, nil, nil, fmt.Errorf("duplicate source id %q refers to conflicting exercises", record.ID)
			}
			skippedSourceIDs = append(skippedSourceIDs, record.ID)
			continue
		}
		seenSourceIDs[record.ID] = key

		if _, found := seenNames[key]; found {
			skippedNames = append(skippedNames, record.ID)
			continue
		}
		seenNames[key] = struct{}{}
		unique = append(unique, *record)
	}
	return unique, skippedSourceIDs, skippedNames, nil
}

// validateRecord enforces the dataset contract before anything is written.
func validateRecord(record *DatasetRecord) error {
	if !sourceIDPattern.MatchString(record.ID) {
		return fmt.Errorf("invalid source id %q: four digits required", record.ID)
	}
	name := strings.TrimSpace(record.Name)
	if name == "" || len([]rune(name)) > maxNameLength {
		return fmt.Errorf("invalid name")
	}
	if !contains(exercises.BodyPartValues, record.BodyPart) {
		return fmt.Errorf("invalid body part %q", record.BodyPart)
	}
	if equipment := strings.TrimSpace(record.Equipment); equipment == "" || len([]rune(equipment)) > maxEquipmentLength {
		return fmt.Errorf("invalid equipment")
	}
	if instructions := strings.TrimSpace(record.Instructions.En); instructions == "" || len([]rune(instructions)) > maxInstructionsLen {
		return fmt.Errorf("invalid English instructions")
	}
	if err := validateSteps(record.InstructionSteps.En); err != nil {
		return err
	}
	if muscle := strings.TrimSpace(record.MuscleGroup); muscle == "" || len([]rune(muscle)) > maxMuscleLength {
		return fmt.Errorf("invalid muscle group")
	}
	if target := strings.TrimSpace(record.Target); target == "" || len([]rune(target)) > maxTargetLength {
		return fmt.Errorf("invalid target")
	}
	if len(record.SecondaryMuscles) > maxSecondaryMuscles {
		return fmt.Errorf("too many secondary muscles")
	}
	for _, muscle := range record.SecondaryMuscles {
		if len([]rune(strings.TrimSpace(muscle))) > maxMuscleLength {
			return fmt.Errorf("secondary muscle exceeds the maximum length")
		}
	}
	if secondary := strings.Join(trimAll(record.SecondaryMuscles), secondarySeparator); len([]rune(secondary)) > maxSecondaryLength {
		return fmt.Errorf("secondary muscles exceed the maximum combined length")
	}
	if err := validateMediaPath(record.Image, mediaImagesPrefix, imagePattern); err != nil {
		return fmt.Errorf("invalid image reference: %w", err)
	}
	if err := validateMediaPath(record.GifURL, mediaVideosPrefix, gifPattern); err != nil {
		return fmt.Errorf("invalid animation reference: %w", err)
	}
	if mediaID := strings.TrimSpace(record.MediaID); mediaID == "" || len(mediaID) > maxMediaIDLength || !mediaIDPattern.MatchString(mediaID) {
		return fmt.Errorf("invalid media id")
	}
	if attribution := strings.TrimSpace(record.Attribution); attribution == "" || len([]rune(attribution)) > maxAttributionLen {
		return fmt.Errorf("invalid media attribution")
	}
	return nil
}

// validateSteps bounds one step list.
func validateSteps(steps []string) error {
	if len(steps) == 0 {
		return errors.New("English instruction steps are required")
	}
	if len(steps) > maxSteps {
		return fmt.Errorf("too many instruction steps")
	}
	for _, step := range steps {
		trimmed := strings.TrimSpace(step)
		if trimmed == "" {
			return errors.New("instruction steps cannot be empty")
		}
		if len([]rune(trimmed)) > maxStepLength {
			return errors.New("instruction step exceeds the maximum length")
		}
	}
	return nil
}

// validateMediaPath enforces the dataset's relative media path shape so a
// stored path can never escape the configured media base once prefixed.
func validateMediaPath(path, prefix string, suffix *regexp.Regexp) error {
	if len([]rune(path)) > maxMediaPathLength || !strings.HasPrefix(path, prefix) || !suffix.MatchString(path) {
		return errors.New("path must be a dataset-relative media reference")
	}
	if strings.Contains(path, "..") {
		return errors.New("path must not contain parent traversal")
	}
	return nil
}

// encodeSteps serializes the validated step list for storage.
func encodeSteps(steps []string) (string, error) {
	if err := validateSteps(steps); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(trimAll(steps))
	if err != nil {
		return "", fmt.Errorf("failed to encode instruction steps: %w", err)
	}
	return string(encoded), nil
}

// canLinkToSource reports whether an existing entry may be linked to the
// dataset. Entries already linked to another source are never hijacked.
func canLinkToSource(entry *models.Exercise) bool {
	return entry.Source == "" || entry.Source == SourceGymVisualDataset
}

// sourceKey builds the case-sensitive lookup key for a source pair.
func sourceKey(source, sourceID string) string {
	return source + ":" + sourceID
}

// normalizeName folds a catalogue name for matching: trimmed, lowercased,
// internal whitespace collapsed.
func normalizeName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

// trimAll trims every entry of a string slice.
func trimAll(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		trimmed = append(trimmed, strings.TrimSpace(value))
	}
	return trimmed
}

// contains reports whether a string slice contains the exact value.
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
