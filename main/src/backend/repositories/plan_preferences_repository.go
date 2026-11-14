package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ryze/backend/models"
)

// PlanPreferences is the member's sex and whether they prefer to see the
// plans made for it first (nil: not answered yet).
type PlanPreferences struct {
	Sex         *string
	FilterBySex *bool
}

// PlanPreferencesRepository reads and writes the member's plan preferences on
// the users row. It is separate from UserRepository so the account flows keep
// their narrow interface.
type PlanPreferencesRepository interface {
	Get(ctx context.Context, userID string) (PlanPreferences, error)
	Save(ctx context.Context, userID string, preferences PlanPreferences) error
}

type planPreferencesRepository struct {
	db *gorm.DB
}

func NewPlanPreferencesRepository(db *gorm.DB) PlanPreferencesRepository {
	return &planPreferencesRepository{db: db}
}

func (r *planPreferencesRepository) Get(ctx context.Context, userID string) (PlanPreferences, error) {
	var user models.User
	err := r.db.WithContext(ctx).Select("id", "sex", "plan_audience_filter").First(&user, "id = ?", userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return PlanPreferences{}, ErrUserNotFound
		}
		return PlanPreferences{}, fmt.Errorf("failed to load plan preferences: %w", err)
	}
	return PlanPreferences{Sex: user.Sex, FilterBySex: user.PlanAudienceFilter}, nil
}

// Save stores both values as given (the service merges them with the stored
// ones first).
func (r *planPreferencesRepository) Save(ctx context.Context, userID string, preferences PlanPreferences) error {
	result := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"sex":                  preferences.Sex,
		"plan_audience_filter": preferences.FilterBySex,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to save plan preferences: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}
