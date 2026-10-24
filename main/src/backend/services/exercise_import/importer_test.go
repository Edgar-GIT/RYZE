package exercise_import_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/exercise_import"
)

// datasetRecord mirrors the imported dataset file shape for fixture building.
type datasetRecord struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Category         string          `json:"category"`
	BodyPart         string          `json:"body_part"`
	Equipment        string          `json:"equipment"`
	Instructions     localizedString `json:"instructions"`
	InstructionSteps localizedSteps  `json:"instruction_steps"`
	MuscleGroup      string          `json:"muscle_group"`
	SecondaryMuscles []string        `json:"secondary_muscles"`
	Target           string          `json:"target"`
	MediaID          string          `json:"media_id"`
	Image            string          `json:"image"`
	GifURL           string          `json:"gif_url"`
	Attribution      string          `json:"attribution"`
}

type localizedString struct {
	En string `json:"en"`
}

type localizedSteps struct {
	En []string `json:"en"`
}

// fixtureRecord builds one valid dataset record. Tests override fields through
// the returned pointer.
func fixtureRecord(id, name string) datasetRecord {
	return datasetRecord{
		ID:        id,
		Name:      name,
		Category:  "upper legs",
		BodyPart:  "upper legs",
		Equipment: "bodyweight",
		Instructions: localizedString{
			En: "Stand tall and move under control.",
		},
		InstructionSteps: localizedSteps{
			En: []string{"Stand tall.", "Move under control."},
		},
		MuscleGroup:      "quads",
		SecondaryMuscles: []string{"glutes", "hamstrings"},
		Target:           "quads",
		MediaID:          "abc1234",
		Image:            "images/" + id + "-abc1234.jpg",
		GifURL:           "videos/" + id + "-abc1234.gif",
		Attribution:      "© Gym visual — https://gymvisual.com/",
	}
}

func encodeFixture(t *testing.T, records []datasetRecord) string {
	t.Helper()
	encoded, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	return string(encoded)
}

// newImportTestContext opens the configured database inside a transaction that
// is rolled back when the test finishes, so imported rows never survive the
// test run.
func newImportTestContext(t *testing.T) (context.Context, repositories.ExerciseRepository) {
	t.Helper()

	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	tx := db.Begin()
	t.Cleanup(func() { _ = tx.Rollback() })

	return context.Background(), repositories.NewExerciseRepository(tx)
}

func TestImporterCreatesAndIsIdempotent(t *testing.T) {
	ctx, repo := newImportTestContext(t)
	importer := exercise_import.NewImporter(repo)

	records := []datasetRecord{
		fixtureRecord("9901", "RYZE-Import-Test Alpha"),
		fixtureRecord("9902", "RYZE-Import-Test Alpha"),
		fixtureRecord("9903", "RYZE-Import-Test Beta"),
		fixtureRecord("9904", "RYZE-Import-Test Gamma"),
	}
	fixture := encodeFixture(t, records)

	report, err := importer.Import(ctx, strings.NewReader(fixture))
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if report.Total != 4 || report.Unique != 3 || report.Created != 3 {
		t.Fatalf("first import report: got %+v", report)
	}
	if len(report.SkippedDuplicateNames) != 1 || report.SkippedDuplicateNames[0] != "9902" {
		t.Fatalf("first import duplicate names: got %v", report.SkippedDuplicateNames)
	}
	if len(report.SkippedDuplicateSourceIDs) != 0 {
		t.Fatalf("first import duplicate source ids: got %v", report.SkippedDuplicateSourceIDs)
	}
	if report.Total != report.Unique+len(report.SkippedDuplicateSourceIDs)+len(report.SkippedDuplicateNames) {
		t.Fatalf("report totals must account for every record: %+v", report)
	}

	alpha, err := repo.FindBySource(ctx, exercise_import.SourceGymVisualDataset, "9901")
	if err != nil {
		t.Fatalf("find imported record: %v", err)
	}
	if alpha.Name != "RYZE-Import-Test Alpha" || alpha.Status != models.ExerciseStatusPublished {
		t.Fatalf("imported entry: got %+v", alpha)
	}
	if alpha.BodyPart == nil || *alpha.BodyPart != "upper legs" || alpha.ImageURL != "images/9901-abc1234.jpg" || alpha.AnimationURL != "videos/9901-abc1234.gif" {
		t.Fatalf("imported entry fields: got %+v", alpha)
	}
	if len(alpha.InstructionSteps) == 0 || alpha.MediaAttribution == "" {
		t.Fatalf("imported content fields must be populated: got %+v", alpha)
	}

	// Re-importing the same source changes nothing.
	second, err := importer.Import(ctx, strings.NewReader(fixture))
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.Created != 0 || second.MatchedBySource != 3 || second.Updated != 0 {
		t.Fatalf("second import must be a no-op: got %+v", second)
	}
}

func TestImporterLinksAndEnrichesExistingEntries(t *testing.T) {
	ctx, repo := newImportTestContext(t)

	created := &models.Exercise{
		Name:         "RYZE-Import-Test Enrichment",
		Description:  "Curated RYZE description that must survive imports.",
		Instructions: "Curated RYZE instructions.",
		Equipment:    "dumbbell",
		Difficulty:   "Beginner",
		Status:       models.ExerciseStatusPublished,
	}
	if err := repo.Create(ctx, created); err != nil {
		t.Fatalf("seed existing exercise: %v", err)
	}

	record := fixtureRecord("9905", "RYZE-Import-Test Enrichment")
	record.BodyPart = "upper arms"
	report, err := exercise_import.NewImporter(repo).Import(ctx, strings.NewReader(encodeFixture(t, []datasetRecord{record})))
	if err != nil {
		t.Fatalf("import enrichment: %v", err)
	}
	if report.Created != 0 || report.MatchedByName != 1 || report.Updated != 1 {
		t.Fatalf("enrichment report: got %+v", report)
	}

	enriched, err := repo.FindAnyByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("reload enriched exercise: %v", err)
	}
	if enriched.Description != "Curated RYZE description that must survive imports." {
		t.Fatalf("existing description must never be overwritten, got %q", enriched.Description)
	}
	if enriched.Instructions != "Curated RYZE instructions." {
		t.Fatalf("existing instructions must never be overwritten, got %q", enriched.Instructions)
	}
	if enriched.Equipment != "dumbbell" {
		t.Fatalf("existing equipment must never be overwritten, got %q", enriched.Equipment)
	}
	if enriched.Source != exercise_import.SourceGymVisualDataset || enriched.SourceID == nil || *enriched.SourceID != "9905" {
		t.Fatalf("existing entry must be linked to its source, got source=%q id=%v", enriched.Source, enriched.SourceID)
	}
	if enriched.BodyPart == nil || *enriched.BodyPart != "upper arms" || enriched.ImageURL == "" || enriched.AnimationURL == "" {
		t.Fatalf("missing fields must be filled from the record, got %+v", enriched)
	}
	if enriched.Status != models.ExerciseStatusPublished {
		t.Fatalf("existing publication status must survive enrichment, got %q", enriched.Status)
	}

	// A curated field set after linking is also never overwritten.
	description := "Edited after import."
	if err := repo.Update(ctx, created.ID, repositories.ExerciseUpdate{Description: &description}); err != nil {
		t.Fatalf("update curated field: %v", err)
	}
	again, err := exercise_import.NewImporter(repo).Import(ctx, strings.NewReader(encodeFixture(t, []datasetRecord{record})))
	if err != nil {
		t.Fatalf("re-import after edit: %v", err)
	}
	if again.Updated != 0 {
		t.Fatalf("re-import of a linked entry must not rewrite fields: got %+v", again)
	}
	reloaded, err := repo.FindAnyByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("reload edited exercise: %v", err)
	}
	if reloaded.Description != description {
		t.Fatalf("edited description must survive re-import, got %q", reloaded.Description)
	}
}

func TestImporterSeparatesDuplicateSourceIDsFromDuplicateNames(t *testing.T) {
	ctx, repo := newImportTestContext(t)

	records := []datasetRecord{
		fixtureRecord("9901", "RYZE-Import-Test Source Identity"),
		fixtureRecord("9901", "RYZE-Import-Test Source Identity"),
		fixtureRecord("9902", "RYZE-Import-Test Duplicate Name"),
		fixtureRecord("9903", "RYZE-Import-Test Duplicate Name"),
	}
	report, err := exercise_import.NewImporter(repo).Import(ctx, strings.NewReader(encodeFixture(t, records)))
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	if report.Total != 4 || report.Unique != 2 || report.Created != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(report.SkippedDuplicateSourceIDs) != 1 || report.SkippedDuplicateSourceIDs[0] != "9901" {
		t.Fatalf("duplicate source ids: got %v", report.SkippedDuplicateSourceIDs)
	}
	if len(report.SkippedDuplicateNames) != 1 || report.SkippedDuplicateNames[0] != "9903" {
		t.Fatalf("duplicate names: got %v", report.SkippedDuplicateNames)
	}
	if len(report.Conflicts) != 0 {
		t.Fatalf("a duplicate inside the dataset must not be reported as a name conflict: %v", report.Conflicts)
	}
	if report.Total != report.Unique+len(report.SkippedDuplicateSourceIDs)+len(report.SkippedDuplicateNames) {
		t.Fatalf("report totals must account for every record: %+v", report)
	}
}

func TestImporterRejectsConflictingDuplicateSourceIDs(t *testing.T) {
	ctx, repo := newImportTestContext(t)

	_, baseline, err := repo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}

	records := []datasetRecord{
		fixtureRecord("9901", "RYZE-Import-Test Source One"),
		fixtureRecord("9901", "RYZE-Import-Test Source Two"),
	}
	if _, err := exercise_import.NewImporter(repo).Import(ctx, strings.NewReader(encodeFixture(t, records))); err == nil {
		t.Fatal("a duplicate source id with a different name must be rejected")
	}

	_, after, err := repo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("after rejected import: %v", err)
	}
	if after != baseline {
		t.Fatalf("a rejected duplicate source id must not write rows: baseline %d, after %d", baseline, after)
	}
}

func TestImporterRequiresExactlyOneJSONArray(t *testing.T) {
	ctx, repo := newImportTestContext(t)
	importer := exercise_import.NewImporter(repo)

	trailing := []string{
		encodeFixture(t, []datasetRecord{fixtureRecord("9901", "RYZE-Import-Test Trailing")}) + `{"extra":true}`,
		encodeFixture(t, []datasetRecord{fixtureRecord("9901", "RYZE-Import-Test Trailing")}) + `[]`,
		encodeFixture(t, []datasetRecord{fixtureRecord("9901", "RYZE-Import-Test Trailing")}) + `garbage`,
	}
	for _, fixture := range trailing {
		if _, err := importer.Import(ctx, strings.NewReader(fixture)); err == nil {
			t.Fatalf("expected trailing data to be rejected: %.80q", fixture)
		}
	}

	empty := " \n\t[] \r\n"
	report, err := importer.Import(ctx, strings.NewReader(empty))
	if err != nil {
		t.Fatalf("empty array with surrounding whitespace must be accepted: %v", err)
	}
	if report.Total != 0 || report.Unique != 0 {
		t.Fatalf("empty array report: %+v", report)
	}
}

func TestImporterRejectsInvalidRecordsWithoutWriting(t *testing.T) {
	ctx, repo := newImportTestContext(t)
	importer := exercise_import.NewImporter(repo)

	_, baseline, err := repo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}

	invalid := []struct {
		name   string
		mutate func(record *datasetRecord)
	}{
		{"source id not numeric", func(record *datasetRecord) { record.ID = "99A1" }},
		{"empty name", func(record *datasetRecord) { record.Name = "   " }},
		{"unknown body part", func(record *datasetRecord) { record.BodyPart = "head" }},
		{"missing instructions", func(record *datasetRecord) { record.Instructions.En = "" }},
		{"missing steps", func(record *datasetRecord) { record.InstructionSteps.En = nil }},
		{"traversal in image path", func(record *datasetRecord) { record.Image = "images/../secret.jpg" }},
		{"image outside images directory", func(record *datasetRecord) { record.Image = "videos/9901-abc1234.jpg" }},
		{"animation not a gif", func(record *datasetRecord) { record.GifURL = "videos/9901-abc1234.jpg" }},
		{"missing attribution", func(record *datasetRecord) { record.Attribution = "" }},
	}

	for _, testCase := range invalid {
		record := fixtureRecord("9901", "RYZE-Import-Test Invalid")
		testCase.mutate(&record)
		if _, err := importer.Import(ctx, strings.NewReader(encodeFixture(t, []datasetRecord{record}))); err == nil {
			t.Fatalf("invalid record %q: expected an error", testCase.name)
		}
	}

	_, after, err := repo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("after invalid imports: %v", err)
	}
	if after != baseline {
		t.Fatalf("invalid imports must not write rows: baseline %d, after %d", baseline, after)
	}

	// A malformed JSON body is rejected before any record is inspected.
	if _, err := importer.Import(ctx, strings.NewReader(`{"not":"an array"}`)); err == nil {
		t.Fatal("malformed dataset: expected an error")
	}
}
