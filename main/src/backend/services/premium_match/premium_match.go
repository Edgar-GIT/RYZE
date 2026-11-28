// Package premium_match picks a Premium Level 1 package for a client (owner
// request, 2026-11-14): the client chooses an objective and gives their sex,
// weight, age and height; RYZE chooses one published curated package with that
// objective, made for everyone or for that sex, at random, never one the client
// already owns, and records the pick.
package premium_match

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

var (
	// ErrInvalidInput indicates an unknown objective or body data out of range.
	ErrInvalidInput = errors.New("invalid premium level 1 match input")
	// ErrNoPackages indicates no package is published for the objective yet.
	ErrNoPackages = errors.New("no premium level 1 package for this objective")
	// ErrAllOwned indicates the client already owns every package of the
	// objective, so a new one cannot be given.
	ErrAllOwned = errors.New("every premium level 1 package of this objective is already owned")
)

// Input is the client's answer.
type Input struct {
	Objective string
	Sex       string
	WeightKg  float64
	HeightCm  float64
	Age       int
}

// Result is the recorded pick and the chosen package.
type Result struct {
	Match   models.PremiumLevel1Match
	Program models.Program
}

type Repository interface {
	CountPublishedPackages(ctx context.Context, profile repositories.MatchProfile) (int64, error)
	ListMatchCandidates(ctx context.Context, userID string, profile repositories.MatchProfile) ([]models.Program, error)
	CreateMatch(ctx context.Context, match *models.PremiumLevel1Match) error
}

type Service interface {
	Match(ctx context.Context, userID string, input Input) (*Result, error)
}

type service struct {
	repo Repository
	pick func(n int) int
}

func NewService(repo Repository) Service {
	return &service{repo: repo, pick: rand.IntN}
}

// NewServiceWithPicker is used by tests to make the random choice
// deterministic.
func NewServiceWithPicker(repo Repository, pick func(n int) int) Service {
	return &service{repo: repo, pick: pick}
}

// Validate checks the objective and the body data ranges (weight 30-300 kg,
// height 120-230 cm, age 14-100), matching the database constraints.
func Validate(input Input) error {
	valid := false
	for _, objective := range models.ProgramObjectives {
		if input.Objective == objective {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("%w: unknown objective", ErrInvalidInput)
	}
	if !models.ValidSex(input.Sex) {
		return fmt.Errorf("%w: sex must be female or male", ErrInvalidInput)
	}
	if math.IsNaN(input.WeightKg) || input.WeightKg < 30 || input.WeightKg > 300 {
		return fmt.Errorf("%w: weight must be between 30 and 300 kg", ErrInvalidInput)
	}
	if math.IsNaN(input.HeightCm) || input.HeightCm < 120 || input.HeightCm > 230 {
		return fmt.Errorf("%w: height must be between 120 and 230 cm", ErrInvalidInput)
	}
	if input.Age < 14 || input.Age > 100 {
		return fmt.Errorf("%w: age must be between 14 and 100", ErrInvalidInput)
	}
	return nil
}

func (s *service) Match(ctx context.Context, userID string, input Input) (*Result, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: user is required", ErrInvalidInput)
	}
	input.Objective = strings.TrimSpace(input.Objective)
	input.Sex = strings.TrimSpace(input.Sex)
	input.WeightKg = math.Round(input.WeightKg*10) / 10
	input.HeightCm = math.Round(input.HeightCm*10) / 10
	if err := Validate(input); err != nil {
		return nil, err
	}

	profile := repositories.MatchProfile{
		Objective: input.Objective, Sex: input.Sex,
		HeightCm: input.HeightCm, WeightKg: input.WeightKg, Age: input.Age,
	}
	candidates, err := s.repo.ListMatchCandidates(ctx, userID, profile)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		total, err := s.repo.CountPublishedPackages(ctx, profile)
		if err != nil {
			return nil, err
		}
		if total == 0 {
			return nil, ErrNoPackages
		}
		return nil, ErrAllOwned
	}

	chosen := candidates[s.pick(len(candidates))]
	match := models.PremiumLevel1Match{
		UserID:    userID,
		ProgramID: chosen.ID,
		Objective: input.Objective,
		WeightKg:  input.WeightKg,
		HeightCm:  input.HeightCm,
		Age:       input.Age,
		Sex:       &input.Sex,
	}
	if err := s.repo.CreateMatch(ctx, &match); err != nil {
		return nil, err
	}
	return &Result{Match: match, Program: chosen}, nil
}

var _ Repository = (repositories.PremiumJourneyRepository)(nil)
