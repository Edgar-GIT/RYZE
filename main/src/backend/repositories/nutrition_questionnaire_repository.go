package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	// ErrNutritionQuestionnaireNotFound indicates the questionnaire does not
	// exist, is soft-deleted or does not belong to the requested user.
	ErrNutritionQuestionnaireNotFound = errors.New("nutrition questionnaire not found")
	// ErrNutritionQuestionnaireAlreadyExists indicates an active questionnaire
	// already exists for the same (user, program) pair. The one-active-
	// questionnaire rule is enforced at the database level by the unique index
	// on the active_questionnaire generated column.
	ErrNutritionQuestionnaireAlreadyExists = errors.New("active nutrition questionnaire already exists")
)

// NutritionQuestionnaireRepository defines the data-access operations for the
// questionnaire entity. Every operation receives the user id explicitly; the
// repository never obtains it from an HTTP context, so a client-supplied user id
// can never influence a query.
//
// Reads that do not need the answer document go through the
// v_nutrition_questionnaire_summary view, so the write model stays out of the
// contract surface that the service layer maps into API responses.
type NutritionQuestionnaireRepository interface {
	UpsertForUserAndProgram(ctx context.Context, userID, programID string, answers []byte) (*models.NutritionQuestionnaire, error)
	FindByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionQuestionnaire, error)
	FindSummaryByUserAndProgram(ctx context.Context, userID, programID string) (*NutritionQuestionnaireSummary, error)
}

type nutritionQuestionnaireRepository struct {
	db *gorm.DB
}

func NewNutritionQuestionnaireRepository(db *gorm.DB) NutritionQuestionnaireRepository {
	return &nutritionQuestionnaireRepository{db: db}
}

// NutritionQuestionnaireSummary is the view-backed read model of a submitted
// questionnaire. It deliberately omits the answer document: status and revision
// are what the client needs in order to know whether it must still answer, and
// the answers themselves are only ever loaded by the owner-scoped read.
type NutritionQuestionnaireSummary struct {
	ID          string    `gorm:"column:id" json:"id"`
	UserID      string    `gorm:"column:user_id" json:"user_id"`
	ProgramID   string    `gorm:"column:program_id" json:"program_id"`
	Version     int       `gorm:"column:version" json:"version"`
	SubmittedAt time.Time `gorm:"column:submitted_at" json:"submitted_at"`
}

// UpsertForUserAndProgram stores the validated answers as the single active
// questionnaire for the (user, program) pair. A first submission inserts the
// row; a resubmission rewrites it in place and increments the version, which is
// how the nutrition assignment detects that a previously generated plan is
// based on an outdated intake.
//
// The insert and the version bump run in one statement so a concurrent
// resubmission cannot lose an increment, and a losing concurrent writer is
// retried against the row the winner left behind. Rows that were previously
// soft-deleted are reused instead of being resurrected through the active
// unique index.
func (r *nutritionQuestionnaireRepository) UpsertForUserAndProgram(ctx context.Context, userID, programID string, answers []byte) (*models.NutritionQuestionnaire, error) {
	db := r.db.WithContext(ctx)

	for attempt := 0; attempt < 2; attempt++ {
		var existing models.NutritionQuestionnaire
		err := db.
			Unscoped().
			Where("user_id = ? AND program_id = ?", userID, programID).
			Order("deleted_at IS NULL DESC, version DESC, id ASC").
			First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			created := &models.NutritionQuestionnaire{
				UserID:    userID,
				ProgramID: programID,
				Version:   1,
				Answers:   answers,
			}
			if createErr := db.Create(created).Error; createErr != nil {
				if isDuplicateEntry(createErr) {
					// A concurrent submission won the race; fall through to
					// the update path on the next attempt.
					continue
				}
				return nil, fmt.Errorf("failed to create nutrition questionnaire: %w", createErr)
			}
			return created, nil
		case err != nil:
			return nil, fmt.Errorf("failed to load nutrition questionnaire: %w", err)
		}

		restore := existing.DeletedAt.Valid
		if restore {
			if err := db.Unscoped().Model(&models.NutritionQuestionnaire{}).
				Where("id = ?", existing.ID).
				Update("deleted_at", nil).Error; err != nil {
				return nil, fmt.Errorf("failed to restore nutrition questionnaire: %w", err)
			}
		}

		updated := &models.NutritionQuestionnaire{}
		if err := db.Model(&models.NutritionQuestionnaire{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{
				"answers":      answers,
				"version":      gorm.Expr("version + 1"),
				"submitted_at": time.Now().UTC(),
			}).Error; err != nil {
			return nil, fmt.Errorf("failed to update nutrition questionnaire: %w", err)
		}
		if err := db.Where("id = ?", existing.ID).First(updated).Error; err != nil {
			return nil, fmt.Errorf("failed to reload nutrition questionnaire: %w", err)
		}
		return updated, nil
	}

	return nil, ErrNutritionQuestionnaireAlreadyExists
}

// FindByUserAndProgram returns the active questionnaire for the given user and
// program pair, including the answer document. A missing, soft-deleted or
// cross-user questionnaire is indistinguishable from a missing one.
func (r *nutritionQuestionnaireRepository) FindByUserAndProgram(ctx context.Context, userID, programID string) (*models.NutritionQuestionnaire, error) {
	var questionnaire models.NutritionQuestionnaire
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND program_id = ?", userID, programID).
		First(&questionnaire).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNutritionQuestionnaireNotFound
		}
		return nil, fmt.Errorf("failed to find nutrition questionnaire: %w", err)
	}
	return &questionnaire, nil
}

// FindSummaryByUserAndProgram returns the view-backed summary of the active
// questionnaire, without loading the answer document.
func (r *nutritionQuestionnaireRepository) FindSummaryByUserAndProgram(ctx context.Context, userID, programID string) (*NutritionQuestionnaireSummary, error) {
	var summary NutritionQuestionnaireSummary
	if err := r.db.WithContext(ctx).
		Table("v_nutrition_questionnaire_summary").
		Where("user_id = ? AND program_id = ?", userID, programID).
		Take(&summary).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNutritionQuestionnaireNotFound
		}
		return nil, fmt.Errorf("failed to find nutrition questionnaire summary: %w", err)
	}
	return &summary, nil
}

// LockQuestionnaireForUpdate re-reads the questionnaire row inside a
// transaction with a row lock, so a generation run always reads a revision that
// cannot change while it is in flight. It is exposed on the concrete
// repository rather than the interface because it is a transaction-internal
// detail, not a service-layer operation.
func (r *nutritionQuestionnaireRepository) LockQuestionnaireForUpdate(ctx context.Context, id string) (*models.NutritionQuestionnaire, error) {
	var questionnaire models.NutritionQuestionnaire
	if err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&questionnaire).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNutritionQuestionnaireNotFound
		}
		return nil, fmt.Errorf("failed to lock nutrition questionnaire: %w", err)
	}
	return &questionnaire, nil
}
