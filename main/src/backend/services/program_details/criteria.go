package program_details

import (
	"fmt"

	"ryze/backend/models"
)

// Criteria says who a Premium Level 1 package is for (v2.133.0): optional
// height, weight and age ranges, matched against the Level 1 questionnaire.
// A nil bound means "any". The bounds are those the questionnaire accepts.
type Criteria struct {
	MinHeightCm *int `json:"min_height_cm"`
	MaxHeightCm *int `json:"max_height_cm"`
	MinWeightKg *int `json:"min_weight_kg"`
	MaxWeightKg *int `json:"max_weight_kg"`
	MinAge      *int `json:"min_age"`
	MaxAge      *int `json:"max_age"`
}

const (
	MinCriteriaHeightCm = 120
	MaxCriteriaHeightCm = 230
	MinCriteriaWeightKg = 30
	MaxCriteriaWeightKg = 300
	MinCriteriaAge      = 14
	MaxCriteriaAge      = 100
)

// IsEmpty reports whether no range is set (the package suits everyone).
func (c Criteria) IsEmpty() bool {
	return c.MinHeightCm == nil && c.MaxHeightCm == nil && c.MinWeightKg == nil &&
		c.MaxWeightKg == nil && c.MinAge == nil && c.MaxAge == nil
}

// ValidateCriteria checks the ranges; only a Premium Level 1 package has them.
func ValidateCriteria(productType string, c Criteria) error {
	if c.IsEmpty() {
		return nil
	}
	if productType != models.ProgramProductTypePremiumLevel1 {
		return fmt.Errorf("%w: only a Premium Level 1 package has height, weight and age ranges", ErrInvalid)
	}
	ranges := []struct {
		name     string
		min, max *int
		low, top int
		unit     string
	}{
		{"height", c.MinHeightCm, c.MaxHeightCm, MinCriteriaHeightCm, MaxCriteriaHeightCm, "cm"},
		{"weight", c.MinWeightKg, c.MaxWeightKg, MinCriteriaWeightKg, MaxCriteriaWeightKg, "kg"},
		{"age", c.MinAge, c.MaxAge, MinCriteriaAge, MaxCriteriaAge, "years"},
	}
	for _, r := range ranges {
		for _, bound := range []*int{r.min, r.max} {
			if bound != nil && (*bound < r.low || *bound > r.top) {
				return fmt.Errorf("%w: the %s range must be between %d and %d %s", ErrInvalid, r.name, r.low, r.top, r.unit)
			}
		}
		if r.min != nil && r.max != nil && *r.min > *r.max {
			return fmt.Errorf("%w: the minimum %s is above the maximum", ErrInvalid, r.name)
		}
	}
	return nil
}

// CriteriaOf reads the ranges stored on a programme.
func CriteriaOf(p *models.Program) Criteria {
	return Criteria{
		MinHeightCm: p.MinHeightCm, MaxHeightCm: p.MaxHeightCm,
		MinWeightKg: p.MinWeightKg, MaxWeightKg: p.MaxWeightKg,
		MinAge: p.MinAge, MaxAge: p.MaxAge,
	}
}

// Apply writes the ranges onto a programme.
func (c Criteria) Apply(p *models.Program) {
	p.MinHeightCm, p.MaxHeightCm = c.MinHeightCm, c.MaxHeightCm
	p.MinWeightKg, p.MaxWeightKg = c.MinWeightKg, c.MaxWeightKg
	p.MinAge, p.MaxAge = c.MinAge, c.MaxAge
}

// Updates is the column map of the ranges, for partial updates.
func (c Criteria) Updates() map[string]any {
	return map[string]any{
		"min_height_cm": c.MinHeightCm, "max_height_cm": c.MaxHeightCm,
		"min_weight_kg": c.MinWeightKg, "max_weight_kg": c.MaxWeightKg,
		"min_age": c.MinAge, "max_age": c.MaxAge,
	}
}
