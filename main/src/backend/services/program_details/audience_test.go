package program_details_test

import (
	"testing"

	"ryze/backend/models"
	"ryze/backend/services/program_details"
)

func TestValidateAudience(t *testing.T) {
	for _, ok := range []string{"", "female", "male", " male "} {
		if err := program_details.ValidateAudience(models.ProgramProductTypeGeneric, ok); err != nil {
			t.Fatalf("%q: %v", ok, err)
		}
	}
	if err := program_details.ValidateAudience(models.ProgramProductTypeGeneric, "kids"); err == nil {
		t.Fatal("an unknown audience must be refused")
	}
	if err := program_details.ValidateAudience(models.ProgramProductTypePremiumLevel2, "female"); err == nil {
		t.Fatal("a premium level 2 offer has no audience")
	}
	if program_details.OptionalAudience("") != nil || *program_details.OptionalAudience("female") != "female" {
		t.Fatal("OptionalAudience maps empty to nil")
	}
}
