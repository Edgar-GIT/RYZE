// Package trainer_earnings reports what a trainer earned from the sales of
// their plans: the gross sales, the trainer's share and the commission
// RYZE kept (owner request, 2026-11-21).
package trainer_earnings

import (
	"context"
	"errors"
	"strings"

	"ryze/backend/repositories"
)

var ErrInvalidTrainer = errors.New("invalid trainer")

// Totals are the earnings of one currency, in minor units (cents).
type Totals struct {
	Currency   string `json:"currency"`
	Sales      int64  `json:"sales"`
	Gross      int64  `json:"gross"`
	Net        int64  `json:"net"`
	Commission int64  `json:"commission"`
}

// Report lists the totals per currency (empty when nothing was sold yet).
type Report struct {
	Totals []Totals `json:"totals"`
}

type Service struct {
	repo repositories.TrainerEarningsRepository
}

func NewService(repo repositories.TrainerEarningsRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Report(ctx context.Context, trainerID string) (*Report, error) {
	if strings.TrimSpace(trainerID) == "" {
		return nil, ErrInvalidTrainer
	}
	rows, err := s.repo.Totals(ctx, trainerID)
	if err != nil {
		return nil, err
	}
	report := &Report{Totals: make([]Totals, 0, len(rows))}
	for _, row := range rows {
		report.Totals = append(report.Totals, Totals(row))
	}
	return report, nil
}
