// Package session_photos lets a member add a photo of themselves to their own
// workout (owner request, 2026-11-22, as in Lyfta): before saving it, or later
// from the history. Only the owner can add, remove or see it; abandoned
// workouts take no photo.
package session_photos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"ryze/backend/models"
	"ryze/backend/repositories"
	uploads "ryze/backend/services/public_uploads"
)

var (
	ErrNotFound      = errors.New("workout not found")
	ErrNotAllowed    = errors.New("this workout cannot take a photo")
	ErrInvalidImage  = errors.New("invalid image")
	ErrImageTooLarge = errors.New("image too large")
	ErrNoPhoto       = errors.New("no photo")
)

// Images is the photo storage.
type Images interface {
	SaveSessionPhoto(ctx context.Context, content io.Reader) (string, error)
	DeleteSessionPhoto(key string) error
	SessionPhotoPath(key string) (string, bool)
}

type Service struct {
	repo   repositories.SessionPhotoRepository
	images Images
}

func NewService(repo repositories.SessionPhotoRepository, images Images) *Service {
	return &Service{repo: repo, images: images}
}

func (s *Service) owned(ctx context.Context, userID, sessionID string) (string, *string, error) {
	status, key, err := s.repo.Find(ctx, userID, sessionID)
	if errors.Is(err, repositories.ErrSessionPhotoSessionNotFound) {
		return "", nil, ErrNotFound
	}
	return status, key, err
}

// Set stores a new photo, replacing the previous one.
func (s *Service) Set(ctx context.Context, userID, sessionID string, content io.Reader) error {
	status, _, err := s.owned(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if status != models.WorkoutSessionStatusInProgress && status != models.WorkoutSessionStatusCompleted {
		return ErrNotAllowed
	}
	key, err := s.images.SaveSessionPhoto(ctx, content)
	if err != nil {
		switch {
		case errors.Is(err, uploads.ErrTooLarge):
			return ErrImageTooLarge
		case errors.Is(err, uploads.ErrEmpty), errors.Is(err, uploads.ErrUnsupportedType), errors.Is(err, uploads.ErrInvalidImage):
			return ErrInvalidImage
		}
		return fmt.Errorf("failed to store workout photo: %w", err)
	}
	previous, err := s.repo.Set(ctx, userID, sessionID, &key)
	if err != nil {
		_ = s.images.DeleteSessionPhoto(key)
		if errors.Is(err, repositories.ErrSessionPhotoSessionNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.deleteFile(previous)
	return nil
}

// Remove deletes the photo (no photo is not an error).
func (s *Service) Remove(ctx context.Context, userID, sessionID string) error {
	if _, _, err := s.owned(ctx, userID, sessionID); err != nil {
		return err
	}
	previous, err := s.repo.Set(ctx, userID, sessionID, nil)
	if err != nil {
		if errors.Is(err, repositories.ErrSessionPhotoSessionNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.deleteFile(previous)
	return nil
}

// Path returns the file of the owner's photo.
func (s *Service) Path(ctx context.Context, userID, sessionID string) (string, error) {
	_, key, err := s.owned(ctx, userID, sessionID)
	if err != nil {
		return "", err
	}
	if key == nil {
		return "", ErrNoPhoto
	}
	path, ok := s.images.SessionPhotoPath(*key)
	if !ok {
		return "", ErrNoPhoto
	}
	return path, nil
}

func (s *Service) deleteFile(key *string) {
	if key == nil {
		return
	}
	if err := s.images.DeleteSessionPhoto(*key); err != nil {
		slog.Warn("previous workout photo not deleted", slog.String("error", err.Error()))
	}
}
