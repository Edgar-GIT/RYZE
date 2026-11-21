package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ErrPublicProfileNotFound: no active, non-Test-Mode account has the username.
var ErrPublicProfileNotFound = errors.New("public profile not found")

// PublicUserRow is what a public profile shows of an account: never the
// email or the registered first and last name.
type PublicUserRow struct {
	ID          string
	Username    string
	DisplayName *string
	AvatarKey   *string
	CreatedAt   time.Time
}

// PublicProfileRepository reads public profiles and stores follows.
type PublicProfileRepository interface {
	FindByUsername(ctx context.Context, username string) (*PublicUserRow, error)
	FollowCounts(ctx context.Context, userID string) (followers, following int64, err error)
	CompletedWorkouts(ctx context.Context, userID string) (int64, error)
	TrainerIDOf(ctx context.Context, userID string) (string, error)
	IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error)
	Follow(ctx context.Context, followerID, followeeID string) error
	Unfollow(ctx context.Context, followerID, followeeID string) error
	Followers(ctx context.Context, userID string, limit int) ([]PublicUserRow, error)
	Following(ctx context.Context, userID string, limit int) ([]PublicUserRow, error)
}

type publicProfileRepository struct {
	db *gorm.DB
}

func NewPublicProfileRepository(db *gorm.DB) PublicProfileRepository {
	return &publicProfileRepository{db: db}
}

// publicAccount is an account that may appear in public.
const publicAccount = `u.deleted_at IS NULL AND u.test_persona IS NULL`

func (r *publicProfileRepository) FindByUsername(ctx context.Context, username string) (*PublicUserRow, error) {
	var rows []PublicUserRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id, u.username, u.display_name, u.avatar_key, u.created_at
		FROM users u
		WHERE u.active_username = ? AND `+publicAccount, username).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("find public profile: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrPublicProfileNotFound
	}
	return &rows[0], nil
}

func (r *publicProfileRepository) FollowCounts(ctx context.Context, userID string) (int64, int64, error) {
	var counts struct {
		Followers int64
		Following int64
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT
		  (SELECT COUNT(*) FROM user_follows f JOIN users u ON u.id = f.follower_id
		    WHERE f.followee_id = ? AND `+publicAccount+`) AS followers,
		  (SELECT COUNT(*) FROM user_follows f JOIN users u ON u.id = f.followee_id
		    WHERE f.follower_id = ? AND `+publicAccount+`) AS following`, userID, userID).Scan(&counts).Error
	if err != nil {
		return 0, 0, fmt.Errorf("count follows: %w", err)
	}
	return counts.Followers, counts.Following, nil
}

func (r *publicProfileRepository) CompletedWorkouts(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM workout_sessions
		WHERE user_id = ? AND status = 'completed' AND deleted_at IS NULL`, userID).Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count completed workouts: %w", err)
	}
	return count, nil
}

// TrainerIDOf returns the account's active trainer id, "" when it is not a
// trainer.
func (r *publicProfileRepository) TrainerIDOf(ctx context.Context, userID string) (string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`SELECT id FROM trainers WHERE user_id = ? AND deleted_at IS NULL LIMIT 1`, userID).Scan(&ids).Error
	if err != nil {
		return "", fmt.Errorf("find trainer: %w", err)
	}
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
}

func (r *publicProfileRepository) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM user_follows WHERE follower_id = ? AND followee_id = ?`,
		followerID, followeeID).Scan(&count).Error
	if err != nil {
		return false, fmt.Errorf("check follow: %w", err)
	}
	return count > 0, nil
}

// Follow is idempotent: following again changes nothing.
func (r *publicProfileRepository) Follow(ctx context.Context, followerID, followeeID string) error {
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO user_follows (follower_id, followee_id) VALUES (?, ?)
		ON CONFLICT (follower_id, followee_id) DO NOTHING`, followerID, followeeID).Error
	if err != nil {
		return fmt.Errorf("follow: %w", err)
	}
	return nil
}

func (r *publicProfileRepository) Unfollow(ctx context.Context, followerID, followeeID string) error {
	if err := r.db.WithContext(ctx).Exec(`DELETE FROM user_follows WHERE follower_id = ? AND followee_id = ?`,
		followerID, followeeID).Error; err != nil {
		return fmt.Errorf("unfollow: %w", err)
	}
	return nil
}

func (r *publicProfileRepository) Followers(ctx context.Context, userID string, limit int) ([]PublicUserRow, error) {
	var rows []PublicUserRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id, u.username, u.display_name, u.avatar_key, u.created_at
		FROM user_follows f JOIN users u ON u.id = f.follower_id
		WHERE f.followee_id = ? AND `+publicAccount+`
		ORDER BY f.created_at DESC LIMIT ?`, userID, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list followers: %w", err)
	}
	return rows, nil
}

func (r *publicProfileRepository) Following(ctx context.Context, userID string, limit int) ([]PublicUserRow, error) {
	var rows []PublicUserRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id, u.username, u.display_name, u.avatar_key, u.created_at
		FROM user_follows f JOIN users u ON u.id = f.followee_id
		WHERE f.follower_id = ? AND `+publicAccount+`
		ORDER BY f.created_at DESC LIMIT ?`, userID, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list following: %w", err)
	}
	return rows, nil
}
