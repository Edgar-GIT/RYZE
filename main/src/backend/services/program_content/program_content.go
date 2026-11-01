// Package program_content maps a loaded program aggregate into the safe,
// client-readable structure shared by every authenticated access surface. It
// never exposes the owning trainer, parent identifiers (except the stable ids
// a client needs to operate on a workout), deletion markers or internal data.
package program_content

import (
	"time"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/services/exercise_media"
)

// Exercise is the safe catalog summary exposed to the client.
type Exercise struct {
	ID               string
	Name             string
	Description      string
	TargetMuscles    string
	Equipment        string
	Difficulty       string
	VideoURL         string
	ImageURL         string
	MediaAttribution string
	MediaAvailable   bool
}

// Set is the safe representation of one prescribed set of a workout exercise.
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

// WorkoutExercise is one exercise usage inside a workout.
type WorkoutExercise struct {
	ID           string
	Position     int
	Instructions string
	Notes        string
	Exercise     Exercise
	Sets         []Set
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Workout is one workout slot inside a program week.
type Workout struct {
	ID        string
	Position  int
	Exercises []WorkoutExercise
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Week is one week slot inside a program.
type Week struct {
	ID         string
	WeekNumber int
	Workouts   []Workout
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Program is the safe representation of a program with its full active
// structure.
type Program struct {
	ID              string
	Name            string
	Description     string
	Type            string
	Status          string
	PriceMinorUnits int64
	Currency        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Weeks           []Week
}

// Builder maps loaded program aggregates using the configured media resolver.
type Builder struct {
	media exercise_media.Resolver
}

// NewBuilder creates a Builder for the given media configuration.
func NewBuilder(media config.ExerciseMediaConfig) *Builder {
	return &Builder{media: exercise_media.New(media)}
}

// Build converts a loaded program aggregate into its safe client form.
func (b *Builder) Build(model *models.Program) *Program {
	if model == nil {
		return nil
	}

	program := &Program{
		ID:              model.ID,
		Name:            model.Name,
		Description:     model.Description,
		Type:            model.Type,
		Status:          model.Status,
		PriceMinorUnits: model.PriceMinorUnits,
		Currency:        model.Currency,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
		Weeks:           make([]Week, 0, len(model.Weeks)),
	}
	for i := range model.Weeks {
		program.Weeks = append(program.Weeks, b.newWeek(&model.Weeks[i]))
	}
	return program
}

func (b *Builder) newWeek(model *models.ProgramWeek) Week {
	week := Week{
		ID:         model.ID,
		WeekNumber: model.WeekNumber,
		Workouts:   make([]Workout, 0, len(model.Workouts)),
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
	for i := range model.Workouts {
		week.Workouts = append(week.Workouts, b.newWorkout(&model.Workouts[i]))
	}
	return week
}

func (b *Builder) newWorkout(model *models.ProgramWorkout) Workout {
	workout := Workout{
		ID:        model.ID,
		Position:  model.Position,
		Exercises: make([]WorkoutExercise, 0, len(model.Exercises)),
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
	for i := range model.Exercises {
		workout.Exercises = append(workout.Exercises, b.newWorkoutExercise(&model.Exercises[i]))
	}
	return workout
}

func (b *Builder) newWorkoutExercise(model *models.WorkoutExercise) WorkoutExercise {
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
	return WorkoutExercise{
		ID:           model.ID,
		Position:     model.Position,
		Instructions: model.Instructions,
		Notes:        model.Notes,
		Exercise:     b.newExercise(model.Exercise),
		Sets:         sets,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func (b *Builder) newExercise(model *models.Exercise) Exercise {
	if model == nil {
		return Exercise{}
	}
	media := b.media.Resolve(model.VideoURL, model.ImageURL, model.AnimationURL, model.MediaAttribution)
	return Exercise{
		ID:               model.ID,
		Name:             model.Name,
		Description:      model.Description,
		TargetMuscles:    model.TargetMuscles,
		Equipment:        model.Equipment,
		Difficulty:       model.Difficulty,
		VideoURL:         media.VideoURL,
		ImageURL:         media.ImageURL,
		MediaAttribution: media.Attribution,
		MediaAvailable:   media.Available,
	}
}
