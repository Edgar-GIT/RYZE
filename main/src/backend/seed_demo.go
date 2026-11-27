package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/curated_nutrition"
	"ryze/backend/services/generic_programs"
	"ryze/backend/services/member_feedback"
	"ryze/backend/services/password"
	"ryze/backend/services/program_details"
	uploads "ryze/backend/services/public_uploads"
)

// runSeedDemoCommand fills a development database with demo content (owner
// request, 2026-11-23): one published free plan per training category, ten
// published Premium Level 1 packages across the four objectives and
// audiences (to try the Level 1 assignment), ten demo members each with an
// approved feedback and a picture, and — when RYZE_DEMO_CLIENT_EMAIL and
// RYZE_DEMO_CLIENT_PASSWORD are set — a normal member account to log in with.
// Everything goes through the application's services and validation. It is
// idempotent: programs are matched by name, members by email.
func runSeedDemoCommand(db *gorm.DB, args []string) {
	flags := flag.NewFlagSet("seed-demo", flag.ContinueOnError)
	assets := flags.String("assets", "devdata/seed", "folder with the demo images")
	if err := flags.Parse(args); err != nil {
		fail("invalid seed-demo arguments: %v", err)
	}
	ctx := context.Background()

	uploadsCfg, err := config.LoadUploads()
	if err != nil {
		fail("uploads configuration: %v", err)
	}
	pricingCfg, err := config.LoadPricing()
	if err != nil {
		fail("pricing configuration: %v", err)
	}
	mediaCfg, err := config.LoadExerciseMedia()
	if err != nil {
		fail("exercise media configuration: %v", err)
	}
	store := uploads.NewStore(uploadsCfg.Dir, uploadsCfg.MaxImageBytes)
	if err := store.EnsureLayout(); err != nil {
		fail("uploads: %v", err)
	}
	seeder := &demoSeeder{
		db:       db,
		store:    store,
		assets:   *assets,
		programs: generic_programs.NewService(repositories.NewGenericProgramRepository(db), pricingCfg, mediaCfg),
	}
	if err := seeder.loadExercises(ctx); err != nil {
		fail("%v", err)
	}

	freeIDs := map[string]string{}
	for _, plan := range demoFreePlans {
		id, created, err := seeder.ensureProgram(ctx, models.ProgramProductTypeGeneric, plan)
		if err != nil {
			fail("free plan %q: %v", plan.name, err)
		}
		freeIDs[plan.trainingType] = id
		report("free plan", plan.name, created)
	}
	for _, plan := range demoLevel1Plans {
		_, created, err := seeder.ensureProgram(ctx, models.ProgramProductTypePremiumLevel1, plan)
		if err != nil {
			fail("level 1 package %q: %v", plan.name, err)
		}
		report("premium level 1", plan.name, created)
	}
	if err := seeder.seedFeedback(ctx, freeIDs); err != nil {
		fail("feedback: %v", err)
	}
	if err := seeder.seedDemoClient(ctx); err != nil {
		fail("demo client: %v", err)
	}
	fmt.Println("seed-demo: done")
}

func report(kind, name string, created bool) {
	state := "exists"
	if created {
		state = "created"
	}
	fmt.Printf("seed-demo: %-16s %-28s %s\n", kind, name, state)
}

type demoSeeder struct {
	db        *gorm.DB
	store     *uploads.Store
	assets    string
	programs  generic_programs.Service
	exercises []demoExercise
}

type demoExercise struct {
	ID       string
	Name     string
	BodyPart string
	Muscle   string
}

// demoDay is one training day: a focus and the words that pick its exercises.
type demoDay struct {
	name  string
	focus string
	words []string
}

type demoPlan struct {
	name         string
	description  string
	cover        string
	level        string
	trainingType string
	weeks        int
	days         []demoDay
	// Premium Level 1 only.
	objective string
	audience  string
	calories  int
}

var (
	dayPush      = demoDay{"Push", "push", []string{"bench", "chest", "press", "shoulder", "tricep", "dip", "fly"}}
	dayPull      = demoDay{"Pull", "pull", []string{"row", "pull", "lat", "back", "curl", "bicep", "chin"}}
	dayLegs      = demoDay{"Legs", "legs", []string{"squat", "lunge", "leg", "deadlift", "calf", "hip", "glute"}}
	dayUpper     = demoDay{"Upper body", "upper_body", []string{"press", "row", "pull", "shoulder", "chest", "curl"}}
	dayLower     = demoDay{"Lower body", "lower_body", []string{"squat", "deadlift", "lunge", "leg", "glute", "calf"}}
	dayFull      = demoDay{"Full body", "full_body", []string{"squat", "press", "row", "deadlift", "lunge", "push"}}
	dayGlutes    = demoDay{"Glutes", "glutes", []string{"glute", "hip", "bridge", "lunge", "squat", "abduct"}}
	dayCore      = demoDay{"Core", "core", []string{"plank", "crunch", "abs", "core", "twist", "raise"}}
	dayCardio    = demoDay{"Conditioning", "cardio", []string{"jump", "burpee", "run", "climber", "swing", "jack", "sprint"}}
	dayMobility  = demoDay{"Mobility", "mobility", []string{"stretch", "mobility", "rotation", "bridge", "cat"}}
	dayShoulders = demoDay{"Shoulders & arms", "arms", []string{"shoulder", "lateral", "curl", "tricep", "raise"}}
)

var demoFreePlans = []demoPlan{
	{name: "Hipertrofia Base", description: "Quatro semanas de hipertrofia em três dias: empurrar, puxar e pernas, com séries de 8 a 12 repetições.", cover: "free_hypertrophy", level: "Beginner", trainingType: "Hypertrophy", weeks: 4, days: []demoDay{dayPush, dayPull, dayLegs}},
	{name: "Força 4 Semanas", description: "Os grandes básicos com cargas pesadas e poucas repetições, para ganhares força com técnica.", cover: "free_strength", level: "Intermediate", trainingType: "Strength", weeks: 4, days: []demoDay{dayLower, dayUpper, dayFull}},
	{name: "HYROX Iniciação", description: "Condição física e força funcional para a tua primeira prova HYROX.", cover: "free_hyrox", level: "Beginner", trainingType: "HYROX", weeks: 4, days: []demoDay{dayCardio, dayFull, dayLegs}},
	{name: "CrossFit Essentials", description: "Movimentos fundamentais, ginástica e condição metabólica em sessões curtas e intensas.", cover: "free_crossfit", level: "Intermediate", trainingType: "CrossFit", weeks: 4, days: []demoDay{dayFull, dayCardio, dayCore}},
	{name: "Queimar 30", description: "Treinos de corpo inteiro e condição para perder gordura sem perder músculo.", cover: "free_fat_loss", level: "Beginner", trainingType: "Fat Loss", weeks: 4, days: []demoDay{dayFull, dayCardio, dayLower}},
	{name: "Treino em Casa", description: "Sem ginásio: peso corporal e halteres, três vezes por semana.", cover: "free_at_home", level: "Beginner", trainingType: "At Home", weeks: 4, days: []demoDay{dayFull, dayCore, dayMobility}},
}

var demoLevel1Plans = []demoPlan{
	{name: "Massa Total", description: "Pacote completo para ganhar músculo: treino em quatro dias e nutrição com excedente controlado.", cover: "l1_muscle_all", level: "Intermediate", trainingType: "Hypertrophy", weeks: 6, days: []demoDay{dayUpper, dayLower, dayPush, dayPull}, objective: models.ProgramObjectiveGainMuscle, calories: 2900},
	{name: "Glúteo & Força", description: "Ganhar músculo com foco em glúteos e pernas, com nutrição para recuperar bem.", cover: "l1_muscle_female", level: "Beginner", trainingType: "Hypertrophy", weeks: 6, days: []demoDay{dayGlutes, dayUpper, dayLower}, objective: models.ProgramObjectiveGainMuscle, audience: models.SexFemale, calories: 2200},
	{name: "V-Shape", description: "Costas largas e ombros: hipertrofia do tronco superior em cinco dias.", cover: "l1_muscle_male", level: "Advanced", trainingType: "Hypertrophy", weeks: 8, days: []demoDay{dayPush, dayPull, dayLegs, dayShoulders, dayUpper}, objective: models.ProgramObjectiveGainMuscle, audience: models.SexMale, calories: 3100},
	{name: "Secar 12", description: "Doze semanas para perder gordura: força para manter músculo, condição e défice moderado.", cover: "l1_fat_all", level: "Beginner", trainingType: "Fat Loss", weeks: 8, days: []demoDay{dayFull, dayCardio, dayFull}, objective: models.ProgramObjectiveLoseFat, calories: 1900},
	{name: "Definir", description: "Definição com treino de corpo inteiro e glúteos e uma nutrição simples de seguir.", cover: "l1_fat_female", level: "Intermediate", trainingType: "Fat Loss", weeks: 6, days: []demoDay{dayGlutes, dayCardio, dayUpper, dayCore}, objective: models.ProgramObjectiveLoseFat, audience: models.SexFemale, calories: 1700},
	{name: "Cut Pro", description: "Perder gordura mantendo a força: quatro dias de força e condição.", cover: "l1_fat_male", level: "Advanced", trainingType: "Fat Loss", weeks: 6, days: []demoDay{dayUpper, dayLower, dayCardio, dayFull}, objective: models.ProgramObjectiveLoseFat, audience: models.SexMale, calories: 2300},
	{name: "Força 5x5", description: "Força e condição com o método 5x5 e nutrição para treinos pesados.", cover: "l1_strength_all", level: "Intermediate", trainingType: "Strength", weeks: 8, days: []demoDay{dayLower, dayUpper, dayFull}, objective: models.ProgramObjectiveStrengthConditioning, calories: 2600},
	{name: "Powerbuild", description: "Força máxima nos básicos e volume para crescer, em quatro dias.", cover: "l1_strength_male", level: "Advanced", trainingType: "Strength", weeks: 8, days: []demoDay{dayLower, dayPush, dayPull, dayFull}, objective: models.ProgramObjectiveStrengthConditioning, audience: models.SexMale, calories: 3000},
	{name: "Futebol Performance", description: "Força, velocidade e prevenção de lesões para futebolistas, fora da época e na época.", cover: "l1_sport_all", level: "Intermediate", trainingType: "Strength", weeks: 6, days: []demoDay{dayLower, dayCardio, dayFull}, objective: models.ProgramObjectiveSportSpecific, calories: 2700},
	{name: "Padel Ready", description: "Agilidade, core e ombros fortes para jogares padel melhor e sem dores.", cover: "l1_sport_female", level: "Beginner", trainingType: "At Home", weeks: 6, days: []demoDay{dayCore, dayShoulders, dayLegs}, objective: models.ProgramObjectiveSportSpecific, audience: models.SexFemale, calories: 2000},
}

func (s *demoSeeder) loadExercises(ctx context.Context) error {
	err := s.db.WithContext(ctx).Raw(`
SELECT id, name, LOWER(COALESCE(body_part, '')) AS body_part, LOWER(COALESCE(primary_muscle_group, '')) AS muscle
FROM exercises WHERE deleted_at IS NULL ORDER BY name`).Scan(&s.exercises).Error
	if err != nil {
		return fmt.Errorf("failed to load exercises: %w", err)
	}
	if len(s.exercises) < 8 {
		return errors.New("the exercise catalogue is empty: import exercises first (go run . import-exercises --file ...)")
	}
	return nil
}

// pick chooses count different exercises matching the day's words, varying
// with offset, and fills up from the whole catalogue when needed.
func (s *demoSeeder) pick(words []string, count, offset int) []string {
	matches := make([]string, 0, 32)
	for _, exercise := range s.exercises {
		haystack := strings.ToLower(exercise.Name) + " " + exercise.BodyPart + " " + exercise.Muscle
		for _, word := range words {
			if strings.Contains(haystack, word) {
				matches = append(matches, exercise.ID)
				break
			}
		}
	}
	chosen := make([]string, 0, count)
	seen := map[string]bool{}
	for i := 0; len(chosen) < count && i < len(matches); i++ {
		id := matches[(offset*7+i*5)%len(matches)]
		if !seen[id] {
			seen[id] = true
			chosen = append(chosen, id)
		}
	}
	for i := 0; len(chosen) < count && i < len(s.exercises); i++ {
		id := s.exercises[(offset*11+i*13)%len(s.exercises)].ID
		if !seen[id] {
			seen[id] = true
			chosen = append(chosen, id)
		}
	}
	return chosen
}

func intPtr(value int) *int { return &value }

func (s *demoSeeder) weeks(plan demoPlan, programIndex int) []generic_programs.WeekInput {
	weeks := make([]generic_programs.WeekInput, 0, plan.weeks)
	for week := 1; week <= plan.weeks; week++ {
		workouts := make([]generic_programs.WorkoutInput, 0, len(plan.days))
		for position, day := range plan.days {
			exercises := []generic_programs.ExerciseInput{}
			for i, id := range s.pick(day.words, 5, programIndex*3+position) {
				reps, sets := 10, 3
				if plan.trainingType == "Strength" && i < 2 {
					reps, sets = 5, 5
				}
				prescription := make([]generic_programs.SetInput, 0, sets+1)
				if i == 0 {
					prescription = append(prescription, generic_programs.SetInput{SetType: "warmup", Reps: intPtr(12), RestSeconds: intPtr(60)})
				}
				for set := 0; set < sets; set++ {
					prescription = append(prescription, generic_programs.SetInput{SetType: "working", Reps: intPtr(reps), RestSeconds: intPtr(90)})
				}
				exercises = append(exercises, generic_programs.ExerciseInput{ExerciseID: id, Prescription: prescription})
			}
			workouts = append(workouts, generic_programs.WorkoutInput{
				Position:    position + 1,
				Name:        day.name,
				Description: fmt.Sprintf("%s — semana %d.", day.name, week),
				Focus:       day.focus,
				Exercises:   exercises,
			})
		}
		weeks = append(weeks, generic_programs.WeekInput{WeekNumber: week, Workouts: workouts})
	}
	return weeks
}

// demoNutrition is a simple curated day around a calorie target.
func demoNutrition(calories int) *curated_nutrition.PlanInput {
	share := func(part float64) int { return int(float64(calories) * part) }
	item := func(food, category string, quantity float64, unit string, kcal, protein, carbs, fat int) curated_nutrition.ItemInput {
		return curated_nutrition.ItemInput{FoodName: food, Category: category, Quantity: quantity, Unit: unit, Calories: kcal, ProteinGrams: protein, CarbohydrateGrams: carbs, FatGrams: fat}
	}
	return &curated_nutrition.PlanInput{
		DietaryPattern:  "balanced",
		Summary:         fmt.Sprintf("Cerca de %d kcal por dia em quatro refeições, com proteína em todas.", calories),
		PrepGuidance:    "Prepara as proteínas e os hidratos para três dias de uma vez; troca alimentos dentro da mesma categoria.",
		HydrationLitres: 2.5,
		HydrationNote:   "Mais meio litro por cada hora de treino.",
		Cautions:        []string{"Ajusta as quantidades se tiveres alguma condição médica ou alergia."},
		Meals: []curated_nutrition.MealInput{
			{Label: "Pequeno-almoço", Kind: models.NutritionPlanMealKindMeal, Items: []curated_nutrition.ItemInput{
				item("Aveia", "carbs", 80, "g", share(0.12), 10, 50, 6),
				item("Iogurte grego", "protein", 200, "g", share(0.08), 18, 8, 6),
				item("Fruta da época", "fruit", 150, "g", share(0.04), 1, 18, 0),
			}},
			{Label: "Almoço", Kind: models.NutritionPlanMealKindMeal, Items: []curated_nutrition.ItemInput{
				item("Peito de frango", "protein", 180, "g", share(0.1), 40, 0, 6),
				item("Arroz", "carbs", 120, "g", share(0.15), 8, 90, 1),
				item("Legumes salteados", "vegetables", 200, "g", share(0.04), 4, 14, 4),
			}},
			{Label: "Lanche", Kind: models.NutritionPlanMealKindSnack, Items: []curated_nutrition.ItemInput{
				item("Pão integral com queijo fresco", "carbs", 1, "dose", share(0.12), 18, 40, 8),
			}},
			{Label: "Jantar", Kind: models.NutritionPlanMealKindMeal, Items: []curated_nutrition.ItemInput{
				item("Salmão", "protein", 160, "g", share(0.15), 34, 0, 20),
				item("Batata-doce", "carbs", 250, "g", share(0.12), 4, 50, 0),
				item("Salada", "vegetables", 150, "g", share(0.03), 2, 6, 4),
			}},
		},
	}
}

// ensureProgram creates and publishes a plan unless one with the same name
// already exists in the family.
func (s *demoSeeder) ensureProgram(ctx context.Context, productType string, plan demoPlan) (string, bool, error) {
	var existing []string
	if err := s.db.WithContext(ctx).Raw(`SELECT id FROM programs WHERE name = ? AND product_type = ? AND deleted_at IS NULL LIMIT 1`,
		plan.name, productType).Scan(&existing).Error; err != nil {
		return "", false, err
	}
	if len(existing) > 0 {
		return existing[0], false, nil
	}
	folder := program_details.CoverFolderGeneric
	if productType == models.ProgramProductTypePremiumLevel1 {
		folder = program_details.CoverFolderLevel1
	}
	cover, err := s.saveAsset(func(file *os.File) (string, error) { return s.store.SaveProgramCover(ctx, folder, file) }, plan.cover)
	if err != nil {
		return "", false, err
	}
	index := 0
	for i, candidate := range append(append([]demoPlan{}, demoFreePlans...), demoLevel1Plans...) {
		if candidate.name == plan.name {
			index = i
		}
	}
	input := generic_programs.ProgramInput{
		Name:             plan.name,
		Description:      plan.description,
		Status:           models.ProgramStatusDraft,
		Level:            plan.level,
		DurationWeeks:    plan.weeks,
		FrequencyPerWeek: len(plan.days),
		TrainingType:     plan.trainingType,
		CoverImage:       cover,
		Audience:         plan.audience,
		Weeks:            s.weeks(plan, index),
	}
	if productType == models.ProgramProductTypePremiumLevel1 {
		input.Objective = plan.objective
		input.NutritionMode = models.ProgramNutritionModeCurated
		input.CuratedNutrition = demoNutrition(plan.calories)
	}
	created, err := s.programs.CreateProgram(ctx, productType, input)
	if err != nil {
		return "", false, err
	}
	if _, err := s.programs.PublishProgram(ctx, productType, created.ID); err != nil {
		return "", false, fmt.Errorf("publish: %w", err)
	}
	return created.ID, true, nil
}

func (s *demoSeeder) saveAsset(save func(*os.File) (string, error), name string) (string, error) {
	file, err := os.Open(filepath.Join(s.assets, name+".jpg"))
	if err != nil {
		return "", fmt.Errorf("demo image %s: %w", name, err)
	}
	defer file.Close()
	return save(file)
}

type demoMember struct {
	first, last, username string
	rating                int
	plan                  string
	message               string
}

var demoMembers = []demoMember{
	{"Ana", "Silva", "demo.ana", 5, "Fat Loss", "Perdi 8 kg em 12 semanas sem passar fome. Os treinos são curtos e o progresso aparece todas as semanas no gráfico."},
	{"Rui", "Costa", "demo.rui", 5, "Strength", "Cheguei aos 100 kg no supino! O registo de cada série e as medalhas de recorde deixam-me sempre com vontade de voltar."},
	{"Marta", "Ferreira", "demo.marta", 5, "Hypertrophy", "90 dias seguidos a treinar. A home mostra-me sempre o próximo treino e assim não tenho desculpas."},
	{"Tiago", "Santos", "demo.tiago", 4, "Strength", "Novo recorde no agachamento: 140 kg. Gostava de mais vídeos, mas os GIFs ajudam muito na técnica."},
	{"Inês", "Pereira", "demo.ines", 5, "HYROX", "Fiz a minha primeira prova HYROX com o plano de iniciação. Cheguei ao fim inteira e já estou inscrita na próxima."},
	{"João", "Oliveira", "demo.joao", 4, "Strength", "O método 5x5 funciona mesmo. A força sobe de semana para semana e a app faz as contas por mim."},
	{"Sofia", "Rodrigues", "demo.sofia", 5, "Hypertrophy", "Ganhei 4 kg de músculo limpo. A recuperação muscular diz-me exatamente o que posso treinar hoje."},
	{"Miguel", "Almeida", "demo.miguel", 5, "Fat Loss", "Já corro 10 km sem parar. Comecei pelo plano grátis e passei para o Premium com nutrição."},
	{"Carolina", "Martins", "demo.carolina", 4, "At Home", "Treino em casa só com halteres e chega perfeitamente. Ideal para quem tem pouco tempo."},
	{"Pedro", "Gomes", "demo.pedro", 5, "CrossFit", "Um ano de RYZE. A melhor parte é ver o histórico todo e perceber o quanto evoluí."},
}

// seedFeedback creates the demo members, gives each a free plan and an
// approved feedback with a picture.
func (s *demoSeeder) seedFeedback(ctx context.Context, freeIDs map[string]string) error {
	users := repositories.NewUserRepository(s.db)
	feedback := member_feedback.NewService(repositories.NewMemberFeedbackRepository(s.db), users, member_feedback.WithImages(s.store))
	activations := repositories.NewFreeProgramActivationRepository(s.db)
	for index, member := range demoMembers {
		email := member.username + "@example.com"
		userID, err := s.ensureUser(ctx, email, member.first, member.last, member.username, randomSecret())
		if err != nil {
			return err
		}
		if planID := freeIDs[member.plan]; planID != "" {
			if err := activations.Activate(ctx, userID, planID, &models.FreeProgramActivation{}); err != nil && !errors.Is(err, repositories.ErrFreeActivationAlreadyActive) {
				return fmt.Errorf("activate plan for %s: %w", member.username, err)
			}
		}
		image, err := os.Open(filepath.Join(s.assets, fmt.Sprintf("feedback%02d.jpg", index+1)))
		if err != nil {
			return err
		}
		created, err := feedback.Submit(ctx, userID, member_feedback.SubmitInput{Rating: member.rating, Message: member.message, Image: image})
		image.Close()
		if errors.Is(err, member_feedback.ErrAlreadyPending) {
			report("feedback", member.first+" "+member.last, false)
			continue
		}
		if err != nil {
			return fmt.Errorf("feedback of %s: %w", member.username, err)
		}
		if _, err := feedback.Review(ctx, created.ID, "ADMIN_2", member_feedback.DecisionApprove); err != nil {
			return fmt.Errorf("approve feedback of %s: %w", member.username, err)
		}
		report("feedback", member.first+" "+member.last, true)
	}
	return nil
}

// seedDemoClient creates the normal member account named in the environment.
func (s *demoSeeder) seedDemoClient(ctx context.Context) error {
	email := strings.TrimSpace(os.Getenv("RYZE_DEMO_CLIENT_EMAIL"))
	secret := os.Getenv("RYZE_DEMO_CLIENT_PASSWORD")
	if email == "" || secret == "" {
		fmt.Println("seed-demo: RYZE_DEMO_CLIENT_EMAIL / RYZE_DEMO_CLIENT_PASSWORD not set; no demo client account")
		return nil
	}
	username := strings.TrimSpace(os.Getenv("RYZE_DEMO_CLIENT_USERNAME"))
	if username == "" {
		username = "cliente.demo"
	}
	if _, err := s.ensureUser(ctx, email, "Cliente", "Demo", username, secret); err != nil {
		return err
	}
	report("demo client", "@"+username, true)
	return nil
}

// ensureUser returns the active account with the email, creating it (email
// verified, @username chosen) when missing. Existing accounts are left as
// they are.
func (s *demoSeeder) ensureUser(ctx context.Context, email, first, last, username, secret string) (string, error) {
	var ids []string
	if err := s.db.WithContext(ctx).Raw(`SELECT id FROM users WHERE email = ? AND deleted_at IS NULL LIMIT 1`, email).Scan(&ids).Error; err != nil {
		return "", err
	}
	if len(ids) > 0 {
		return ids[0], nil
	}
	hash, err := password.HashPassword(secret)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	now := time.Now().UTC()
	user := &models.User{
		Email:           email,
		PasswordHash:    hash,
		FirstName:       first,
		LastName:        last,
		Username:        username,
		UsernameChosen:  true,
		EmailVerifiedAt: &now,
	}
	if err := repositories.NewUserRepository(s.db).Create(ctx, user); err != nil {
		return "", fmt.Errorf("create %s: %w", email, err)
	}
	return user.ID, nil
}

// randomSecret is an unusable random password for demo members (they are
// never logged in to).
func randomSecret() string {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer)
}
