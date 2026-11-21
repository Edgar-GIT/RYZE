package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ProfileDetailsRow is the optional "about" of a public profile.
type ProfileDetailsRow struct {
	Bio           *string
	BirthDate     *time.Time
	TrainingSince *int
	WeightKg      *float64
	// SocialLinks maps a platform to the stored handle.
	SocialLinks map[string]string
}

// ProfileDetailsRepository stores one row per account (absent: nothing set).
type ProfileDetailsRepository interface {
	Get(ctx context.Context, userID string) (*ProfileDetailsRow, error)
	Save(ctx context.Context, userID string, details ProfileDetailsRow) error
}

type profileDetailsRepository struct {
	db *gorm.DB
}

func NewProfileDetailsRepository(db *gorm.DB) ProfileDetailsRepository {
	return &profileDetailsRepository{db: db}
}

func (r *profileDetailsRepository) Get(ctx context.Context, userID string) (*ProfileDetailsRow, error) {
	var rows []struct {
		Bio           *string
		BirthDate     *time.Time
		TrainingSince *int
		WeightKg      *float64
		SocialLinks   []byte
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT bio, birth_date, training_since, weight_kg::float8 AS weight_kg, social_links
		FROM user_profile_details WHERE user_id = ?`, userID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load profile details: %w", err)
	}
	details := &ProfileDetailsRow{SocialLinks: map[string]string{}}
	if len(rows) == 0 {
		return details, nil
	}
	row := rows[0]
	details.Bio, details.BirthDate, details.TrainingSince, details.WeightKg = row.Bio, row.BirthDate, row.TrainingSince, row.WeightKg
	if len(row.SocialLinks) > 0 {
		if err := json.Unmarshal(row.SocialLinks, &details.SocialLinks); err != nil {
			return nil, fmt.Errorf("decode social links: %w", err)
		}
	}
	return details, nil
}

func (r *profileDetailsRepository) Save(ctx context.Context, userID string, details ProfileDetailsRow) error {
	links := details.SocialLinks
	if links == nil {
		links = map[string]string{}
	}
	encoded, err := json.Marshal(links)
	if err != nil {
		return fmt.Errorf("encode social links: %w", err)
	}
	var birth any
	if details.BirthDate != nil {
		birth = details.BirthDate.Format("2006-01-02")
	}
	err = r.db.WithContext(ctx).Exec(`
		INSERT INTO user_profile_details (user_id, bio, birth_date, training_since, weight_kg, social_links)
		VALUES (?, ?, ?, ?, ?, ?::jsonb)
		ON CONFLICT (user_id) DO UPDATE SET bio = EXCLUDED.bio, birth_date = EXCLUDED.birth_date,
		    training_since = EXCLUDED.training_since, weight_kg = EXCLUDED.weight_kg, social_links = EXCLUDED.social_links`,
		userID, details.Bio, birth, details.TrainingSince, details.WeightKg, string(encoded)).Error
	if err != nil {
		return fmt.Errorf("save profile details: %w", err)
	}
	return nil
}
