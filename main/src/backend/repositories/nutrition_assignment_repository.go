package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	// ErrNutritionAssignmentNotFound indicates the assignment does not exist,
	// is soft-deleted or does not belong to the requested user.
	ErrNutritionAssignmentNotFound = errors.New("nutrition assignment not found")
	// ErrNutritionAssignmentAlreadyExists indicates an active assignment
	// already exists for the same (user, program) pair. The one-active-
	// assignment rule is enforced at the database level by the unique index on
	// the active_assignment generated column.
	ErrNutritionAssignmentAlreadyExists = errors.New("active nutrition assignment already exists")
	// ErrNutritionAssignmentNotRetryable indicates the assignment is not in a
	// state a generation retry can move forward, so the request is rejected
	// rather than silently producing a second plan.
	ErrNutritionAssignmentNotRetryable = errors.New("nutrition assignment is not retryable")
)

// NutritionAssignmentRepository defines the data-access operations for the
// nutrition assignment entity. Every operation receives the user id explicitly;
// the repository never obtains it from an HTTP context, so a client-supplied
// user id can never influence a query.
//
// The create path is idempotent by design: a verified purchase can be delivered
// more than once (provider webhook, client capture callback, Test Mode) and
// must still result in exactly one plan.
type NutritionAssignmentRepository interface {
	// EnsurePendingForUserAndProgram returns the single active assignment for
	// the pair, creating it in the pending state when it does not exist yet.
	// The returned boolean reports whether this call created the row.
	EnsurePendingForUserAndProgram(ctx context.Context, userID, programID, questionnaireID string, questionnaireVersion int) (*models.NutritionAssignment, bool, error)
	FindByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionAssignment, error)
	MarkProcessing(ctx context.Context, userID, programID string) error
	MarkCompleted(ctx context.Context, userID, programID string, configuration []byte, questionnaireVersion int) error
	MarkFailed(ctx context.Context, userID, programID, reason string) error
}

type nutritionAssignmentRepository struct {
	db *gorm.DB
}

func NewNutritionAssignmentRepository(db *gorm.DB) NutritionAssignmentRepository {
	return &nutritionAssignmentRepository{db: db}
}

// EnsurePendingForUserAndProgram returns the active assignment for the pair,
// inserting a pending placeholder when none exists. When an assignment already
// exists it is returned untouched, which is what makes a repeated purchase
// delivery safe.
func (r *nutritionAssignmentRepository) EnsurePendingForUserAndProgram(ctx context.Context, userID, programID, questionnaireID string, questionnaireVersion int) (*models.NutritionAssignment, bool, error) {
	if existing, err := r.FindByUserAndProgram(ctx, userID, programID); err == nil {
		return existing, false, nil
	} else if !errors.Is(err, ErrNutritionAssignmentNotFound) {
		return nil, false, err
	}

	created := &models.NutritionAssignment{
		UserID:               userID,
		ProgramID:            programID,
		QuestionnaireID:      questionnaireID,
		QuestionnaireVersion: questionnaireVersion,
		Status:               models.NutritionAssignmentStatusPending,
	}
	if err := r.db.WithContext(ctx).Create(created).Error; err != nil {
		if isDuplicateEntry(err) {
			// A concurrent purchase delivery won the race. The single active
			// assignment is the correct outcome either way.
			existing, findErr := r.FindByUserAndProgram(ctx, userID, programID)
			if findErr != nil {
				return nil, false, findErr
			}
			return existing, false, nil
		}
		return nil, false, fmt.Errorf("failed to create nutrition assignment: %w", err)
	}
	return created, true, nil
}

// FindByUserAndProgram returns the active assignment for the given user and
// program pair, or ErrNutritionAssignmentNotFound when none exists.
//
// The read goes through v_nutrition_assignment, which already excludes
// soft-deleted rows. Unscoped is therefore required rather than a soft-delete
// filter: the view does not project deleted_at, so re-applying GORM's implicit
// predicate would reference a column the view does not expose.
func (r *nutritionAssignmentRepository) FindByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionAssignment, error) {
	var assignment models.NutritionAssignment
	if err := r.db.WithContext(ctx).
		Unscoped().
		Table("v_nutrition_assignment").
		Where("user_id = ? AND program_id = ?", userID, programID).
		Take(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNutritionAssignmentNotFound
		}
		return nil, fmt.Errorf("failed to find nutrition assignment: %w", err)
	}
	return &assignment, nil
}

// MarkProcessing claims the assignment for generation. It only succeeds from a
// pending or failed state, so a second concurrent generator cannot start a run
// against an assignment that is already in flight or already finished.
func (r *nutritionAssignmentRepository) MarkProcessing(ctx context.Context, userID, programID string) error {
	result := r.db.WithContext(ctx).
		Model(&models.NutritionAssignment{}).
		Where(
			"user_id = ? AND program_id = ? AND status IN ?",
			userID, programID,
			[]string{models.NutritionAssignmentStatusPending, models.NutritionAssignmentStatusFailed},
		).
		Updates(map[string]any{
			"status":         models.NutritionAssignmentStatusProcessing,
			"failure_reason": nil,
		})
	if result.Error != nil {
		return fmt.Errorf("failed to mark nutrition assignment as processing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNutritionAssignmentNotRetryable
	}
	return nil
}

// MarkCompleted stores the generated configuration and stamps the generation
// time. The questionnaire version is re-asserted under a row lock so a
// completion can never claim to be current for a revision it did not read.
func (r *nutritionAssignmentRepository) MarkCompleted(ctx context.Context, userID, programID string, configuration []byte, questionnaireVersion int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var assignment models.NutritionAssignment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND program_id = ?", userID, programID).
			First(&assignment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNutritionAssignmentNotFound
			}
			return fmt.Errorf("failed to lock nutrition assignment: %w", err)
		}
		if assignment.QuestionnaireVersion != questionnaireVersion {
			return fmt.Errorf("%w: assignment targets questionnaire version %d", ErrNutritionAssignmentNotRetryable, assignment.QuestionnaireVersion)
		}
		if assignment.Status != models.NutritionAssignmentStatusProcessing {
			return ErrNutritionAssignmentNotRetryable
		}

		now := time.Now().UTC()
		if err := tx.Model(&models.NutritionAssignment{}).
			Where("id = ?", assignment.ID).
			Updates(map[string]any{
				"status":         models.NutritionAssignmentStatusCompleted,
				"configuration":  configuration,
				"failure_reason": nil,
				"generated_at":   now,
				"version":        gorm.Expr("version + 1"),
			}).Error; err != nil {
			return fmt.Errorf("failed to complete nutrition assignment: %w", err)
		}
		return nil
	})
}

// MarkFailed records the reason a generation run could not finish. The reason is
// internal diagnostics: it is stored for administrators and never projected
// into a client response.
func (r *nutritionAssignmentRepository) MarkFailed(ctx context.Context, userID, programID, reason string) error {
	result := r.db.WithContext(ctx).
		Model(&models.NutritionAssignment{}).
		Where("user_id = ? AND program_id = ?", userID, programID).
		Updates(map[string]any{
			"status":         models.NutritionAssignmentStatusFailed,
			"failure_reason": truncateFailureReason(reason),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to mark nutrition assignment as failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNutritionAssignmentNotFound
	}
	return nil
}

// truncateFailureReason keeps a stored diagnostic inside the column width. The
// reason is always a server-generated code, never questionnaire content, so the
// truncation can never cut a user answer short.
func truncateFailureReason(reason string) string {
	const maxReasonLength = 255
	if len(reason) <= maxReasonLength {
		return reason
	}
	return reason[:maxReasonLength]
}

// NutritionAssignmentConfiguration is the view-backed read model of a generated
// plan. FailureReason is intentionally absent: the safe read surface never
// carries internal diagnostics.
type NutritionAssignmentConfiguration struct {
	ID                   string          `gorm:"column:id" json:"id"`
	UserID               string          `gorm:"column:user_id" json:"user_id"`
	ProgramID            string          `gorm:"column:program_id" json:"program_id"`
	QuestionnaireID      string          `gorm:"column:questionnaire_id" json:"questionnaire_id"`
	QuestionnaireVersion int             `gorm:"column:questionnaire_version" json:"questionnaire_version"`
	Status               string          `gorm:"column:status" json:"status"`
	Version              int             `gorm:"column:version" json:"version"`
	Configuration        json.RawMessage `gorm:"column:configuration" json:"configuration"`
	GeneratedAt          *time.Time      `gorm:"column:generated_at" json:"generated_at"`
}
