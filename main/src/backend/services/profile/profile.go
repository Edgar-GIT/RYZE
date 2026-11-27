// Package profile manages the identity people see (owner request,
// 2026-11-20): a display name, the unique @username (the account's public
// reference) and the profile photo.
package profile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"ryze/backend/models"
	"ryze/backend/repositories"
	uploads "ryze/backend/services/public_uploads"
)

var (
	ErrNotFound           = errors.New("account not found")
	ErrInvalidDisplayName = errors.New("invalid display name")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrUsernameTaken      = errors.New("username taken")
	// ErrUsernameLocked: a chosen @username never changes (owner decision,
	// 2026-11-23); only a generated placeholder is replaced, once.
	ErrUsernameLocked = errors.New("username cannot change")
	// ErrNicknameCooldown: the nickname changes at most once every 7 days.
	ErrNicknameCooldown   = errors.New("nickname changed recently")
	ErrInvalidImage       = errors.New("invalid image")
	ErrImageTooLarge      = errors.New("image too large")
	ErrStorageUnavailable = errors.New("photo storage unavailable")
)

const MaxDisplayNameLength = 40

// NicknameCooldown is the time between two nickname changes.
const NicknameCooldown = 7 * 24 * time.Hour

// NextNicknameChange is when the nickname may change again (nil: now).
func NextNicknameChange(user *models.User, now time.Time) *time.Time {
	if user == nil || user.DisplayNameChangedAt == nil {
		return nil
	}
	next := user.DisplayNameChangedAt.Add(NicknameCooldown)
	if !next.After(now) {
		return nil
	}
	return &next
}

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._]{2,29}$`)

// reserved names cannot be chosen (they could pass for RYZE itself).
var reserved = map[string]bool{
	"admin": true, "administrator": true, "ryze": true, "ryzeapp": true, "support": true, "help": true,
	"root": true, "system": true, "staff": true, "moderator": true, "official": true, "trainer": true,
	"coach": true, "api": true, "me": true, "settings": true, "null": true, "undefined": true,
}

// Images stores profile photos.
type Images interface {
	SaveAvatar(ctx context.Context, content io.Reader) (string, error)
	DeleteAvatar(key string) error
}

// Service is the profile contract.
type Service interface {
	UpdateProfile(ctx context.Context, userID, displayName, username string) (*models.User, error)
	// ChooseUsername records the @username the person chose (first visit of
	// an account whose @username was generated, or any later change).
	ChooseUsername(ctx context.Context, userID, username string) (*models.User, error)
	SetAvatar(ctx context.Context, userID string, content io.Reader) (*models.User, error)
	RemoveAvatar(ctx context.Context, userID string) (*models.User, error)
}

type service struct {
	users  repositories.ProfileRepository
	images Images
}

func New(users repositories.ProfileRepository, images Images) Service {
	return &service{users: users, images: images}
}

// IsReserved reports whether a username cannot be chosen.
func IsReserved(username string) bool {
	return reserved[username]
}

// NormalizeDisplayName trims, collapses inner spaces and validates a display
// name; "" means "no display name". ok is false for anything invalid.
func NormalizeDisplayName(raw string) (string, bool) {
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" {
		return "", true
	}
	if utf8.RuneCountInString(name) > MaxDisplayNameLength {
		return "", false
	}
	for _, r := range name {
		if unicode.IsControl(r) || r == '<' || r == '>' {
			return "", false
		}
	}
	return name, true
}

// NormalizeUsername lowercases a username (an optional leading @ is ignored)
// and validates its shape. ok is false for anything invalid.
func NormalizeUsername(raw string) (string, bool) {
	name := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(raw), "@"))
	if !usernamePattern.MatchString(name) || strings.Contains(name, "..") || strings.HasSuffix(name, ".") {
		return "", false
	}
	return name, true
}

func (s *service) UpdateProfile(ctx context.Context, userID, displayName, username string) (*models.User, error) {
	user, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	name, ok := NormalizeDisplayName(displayName)
	if !ok {
		return nil, ErrInvalidDisplayName
	}
	handle := user.Username
	if strings.TrimSpace(username) != "" {
		normalized, ok := NormalizeUsername(username)
		if !ok {
			return nil, ErrInvalidUsername
		}
		if normalized != user.Username {
			// A chosen @username is permanent; a generated one is replaced
			// once, by a name that is not reserved.
			if user.UsernameChosen {
				return nil, ErrUsernameLocked
			}
			if reserved[normalized] {
				return nil, ErrInvalidUsername
			}
		}
		handle = normalized
	}
	var display *string
	if name != "" {
		display = &name
	}
	current := ""
	if user.DisplayName != nil {
		current = *user.DisplayName
	}
	nicknameChanged := name != current
	if nicknameChanged && NextNicknameChange(user, time.Now()) != nil {
		return nil, ErrNicknameCooldown
	}
	if err := s.users.UpdateProfile(ctx, userID, display, handle, nicknameChanged); err != nil {
		switch {
		case errors.Is(err, repositories.ErrUsernameTaken):
			return nil, ErrUsernameTaken
		case errors.Is(err, repositories.ErrUserNotFound):
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.load(ctx, userID)
}

func (s *service) ChooseUsername(ctx context.Context, userID, username string) (*models.User, error) {
	user, err := s.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.UsernameChosen {
		return nil, ErrUsernameLocked
	}
	handle, ok := NormalizeUsername(username)
	if !ok || reserved[handle] {
		return nil, ErrInvalidUsername
	}
	if err := s.users.ChooseUsername(ctx, userID, handle); err != nil {
		switch {
		case errors.Is(err, repositories.ErrUsernameTaken):
			return nil, ErrUsernameTaken
		case errors.Is(err, repositories.ErrUserNotFound):
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.load(ctx, userID)
}

func (s *service) SetAvatar(ctx context.Context, userID string, content io.Reader) (*models.User, error) {
	if _, err := s.load(ctx, userID); err != nil {
		return nil, err
	}
	if s.images == nil {
		return nil, ErrStorageUnavailable
	}
	key, err := s.images.SaveAvatar(ctx, content)
	if err != nil {
		switch {
		case errors.Is(err, uploads.ErrTooLarge):
			return nil, ErrImageTooLarge
		case errors.Is(err, uploads.ErrEmpty), errors.Is(err, uploads.ErrUnsupportedType), errors.Is(err, uploads.ErrInvalidImage):
			return nil, ErrInvalidImage
		}
		return nil, fmt.Errorf("failed to store avatar: %w", err)
	}
	previous, err := s.users.SetAvatar(ctx, userID, &key)
	if err != nil {
		_ = s.images.DeleteAvatar(key)
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.deleteFile(previous)
	return s.load(ctx, userID)
}

func (s *service) RemoveAvatar(ctx context.Context, userID string) (*models.User, error) {
	previous, err := s.users.SetAvatar(ctx, userID, nil)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.deleteFile(previous)
	return s.load(ctx, userID)
}

func (s *service) deleteFile(key *string) {
	if key == nil || s.images == nil {
		return
	}
	if err := s.images.DeleteAvatar(*key); err != nil {
		slog.Warn("previous avatar not deleted", slog.String("error", err.Error()))
	}
}

func (s *service) load(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}
