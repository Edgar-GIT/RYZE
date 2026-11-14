package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

// TestAchievementsPreferencesAndAudience covers the queries added with the
// EXP/medals, the plan preferences and the plan audience (v2.116.0).
func TestAchievementsPreferencesAndAudience(t *testing.T) {
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
	defer tx.Rollback()
	ctx := context.Background()

	users := repositories.NewUserRepository(tx)
	seedUser := func() *models.User {
		user := &models.User{Email: fmt.Sprintf("achieve-%d@ryze.local", time.Now().UnixNano()), PasswordHash: "prepared-hash", FirstName: "Ana", LastName: "Lima"}
		if err := users.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}
	member := seedUser()
	other := seedUser()
	trainer := &models.Trainer{UserID: seedUser().ID}
	if err := repositories.NewTrainerRepository(tx).Create(ctx, trainer); err != nil {
		t.Fatalf("create trainer: %v", err)
	}
	program := &models.Program{TrainerID: trainer.ID, Name: "Strength", Type: models.ProgramTypeFree, Status: models.ProgramStatusPublished}
	if err := repositories.NewProgramRepository(tx).Create(ctx, program); err != nil {
		t.Fatalf("create program: %v", err)
	}
	week := &models.ProgramWeek{}
	if err := repositories.NewProgramWeekRepository(tx).Create(ctx, trainer.ID, program.ID, week); err != nil {
		t.Fatalf("create week: %v", err)
	}
	workout := &models.ProgramWorkout{}
	if err := repositories.NewProgramWorkoutRepository(tx).Create(ctx, trainer.ID, program.ID, week.ID, workout); err != nil {
		t.Fatalf("create workout: %v", err)
	}
	exercise := &models.Exercise{Name: "Bench press", Status: models.ExerciseStatusPublished}
	if err := tx.Create(exercise).Error; err != nil {
		t.Fatalf("create exercise: %v", err)
	}

	at := time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)
	seedSession := func(userID, status string, statuses ...string) *models.WorkoutSession {
		session := &models.WorkoutSession{UserID: userID, ProgramID: program.ID, ProgramWorkoutID: workout.ID, ProgramName: program.Name, WeekNumber: 1, WorkoutPosition: 1, Status: status, StartedAt: at}
		if status == models.WorkoutSessionStatusCompleted {
			done := at.Add(time.Hour)
			session.CompletedAt = &done
		}
		if err := tx.Create(session).Error; err != nil {
			t.Fatalf("create session: %v", err)
		}
		row := &models.WorkoutSessionExercise{WorkoutSessionID: session.ID, ExerciseID: exercise.ID, Position: 1, Name: exercise.Name}
		if err := tx.Create(row).Error; err != nil {
			t.Fatalf("create session exercise: %v", err)
		}
		for i, setStatus := range statuses {
			weight, reps := 60.0+float64(i)*5, 8
			set := &models.WorkoutSessionSet{WorkoutSessionExerciseID: row.ID, SetNumber: i + 1, SetType: "working", ActualWeightKg: &weight, ActualReps: &reps, Status: setStatus}
			if err := tx.Create(set).Error; err != nil {
				t.Fatalf("create set: %v", err)
			}
		}
		return session
	}
	completed := seedSession(member.ID, models.WorkoutSessionStatusCompleted, models.WorkoutSetStatusCompleted, models.WorkoutSetStatusCompleted, models.WorkoutSetStatusSkipped)
	seedSession(member.ID, models.WorkoutSessionStatusAbandoned, models.WorkoutSetStatusCompleted)
	foreign := seedSession(other.ID, models.WorkoutSessionStatusCompleted, models.WorkoutSetStatusCompleted)

	achievements := repositories.NewAchievementsRepository(tx)
	rows, err := achievements.ListCompletedSets(ctx, member.ID)
	if err != nil {
		t.Fatalf("ListCompletedSets: %v", err)
	}
	// Only the completed sets of the member's completed session.
	if len(rows) != 2 || rows[0].SessionID != completed.ID || *rows[1].WeightKg != 65 || rows[0].ExerciseName != "Bench press" {
		t.Fatalf("rows = %+v", rows)
	}
	if status, err := achievements.SessionStatus(ctx, member.ID, completed.ID); err != nil || status != models.WorkoutSessionStatusCompleted {
		t.Fatalf("SessionStatus = %q, %v", status, err)
	}
	if _, err := achievements.SessionStatus(ctx, member.ID, foreign.ID); !errors.Is(err, repositories.ErrAchievementSessionNotFound) {
		t.Fatalf("another member's session must not be found: %v", err)
	}

	// Plan preferences.
	preferences := repositories.NewPlanPreferencesRepository(tx)
	if got, err := preferences.Get(ctx, member.ID); err != nil || got.Sex != nil || got.FilterBySex != nil {
		t.Fatalf("fresh preferences = %+v, %v", got, err)
	}
	female, yes := models.SexFemale, true
	if err := preferences.Save(ctx, member.ID, repositories.PlanPreferences{Sex: &female, FilterBySex: &yes}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got, _ := preferences.Get(ctx, member.ID); got.Sex == nil || *got.Sex != female || got.FilterBySex == nil || !*got.FilterBySex {
		t.Fatalf("saved preferences = %+v", got)
	}

	// Audience: the database refuses an unknown value and a Level 2 audience;
	// the catalogue keeps plans for everyone plus the requested audience.
	male := models.SexMale
	platform := func(name string, audience *string) *models.Program {
		item := &models.Program{Name: name, Type: models.ProgramTypeFree, Status: models.ProgramStatusPublished, ProductType: models.ProgramProductTypeGeneric, Currency: "EUR", Audience: audience}
		if err := tx.Omit("TrainerID", "Weeks", "CuratedNutrition").Create(item).Error; err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		return item
	}
	tag := fmt.Sprintf("Audience %d", time.Now().UnixNano())
	everyone := platform(tag+" all", nil)
	forWomen := platform(tag+" women", &female)
	platform(tag+" men", &male)
	if err := tx.Exec("SAVEPOINT bad_audience").Error; err != nil {
		t.Fatal(err)
	}
	invalid := "kids"
	if err := tx.Omit("TrainerID", "Weeks", "CuratedNutrition").Create(&models.Program{Name: "Bad", Type: models.ProgramTypeFree, Status: models.ProgramStatusDraft, ProductType: models.ProgramProductTypeGeneric, Currency: "EUR", Audience: &invalid}).Error; err == nil {
		t.Fatal("an unknown audience must be refused")
	}
	tx.Exec("ROLLBACK TO SAVEPOINT bad_audience")

	found, total, err := repositories.NewProgramRepository(tx).SearchPublished(ctx, repositories.PublicCatalogFilter{Query: tag, ScopeGeneric: true, Audience: female}, 1, 10)
	if err != nil {
		t.Fatalf("SearchPublished: %v", err)
	}
	ids := map[string]bool{}
	for _, item := range found {
		ids[item.ID] = true
	}
	if total != 2 || !ids[everyone.ID] || !ids[forWomen.ID] {
		t.Fatalf("audience=female returned %d programs %v, want the plan for everyone and the plan for women", total, ids)
	}
}
