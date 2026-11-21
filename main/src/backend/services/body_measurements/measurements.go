// Package body_measurements lets a member log body measurements over time
// (owner request, 2026-11-20, as in Lyfta): weight in kg, body fat in % and
// circumferences in cm, one value per kind and day.
package body_measurements

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"ryze/backend/repositories"
)

var (
	ErrInvalidKind        = errors.New("body measurements: unknown kind")
	ErrInvalidValue       = errors.New("body measurements: value out of range")
	ErrInvalidDate        = errors.New("body measurements: invalid date")
	ErrMeasurementMissing = errors.New("body measurements: not found")
	ErrInvalidUser        = errors.New("body measurements: user id required")
)

// listLimit bounds one member's list (15 kinds a day for half a year).
const listLimit = 3000

type valueRange struct{ min, max float64 }

// Ranges are the accepted values of each kind (the database also checks
// the kinds).
var ranges = map[string]valueRange{
	"weight":        {20, 500},
	"body_fat":      {2, 75},
	"neck":          {10, 300},
	"shoulders":     {10, 300},
	"chest":         {10, 300},
	"biceps_left":   {10, 300},
	"biceps_right":  {10, 300},
	"forearm_left":  {10, 300},
	"forearm_right": {10, 300},
	"waist":         {10, 300},
	"hips":          {10, 300},
	"thigh_left":    {10, 300},
	"thigh_right":   {10, 300},
	"calf_left":     {10, 300},
	"calf_right":    {10, 300},
}

// Entry is one logged measurement.
type Entry struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"`
	Value      float64 `json:"value"`
	MeasuredOn string  `json:"measured_on"`
}

// Repository stores measurements.
type Repository interface {
	List(ctx context.Context, userID string, limit int) ([]repositories.BodyMeasurementRow, error)
	Save(ctx context.Context, userID, kind string, value float64, measuredOn time.Time) (*repositories.BodyMeasurementRow, error)
	Delete(ctx context.Context, userID, id string) (bool, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func toEntry(row repositories.BodyMeasurementRow) Entry {
	return Entry{ID: row.ID, Kind: row.Kind, Value: row.Value, MeasuredOn: row.MeasuredOn.Format("2006-01-02")}
}

// List returns the member's measurements, oldest first.
func (s *Service) List(ctx context.Context, userID string) ([]Entry, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidUser
	}
	rows, err := s.repo.List(ctx, userID, listLimit)
	if err != nil {
		return nil, fmt.Errorf("body measurements: %w", err)
	}
	entries := make([]Entry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, toEntry(row))
	}
	return entries, nil
}

// Save validates and records a value; the same kind on the same day replaces
// the earlier value. measuredOn is YYYY-MM-DD, not later than tomorrow (the
// member's day may be ahead of the server's) nor before 1950.
func (s *Service) Save(ctx context.Context, userID, kind string, value float64, measuredOn string) (*Entry, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidUser
	}
	kind = strings.TrimSpace(kind)
	accepted, ok := ranges[kind]
	if !ok {
		return nil, ErrInvalidKind
	}
	if math.IsNaN(value) || math.IsInf(value, 0) || value < accepted.min || value > accepted.max {
		return nil, ErrInvalidValue
	}
	value = math.Round(value*100) / 100
	day, err := time.Parse("2006-01-02", strings.TrimSpace(measuredOn))
	if err != nil {
		return nil, ErrInvalidDate
	}
	latest := s.now().UTC().AddDate(0, 0, 1)
	if day.Before(time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC)) || day.After(latest) {
		return nil, ErrInvalidDate
	}
	row, err := s.repo.Save(ctx, userID, kind, value, day)
	if err != nil {
		return nil, fmt.Errorf("body measurements: %w", err)
	}
	entry := toEntry(*row)
	return &entry, nil
}

// Delete removes one of the member's measurements.
func (s *Service) Delete(ctx context.Context, userID, id string) error {
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidUser
	}
	if _, err := uuid.Parse(id); err != nil {
		return ErrMeasurementMissing
	}
	deleted, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return fmt.Errorf("body measurements: %w", err)
	}
	if !deleted {
		return ErrMeasurementMissing
	}
	return nil
}
