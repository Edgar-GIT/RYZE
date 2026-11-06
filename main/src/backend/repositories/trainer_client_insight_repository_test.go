package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

func TestTrainerClientInsightRepository(t *testing.T) {
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
	userRepo := repositories.NewUserRepository(tx)
	trainerRepo := repositories.NewTrainerRepository(tx)
	programRepo := repositories.NewProgramRepository(tx)
	weekRepo := repositories.NewProgramWeekRepository(tx)
	workoutRepo := repositories.NewProgramWorkoutRepository(tx)
	clientRepo := repositories.NewTrainerClientRepository(tx)
	assignmentRepo := repositories.NewProgramAssignmentRepository(tx)
	repo := repositories.NewTrainerClientInsightRepository(tx)

	seedUser := func() *models.User {
		user := &models.User{
			Email:        fmt.Sprintf("insight-repo-%d@ryze.local", time.Now().UnixNano()),
			PasswordHash: "prepared-hash",
			FirstName:    "Jane",
			LastName:     "Roe",
		}
		if err := userRepo.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
		return user
	}
	seedTrainer := func() *models.Trainer {
		item := &models.Trainer{UserID: seedUser().ID}
		if err := trainerRepo.Create(ctx, item); err != nil {
			t.Fatalf("create trainer: %v", err)
		}
		return item
	}
	seedProgram := func(trainerID, name string) *models.Program {
		program := &models.Program{
			TrainerID: trainerID,
			Name:      name,
			Type:      models.ProgramTypePremium,
			Status:    models.ProgramStatusPublished,
		}
		if err := programRepo.Create(ctx, program); err != nil {
			t.Fatalf("create program: %v", err)
		}
		return program
	}
	seedWeek := func(trainerID, programID string) *models.ProgramWeek {
		week := &models.ProgramWeek{}
		if err := weekRepo.Create(ctx, trainerID, programID, week); err != nil {
			t.Fatalf("create week: %v", err)
		}
		return week
	}
	seedWorkout := func(trainerID, programID, weekID string) *models.ProgramWorkout {
		workout := &models.ProgramWorkout{}
		if err := workoutRepo.Create(ctx, trainerID, programID, weekID, workout); err != nil {
			t.Fatalf("create workout: %v", err)
		}
		return workout
	}
	seedExercise := func(name string) *models.Exercise {
		exercise := &models.Exercise{Name: name, Status: models.ExerciseStatusPublished}
		if err := tx.Create(exercise).Error; err != nil {
			t.Fatalf("create exercise: %v", err)
		}
		return exercise
	}
	seedSession := func(userID, programID, workoutID, programName, status string, startedAt time.Time, completedAt, abandonedAt, lastResumedAt *time.Time, accumulated int) *models.WorkoutSession {
		session := &models.WorkoutSession{
			UserID:             userID,
			ProgramID:          programID,
			ProgramWorkoutID:   workoutID,
			ProgramName:        programName,
			WeekNumber:         1,
			WorkoutPosition:    1,
			Status:             status,
			StartedAt:          startedAt,
			CompletedAt:        completedAt,
			AbandonedAt:        abandonedAt,
			LastResumedAt:      lastResumedAt,
			AccumulatedSeconds: accumulated,
		}
		if err := tx.Create(session).Error; err != nil {
			t.Fatalf("create session: %v", err)
		}
		return session
	}
	seedSets := func(sessionID, exerciseID, name string) []models.WorkoutSessionSet {
		exercise := &models.WorkoutSessionExercise{
			WorkoutSessionID: sessionID,
			ExerciseID:       exerciseID,
			Position:         1,
			Name:             name,
		}
		if err := tx.Create(exercise).Error; err != nil {
			t.Fatalf("create session exercise: %v", err)
		}
		reps := 10
		actualReps := 8
		weight := 80.5
		actualWeight := 75.0
		sets := []models.WorkoutSessionSet{
			{
				WorkoutSessionExerciseID: exercise.ID,
				SetNumber:                1,
				SetType:                  "normal",
				PrescribedReps:           &reps,
				PrescribedWeightKg:       &weight,
				ActualReps:               &actualReps,
				ActualWeightKg:           &actualWeight,
				Status:                   models.WorkoutSetStatusCompleted,
			},
			{
				WorkoutSessionExerciseID: exercise.ID,
				SetNumber:                2,
				SetType:                  "normal",
				PrescribedReps:           &reps,
				Status:                   models.WorkoutSetStatusSkipped,
			},
			{
				WorkoutSessionExerciseID: exercise.ID,
				SetNumber:                3,
				SetType:                  "normal",
				PrescribedReps:           &reps,
				Status:                   models.WorkoutSetStatusPending,
			},
		}
		if err := tx.Create(&sets).Error; err != nil {
			t.Fatalf("create session sets: %v", err)
		}
		return sets
	}

	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	trainer := seedTrainer()
	otherTrainer := seedTrainer()
	client := seedUser()
	otherClient := seedUser()
	if err := clientRepo.Create(ctx, &models.TrainerClient{TrainerID: trainer.ID, UserID: client.ID}); err != nil {
		t.Fatalf("create relationship: %v", err)
	}

	program := seedProgram(trainer.ID, "Strength Builder")
	otherProgram := seedProgram(otherTrainer.ID, "Other Plan")
	purchasedProgram := seedProgram(trainer.ID, "Purchased Plan")
	expiredProgram := seedProgram(trainer.ID, "Expired Plan")
	freeProgram := seedProgram(trainer.ID, "Free Plan")
	week := seedWeek(trainer.ID, program.ID)
	workout := seedWorkout(trainer.ID, program.ID, week.ID)
	exercise := seedExercise(fmt.Sprintf("Insight Squat %d", time.Now().UnixNano()))

	if err := assignmentRepo.Create(ctx, trainer.ID, client.ID, program.ID, &models.ProgramAssignment{}); err != nil {
		t.Fatalf("create current assignment: %v", err)
	}
	if err := tx.Create(&models.ProgramAssignment{
		TrainerID: otherTrainer.ID,
		UserID:    client.ID,
		ProgramID: otherProgram.ID,
	}).Error; err != nil {
		t.Fatalf("create other assignment: %v", err)
	}
	if err := tx.Create(&models.Entitlement{UserID: client.ID, ProgramID: purchasedProgram.ID}).Error; err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	past := now.Add(-time.Hour)
	if err := tx.Create(&models.Entitlement{UserID: client.ID, ProgramID: expiredProgram.ID, ExpiresAt: &past}).Error; err != nil {
		t.Fatalf("create expired entitlement: %v", err)
	}
	if err := tx.Create(&models.FreeProgramActivation{UserID: client.ID, ProgramID: freeProgram.ID, ActivatedAt: now.Add(-2 * time.Hour)}).Error; err != nil {
		t.Fatalf("create free activation: %v", err)
	}

	completedAt := now.Add(-time.Hour)
	abandonedAt := now.Add(-2 * time.Hour)
	lastResumedAt := now.Add(-30 * time.Second)
	completedSession := seedSession(client.ID, program.ID, workout.ID, program.Name, models.WorkoutSessionStatusCompleted, now.Add(-3*time.Hour), &completedAt, nil, nil, 1800)
	abandonedSession := seedSession(client.ID, program.ID, workout.ID, program.Name, models.WorkoutSessionStatusAbandoned, now.Add(-4*time.Hour), nil, &abandonedAt, nil, 600)
	activeSession := seedSession(client.ID, program.ID, workout.ID, program.Name, models.WorkoutSessionStatusInProgress, now.Add(-10*time.Minute), nil, nil, &lastResumedAt, 10)
	seedSession(otherClient.ID, program.ID, workout.ID, program.Name, models.WorkoutSessionStatusCompleted, now.Add(-30*time.Minute), &now, nil, nil, 1200)

	completedSets := seedSets(completedSession.ID, exercise.ID, exercise.Name)
	seedSets(abandonedSession.ID, exercise.ID, exercise.Name)
	seedSets(activeSession.ID, exercise.ID, exercise.Name)

	historyTime := now.Add(-5 * time.Hour)
	if err := tx.Create(&models.WorkoutHistory{UserID: client.ID, ProgramWorkoutID: workout.ID, CompletedAt: historyTime}).Error; err != nil {
		t.Fatalf("create legacy history: %v", err)
	}
	if err := tx.Create(&models.WorkoutHistory{UserID: client.ID, ProgramWorkoutID: workout.ID, CompletedAt: now.Add(-4 * time.Hour)}).Error; err != nil {
		t.Fatalf("create second legacy history: %v", err)
	}

	assignments, err := repo.ListActiveAssignmentsByUser(ctx, client.ID)
	if err != nil {
		t.Fatalf("list assignments: %v", err)
	}
	if len(assignments) != 2 || assignments[0].Program.ID == "" || assignments[1].Program.ID == "" {
		t.Fatalf("assignments = %+v", assignments)
	}
	entitlements, err := repo.ListActiveEntitlementsByUser(ctx, client.ID)
	if err != nil {
		t.Fatalf("list entitlements: %v", err)
	}
	if len(entitlements) != 1 || entitlements[0].ProgramID != purchasedProgram.ID || entitlements[0].Program.ID != purchasedProgram.ID {
		t.Fatalf("entitlements = %+v", entitlements)
	}
	activations, err := repo.ListActiveFreeActivationsByUser(ctx, client.ID)
	if err != nil {
		t.Fatalf("list activations: %v", err)
	}
	if len(activations) != 1 || activations[0].ProgramID != freeProgram.ID || activations[0].Program.ID != freeProgram.ID {
		t.Fatalf("activations = %+v", activations)
	}

	total, err := repo.CountSessions(ctx, client.ID)
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if total != 2 {
		t.Fatalf("session count = %d, want 2", total)
	}
	sessions, err := repo.ListSessions(ctx, client.ID, 10, 0)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 2 || sessions[0].ID != completedSession.ID || sessions[1].ID != abandonedSession.ID {
		t.Fatalf("sessions = %+v", sessions)
	}
	if sessions[0].TotalSets != 3 || sessions[0].CompletedSets != 1 || sessions[0].SkippedSets != 1 {
		t.Fatalf("completed session counters = %+v", sessions[0])
	}
	if sessions[0].CompletedAt == nil {
		t.Fatal("completed session lost its completion timestamp")
	}

	active, err := repo.FindActiveSession(ctx, client.ID)
	if err != nil {
		t.Fatalf("find active session: %v", err)
	}
	if active == nil || active.ID != activeSession.ID || active.TotalSets != 3 || active.LastResumedAt == nil {
		t.Fatalf("active session = %+v", active)
	}

	legacy, err := repo.LegacySummary(ctx, client.ID)
	if err != nil {
		t.Fatalf("legacy summary: %v", err)
	}
	if legacy.Count != 2 || legacy.LastCompletedAt == nil || !legacy.LastCompletedAt.Equal(now.Add(-4*time.Hour)) {
		t.Fatalf("legacy summary = %+v", legacy)
	}

	summaries, err := repo.ListExerciseSummaries(ctx, client.ID)
	if err != nil {
		t.Fatalf("exercise summaries: %v", err)
	}
	if len(summaries) != 1 || summaries[0].ExerciseID != exercise.ID || summaries[0].SessionCount != 1 || summaries[0].Name != exercise.Name {
		t.Fatalf("exercise summaries = %+v", summaries)
	}

	history, err := repo.ListExerciseHistory(ctx, client.ID, exercise.ID, 10)
	if err != nil {
		t.Fatalf("exercise history: %v", err)
	}
	if len(history) != 1 || history[0].ID != completedSession.ID || len(history[0].Exercises) != 1 {
		t.Fatalf("exercise history = %+v", history)
	}
	sets := history[0].Exercises[0].Sets
	if len(sets) != len(completedSets) {
		t.Fatalf("history sets = %d, want %d", len(sets), len(completedSets))
	}
	if sets[0].SetNumber != 1 || sets[0].ActualReps == nil || *sets[0].ActualReps != 8 || sets[0].PrescribedReps == nil || *sets[0].PrescribedReps != 10 {
		t.Fatalf("prescribed and actual values not preserved: %+v", sets[0])
	}

	if _, err := repo.FindActiveSession(ctx, otherClient.ID); err != nil {
		t.Fatalf("other client active session: %v", err)
	}
	otherSessions, err := repo.ListSessions(ctx, otherClient.ID, 10, 0)
	if err != nil {
		t.Fatalf("other client sessions: %v", err)
	}
	if len(otherSessions) != 1 {
		t.Fatalf("other client sessions = %d, want 1", len(otherSessions))
	}
}
