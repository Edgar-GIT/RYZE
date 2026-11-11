package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	ErrPremiumLevel2RequestNotFound = errors.New("premium level 2 request not found")
	ErrPremiumLevel2IntakeNotFound  = errors.New("premium level 2 intake not found")
	ErrPremiumLevel2PlanNotFound    = errors.New("premium level 2 nutrition plan not found")
	ErrPremiumLevel2InvalidState    = errors.New("premium level 2 invalid state")
)

// PremiumLevel2Filter narrows the administration/trainer request queues.
type PremiumLevel2Filter struct {
	Status    string
	Query     string
	TrainerID string
}

type PremiumLevel2Repository interface {
	CreateRequest(ctx context.Context, userID, programID string) (*models.PremiumLevel2Request, error)
	FindRequestByUserAndProgram(ctx context.Context, userID, programID string) (*models.PremiumLevel2Request, error)
	FindRequestByID(ctx context.Context, requestID string) (*models.PremiumLevel2Request, error)
	ListByUser(ctx context.Context, userID string) ([]models.PremiumLevel2Request, error)
	List(ctx context.Context, filter PremiumLevel2Filter, page, limit int) ([]models.PremiumLevel2Request, int64, error)
	CreateIntake(ctx context.Context, requestID string, answers []byte) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, error)
	FindLatestIntake(ctx context.Context, requestID string) (*models.PremiumLevel2Intake, error)
	AssignTrainer(ctx context.Context, requestID, trainerID, actorID string) (*models.PremiumLevel2Request, error)
	SetTrainingProgram(ctx context.Context, requestID, trainerID, programID string) (*models.PremiumLevel2Request, error)
	SaveNutritionPlan(ctx context.Context, requestID, trainerID string, plan *models.PremiumLevel2NutritionPlan) (*models.PremiumLevel2NutritionPlan, error)
	FindNutritionPlan(ctx context.Context, requestID string) (*models.PremiumLevel2NutritionPlan, error)
	Deliver(ctx context.Context, requestID, trainerID string) (*models.PremiumLevel2Request, error)
	RefreshReadiness(ctx context.Context, requestID string) error
	ListEvents(ctx context.Context, requestID string) ([]models.PremiumLevel2RequestEvent, error)
}

type premiumLevel2Repository struct {
	db *gorm.DB
}

func NewPremiumLevel2Repository(db *gorm.DB) PremiumLevel2Repository {
	return &premiumLevel2Repository{db: db}
}

func (r *premiumLevel2Repository) CreateRequest(ctx context.Context, userID, programID string) (*models.PremiumLevel2Request, error) {
	request := &models.PremiumLevel2Request{
		UserID:    userID,
		ProgramID: programID,
		Status:    models.PremiumLevel2StatusAwaitingIntake,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(request).Error; err != nil {
			if isDuplicateEntry(err) {
				return ErrPremiumLevel2RequestNotFound
			}
			return fmt.Errorf("failed to create premium level 2 request: %w", err)
		}
		return createPremiumLevel2Event(tx, request.ID, "system", "purchase", "request_created", "", request.Status)
	})
	if err != nil {
		if errors.Is(err, ErrPremiumLevel2RequestNotFound) {
			return r.FindRequestByUserAndProgram(ctx, userID, programID)
		}
		return nil, err
	}
	return request, nil
}

func (r *premiumLevel2Repository) FindRequestByUserAndProgram(ctx context.Context, userID, programID string) (*models.PremiumLevel2Request, error) {
	var request models.PremiumLevel2Request
	if err := r.db.WithContext(ctx).Unscoped().Table("v_premium_level_2_request").
		Where("user_id = ? AND program_id = ?", userID, programID).Take(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPremiumLevel2RequestNotFound
		}
		return nil, fmt.Errorf("failed to find premium level 2 request: %w", err)
	}
	return &request, nil
}

func (r *premiumLevel2Repository) FindRequestByID(ctx context.Context, requestID string) (*models.PremiumLevel2Request, error) {
	var request models.PremiumLevel2Request
	if err := r.db.WithContext(ctx).Unscoped().Table("v_premium_level_2_request").
		Where("id = ?", requestID).Take(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPremiumLevel2RequestNotFound
		}
		return nil, fmt.Errorf("failed to find premium level 2 request: %w", err)
	}
	return &request, nil
}

func (r *premiumLevel2Repository) ListByUser(ctx context.Context, userID string) ([]models.PremiumLevel2Request, error) {
	var requests []models.PremiumLevel2Request
	// The active entitlement is part of the read condition: a revoked purchase
	// hides the request exactly like a missing one, instead of leaking the
	// programme name and coaching status after access has been removed.
	if err := r.db.WithContext(ctx).Unscoped().Table("v_premium_level_2_request AS r").
		Select("r.*").
		Joins("JOIN entitlements e ON e.program_id = r.program_id AND e.user_id = r.user_id AND e.deleted_at IS NULL AND (e.expires_at IS NULL OR e.expires_at > NOW())").
		Where("r.user_id = ?", userID).Order("r.created_at DESC, r.id ASC").Find(&requests).Error; err != nil {
		return nil, fmt.Errorf("failed to list premium level 2 requests: %w", err)
	}
	return requests, nil
}

func (r *premiumLevel2Repository) List(ctx context.Context, filter PremiumLevel2Filter, page, limit int) ([]models.PremiumLevel2Request, int64, error) {
	query := r.db.WithContext(ctx).Unscoped().Table("v_premium_level_2_request")
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.TrainerID != "" {
		query = query.Where("trainer_id = ?", filter.TrainerID)
	}
	if strings.TrimSpace(filter.Query) != "" {
		escaped := escapeSQLLike(strings.TrimSpace(filter.Query))
		query = query.Where("id ILIKE ? OR user_id ILIKE ? OR program_id ILIKE ? OR offer_name ILIKE ? OR client_email ILIKE ? OR client_first_name ILIKE ? OR client_last_name ILIKE ?",
			"%"+escaped+"%", "%"+escaped+"%", "%"+escaped+"%", "%"+escaped+"%", "%"+escaped+"%", "%"+escaped+"%", "%"+escaped+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count premium level 2 requests: %w", err)
	}
	var requests []models.PremiumLevel2Request
	if err := query.Order("created_at DESC, id ASC").Limit(limit).Offset((page - 1) * limit).Find(&requests).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list premium level 2 requests: %w", err)
	}
	return requests, total, nil
}

func (r *premiumLevel2Repository) CreateIntake(ctx context.Context, requestID string, answers []byte) (*models.PremiumLevel2Request, *models.PremiumLevel2Intake, error) {
	var request models.PremiumLevel2Request
	var intake models.PremiumLevel2Intake
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPremiumLevel2Request(tx, requestID, &request); err != nil {
			return err
		}
		switch request.Status {
		case models.PremiumLevel2StatusAwaitingIntake, models.PremiumLevel2StatusAwaitingTrainerAssignment:
		default:
			return ErrPremiumLevel2InvalidState
		}
		intake = models.PremiumLevel2Intake{
			RequestID: requestID,
			Version:   request.IntakeVersion + 1,
			Answers:   json.RawMessage(answers),
		}
		if err := tx.Create(&intake).Error; err != nil {
			return fmt.Errorf("failed to create premium level 2 intake: %w", err)
		}
		from := request.Status
		request.IntakeVersion = intake.Version
		request.Status = models.PremiumLevel2StatusAwaitingTrainerAssignment
		if err := tx.Model(&models.PremiumLevel2Request{}).Where("id = ?", requestID).Updates(map[string]any{
			"intake_version": intake.Version,
			"status":         request.Status,
		}).Error; err != nil {
			return fmt.Errorf("failed to update premium level 2 request intake: %w", err)
		}
		return createPremiumLevel2Event(tx, requestID, "client", request.UserID, "intake_submitted", from, request.Status)
	})
	if err != nil {
		return nil, nil, err
	}
	return &request, &intake, nil
}

func (r *premiumLevel2Repository) FindLatestIntake(ctx context.Context, requestID string) (*models.PremiumLevel2Intake, error) {
	var intake models.PremiumLevel2Intake
	if err := r.db.WithContext(ctx).Unscoped().Table("v_premium_level_2_intake").
		Where("request_id = ?", requestID).Order("version DESC").Take(&intake).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPremiumLevel2IntakeNotFound
		}
		return nil, fmt.Errorf("failed to find premium level 2 intake: %w", err)
	}
	return &intake, nil
}

func (r *premiumLevel2Repository) AssignTrainer(ctx context.Context, requestID, trainerID, actorID string) (*models.PremiumLevel2Request, error) {
	var request models.PremiumLevel2Request
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPremiumLevel2Request(tx, requestID, &request); err != nil {
			return err
		}
		if request.Status == models.PremiumLevel2StatusDelivered || request.Status == models.PremiumLevel2StatusCancelled {
			return ErrPremiumLevel2InvalidState
		}
		if request.IntakeVersion == 0 {
			return ErrPremiumLevel2InvalidState
		}
		var trainer models.Trainer
		if err := tx.First(&trainer, "id = ?", trainerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPremiumLevel2InvalidState
			}
			return fmt.Errorf("failed to verify trainer: %w", err)
		}
		var relation models.TrainerClient
		err := tx.Unscoped().Where("trainer_id = ? AND user_id = ?", trainerID, request.UserID).First(&relation).Error
		switch {
		case err == nil:
			if relation.DeletedAt.Valid {
				if err := tx.Unscoped().Model(&models.TrainerClient{}).Where("id = ?", relation.ID).Updates(map[string]any{"deleted_at": nil, "updated_at": time.Now()}).Error; err != nil {
					return fmt.Errorf("failed to restore trainer-client relationship: %w", err)
				}
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			relation = models.TrainerClient{TrainerID: trainerID, UserID: request.UserID}
			if err := tx.Create(&relation).Error; err != nil {
				return fmt.Errorf("failed to create trainer-client relationship: %w", err)
			}
		default:
			return fmt.Errorf("failed to inspect trainer-client relationship: %w", err)
		}
		from := request.Status
		updates := map[string]any{
			"trainer_id": trainerID,
			"status":     models.PremiumLevel2StatusInProgress,
		}
		// A reassignment must not carry the previous trainer's programme into
		// delivery. The old programme belongs to another owner and cannot be
		// edited by the new trainer, so the new trainer starts by attaching
		// their own draft. Keeping the existing trainer leaves their work intact.
		if request.TrainerID == nil || *request.TrainerID != trainerID {
			updates["training_program_id"] = nil
		}
		request.TrainerID = &trainerID
		request.Status = models.PremiumLevel2StatusInProgress
		if err := tx.Model(&models.PremiumLevel2Request{}).Where("id = ?", requestID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to assign premium level 2 trainer: %w", err)
		}
		return createPremiumLevel2Event(tx, requestID, "admin", actorID, "trainer_assigned", from, request.Status)
	})
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *premiumLevel2Repository) SetTrainingProgram(ctx context.Context, requestID, trainerID, programID string) (*models.PremiumLevel2Request, error) {
	var request models.PremiumLevel2Request
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPremiumLevel2Request(tx, requestID, &request); err != nil {
			return err
		}
		if request.TrainerID == nil || *request.TrainerID != trainerID {
			return ErrPremiumLevel2RequestNotFound
		}
		if request.Status == models.PremiumLevel2StatusDelivered || request.Status == models.PremiumLevel2StatusCancelled {
			return ErrPremiumLevel2InvalidState
		}
		var program models.Program
		if err := tx.First(&program, "id = ? AND trainer_id = ? AND deleted_at IS NULL", programID, trainerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProgramNotFound
			}
			return fmt.Errorf("failed to verify training program: %w", err)
		}
		if program.Status != models.ProgramStatusDraft {
			return ErrPremiumLevel2InvalidState
		}
		var exerciseCount int64
		if err := tx.Model(&models.WorkoutExercise{}).
			Joins("JOIN program_workouts w ON w.id = workout_exercises.program_workout_id").
			Joins("JOIN program_weeks pw ON pw.id = w.program_week_id").
			Where("pw.program_id = ? AND workout_exercises.deleted_at IS NULL AND w.deleted_at IS NULL AND pw.deleted_at IS NULL", programID).
			Count(&exerciseCount).Error; err != nil {
			return fmt.Errorf("failed to inspect training program content: %w", err)
		}
		if exerciseCount == 0 {
			return ErrPremiumLevel2InvalidState
		}
		from := request.Status
		request.TrainingProgramID = &programID
		if err := tx.Model(&models.PremiumLevel2Request{}).Where("id = ?", requestID).Update("training_program_id", programID).Error; err != nil {
			return fmt.Errorf("failed to attach training program: %w", err)
		}
		return createPremiumLevel2Event(tx, requestID, "trainer", trainerID, "training_program_attached", from, from)
	})
	if err != nil {
		return nil, err
	}
	if err := r.RefreshReadiness(ctx, requestID); err != nil {
		return nil, err
	}
	return r.FindRequestByID(ctx, requestID)
}

func (r *premiumLevel2Repository) SaveNutritionPlan(ctx context.Context, requestID, trainerID string, plan *models.PremiumLevel2NutritionPlan) (*models.PremiumLevel2NutritionPlan, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var request models.PremiumLevel2Request
		if err := lockPremiumLevel2Request(tx, requestID, &request); err != nil {
			return err
		}
		if request.TrainerID == nil || *request.TrainerID != trainerID {
			return ErrPremiumLevel2RequestNotFound
		}
		if request.Status == models.PremiumLevel2StatusDelivered || request.Status == models.PremiumLevel2StatusCancelled {
			return ErrPremiumLevel2InvalidState
		}
		var existing models.PremiumLevel2NutritionPlan
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", requestID).Take(&existing).Error
		version := 1
		switch {
		case err == nil:
			version = existing.Version + 1
			if err := tx.Where("plan_id = ?", existing.ID).Delete(&models.PremiumLevel2NutritionPlanMealItem{}).Error; err != nil {
				return fmt.Errorf("failed to replace nutrition foods: %w", err)
			}
			if err := tx.Where("plan_id = ?", existing.ID).Delete(&models.PremiumLevel2NutritionPlanMeal{}).Error; err != nil {
				return fmt.Errorf("failed to replace nutrition meals: %w", err)
			}
			if err := tx.Model(&models.PremiumLevel2NutritionPlan{}).Where("id = ?", existing.ID).Updates(map[string]any{
				"version":          version,
				"status":           "draft",
				"dietary_pattern":  plan.DietaryPattern,
				"summary":          plan.Summary,
				"prep_guidance":    plan.PrepGuidance,
				"hydration_litres": plan.HydrationLitres,
				"hydration_note":   plan.HydrationNote,
				"cautions":         plan.Cautions,
			}).Error; err != nil {
				return fmt.Errorf("failed to update nutrition plan: %w", err)
			}
			plan.ID = existing.ID
		case errors.Is(err, gorm.ErrRecordNotFound):
			plan.RequestID = requestID
			plan.Version = version
			plan.Status = "draft"
			if err := tx.Omit("Meals").Create(plan).Error; err != nil {
				return fmt.Errorf("failed to create nutrition plan: %w", err)
			}
		default:
			return fmt.Errorf("failed to load nutrition plan: %w", err)
		}
		for mi := range plan.Meals {
			meal := &plan.Meals[mi]
			meal.PlanID = plan.ID
			meal.ID = ""
			if err := tx.Omit("Items").Create(meal).Error; err != nil {
				return fmt.Errorf("failed to create nutrition meal: %w", err)
			}
			for ii := range meal.Items {
				item := &meal.Items[ii]
				item.ID = ""
				item.PlanID = plan.ID
				item.MealID = meal.ID
				if err := tx.Create(item).Error; err != nil {
					return fmt.Errorf("failed to create nutrition food: %w", err)
				}
			}
		}
		return createPremiumLevel2Event(tx, requestID, "trainer", trainerID, "nutrition_saved", request.Status, request.Status)
	})
	if err != nil {
		return nil, err
	}
	if err := r.RefreshReadiness(ctx, requestID); err != nil {
		return nil, err
	}
	return r.FindNutritionPlan(ctx, requestID)
}

func (r *premiumLevel2Repository) FindNutritionPlan(ctx context.Context, requestID string) (*models.PremiumLevel2NutritionPlan, error) {
	var plan models.PremiumLevel2NutritionPlan
	if err := r.db.WithContext(ctx).Unscoped().Table("v_premium_level_2_nutrition_plan").
		Where("request_id = ?", requestID).Take(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPremiumLevel2PlanNotFound
		}
		return nil, fmt.Errorf("failed to find nutrition plan: %w", err)
	}
	var meals []models.PremiumLevel2NutritionPlanMeal
	if err := r.db.WithContext(ctx).Unscoped().Table("v_premium_level_2_nutrition_plan_meal").
		Where("plan_id = ?", plan.ID).Order("position ASC").Find(&meals).Error; err != nil {
		return nil, fmt.Errorf("failed to find nutrition meals: %w", err)
	}
	var items []models.PremiumLevel2NutritionPlanMealItem
	if err := r.db.WithContext(ctx).Unscoped().Table("v_premium_level_2_nutrition_plan_meal_item").
		Where("plan_id = ?", plan.ID).Order("position ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to find nutrition foods: %w", err)
	}
	byMeal := make(map[string][]models.PremiumLevel2NutritionPlanMealItem, len(meals))
	for _, item := range items {
		byMeal[item.MealID] = append(byMeal[item.MealID], item)
	}
	for i := range meals {
		meals[i].Items = byMeal[meals[i].ID]
	}
	plan.Meals = meals
	return &plan, nil
}

func (r *premiumLevel2Repository) Deliver(ctx context.Context, requestID, trainerID string) (*models.PremiumLevel2Request, error) {
	var request models.PremiumLevel2Request
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPremiumLevel2Request(tx, requestID, &request); err != nil {
			return err
		}
		if request.TrainerID == nil || *request.TrainerID != trainerID {
			return ErrPremiumLevel2RequestNotFound
		}
		if request.Status == models.PremiumLevel2StatusDelivered {
			return nil
		}
		if request.Status != models.PremiumLevel2StatusReadyForDelivery || request.TrainingProgramID == nil {
			return ErrPremiumLevel2InvalidState
		}
		var plan models.PremiumLevel2NutritionPlan
		if err := tx.Where("request_id = ?", requestID).Take(&plan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPremiumLevel2InvalidState
			}
			return fmt.Errorf("failed to load nutrition plan for delivery: %w", err)
		}
		var active models.Entitlement
		err := tx.Where("user_id = ? AND program_id = ?", request.UserID, *request.TrainingProgramID).Take(&active).Error
		switch {
		case err == nil:
		case errors.Is(err, gorm.ErrRecordNotFound):
			var deleted models.Entitlement
			err := tx.Unscoped().Where("user_id = ? AND program_id = ?", request.UserID, *request.TrainingProgramID).Order("created_at DESC").Take(&deleted).Error
			switch {
			case err == nil:
				if err := tx.Unscoped().Model(&models.Entitlement{}).Where("id = ?", deleted.ID).Updates(map[string]any{"deleted_at": nil, "expires_at": nil}).Error; err != nil {
					return fmt.Errorf("failed to restore delivered entitlement: %w", err)
				}
			case errors.Is(err, gorm.ErrRecordNotFound):
				entitlement := &models.Entitlement{UserID: request.UserID, ProgramID: *request.TrainingProgramID}
				// A concurrent delivery may have created the entitlement already;
				// the savepoint keeps this transaction usable after that duplicate.
				if err := inSavepoint(tx, func(sp *gorm.DB) error { return sp.Create(entitlement).Error }); err != nil && !isDuplicateEntry(err) {
					return fmt.Errorf("failed to create delivered entitlement: %w", err)
				}
			default:
				return fmt.Errorf("failed to inspect delivered entitlement: %w", err)
			}
		default:
			return fmt.Errorf("failed to inspect delivered entitlement: %w", err)
		}
		now := time.Now()
		from := request.Status
		request.Status = models.PremiumLevel2StatusDelivered
		request.DeliveredAt = &now
		if err := tx.Model(&models.PremiumLevel2Request{}).Where("id = ?", requestID).Updates(map[string]any{
			"status":       request.Status,
			"delivered_at": now,
		}).Error; err != nil {
			return fmt.Errorf("failed to mark request delivered: %w", err)
		}
		if err := tx.Model(&models.PremiumLevel2NutritionPlan{}).Where("id = ?", plan.ID).Update("status", "delivered").Error; err != nil {
			return fmt.Errorf("failed to mark nutrition plan delivered: %w", err)
		}
		return createPremiumLevel2Event(tx, requestID, "trainer", trainerID, "delivered", from, request.Status)
	})
	if err != nil {
		return nil, err
	}
	return r.FindRequestByID(ctx, requestID)
}

func (r *premiumLevel2Repository) RefreshReadiness(ctx context.Context, requestID string) error {
	request, err := r.FindRequestByID(ctx, requestID)
	if err != nil {
		return err
	}
	if request.Status != models.PremiumLevel2StatusInProgress && request.Status != models.PremiumLevel2StatusReadyForDelivery {
		return nil
	}
	ready := request.TrainingProgramID != nil && request.IntakeVersion > 0
	if ready {
		var count int64
		if err := r.db.WithContext(ctx).Unscoped().Table("v_premium_level_2_nutrition_plan").Where("request_id = ?", requestID).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check nutrition readiness: %w", err)
		}
		ready = count > 0
	}
	status := models.PremiumLevel2StatusInProgress
	if ready {
		status = models.PremiumLevel2StatusReadyForDelivery
	}
	if status == request.Status {
		return nil
	}
	if err := r.db.WithContext(ctx).Model(&models.PremiumLevel2Request{}).Where("id = ?", requestID).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to refresh request readiness: %w", err)
	}
	return createPremiumLevel2Event(r.db.WithContext(ctx), requestID, "system", "system", "readiness_changed", request.Status, status)
}

func (r *premiumLevel2Repository) ListEvents(ctx context.Context, requestID string) ([]models.PremiumLevel2RequestEvent, error) {
	var events []models.PremiumLevel2RequestEvent
	if err := r.db.WithContext(ctx).Where("request_id = ?", requestID).Order("created_at ASC, id ASC").Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to list request events: %w", err)
	}
	return events, nil
}

func lockPremiumLevel2Request(tx *gorm.DB, requestID string, request *models.PremiumLevel2Request) error {
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", requestID).First(request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPremiumLevel2RequestNotFound
		}
		return fmt.Errorf("failed to load premium level 2 request: %w", err)
	}
	return nil
}

func createPremiumLevel2Event(tx *gorm.DB, requestID, actorKind, actorID, eventType, from, to string) error {
	event := &models.PremiumLevel2RequestEvent{
		RequestID: requestID,
		ActorKind: actorKind,
		ActorID:   actorID,
		EventType: eventType,
	}
	if from != "" {
		event.FromStatus = &from
	}
	if to != "" {
		event.ToStatus = &to
	}
	if err := tx.Create(event).Error; err != nil {
		return fmt.Errorf("failed to create premium level 2 event: %w", err)
	}
	return nil
}
