package program_details_test

import (
	"errors"
	"testing"

	"ryze/backend/models"
	"ryze/backend/services/program_details"
)

func ptr(v int) *int { return &v }

func TestValidateCriteria(t *testing.T) {
	l1 := models.ProgramProductTypePremiumLevel1
	ok := []program_details.Criteria{
		{},
		{MinHeightCm: ptr(150), MaxHeightCm: ptr(180)},
		{MinWeightKg: ptr(50), MaxWeightKg: ptr(55)},
		{MinAge: ptr(18)},
		{MaxAge: ptr(40)},
	}
	for _, c := range ok {
		if err := program_details.ValidateCriteria(l1, c); err != nil {
			t.Fatalf("%+v should be valid: %v", c, err)
		}
	}
	bad := []program_details.Criteria{
		{MinHeightCm: ptr(180), MaxHeightCm: ptr(150)},
		{MinHeightCm: ptr(100)},
		{MaxWeightKg: ptr(400)},
		{MinAge: ptr(10)},
	}
	for _, c := range bad {
		if err := program_details.ValidateCriteria(l1, c); !errors.Is(err, program_details.ErrInvalid) {
			t.Fatalf("%+v should be refused, got %v", c, err)
		}
	}
	if err := program_details.ValidateCriteria(models.ProgramProductTypeGeneric, program_details.Criteria{MinAge: ptr(18)}); err == nil {
		t.Fatalf("a free plan has no ranges")
	}
	if err := program_details.ValidateCriteria(models.ProgramProductTypeGeneric, program_details.Criteria{}); err != nil {
		t.Fatalf("no ranges is fine for any plan: %v", err)
	}
}
