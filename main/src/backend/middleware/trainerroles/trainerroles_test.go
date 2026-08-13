package trainerroles_test

import (
	"errors"
	"testing"

	"ryze/backend/middleware/trainerroles"
)

func TestAllPermissionsDeterministic(t *testing.T) {
	first := trainerroles.AllPermissions()
	second := trainerroles.AllPermissions()

	if len(first) == 0 {
		t.Fatal("official permission set must not be empty")
	}
	if len(first) != len(second) {
		t.Fatalf("permission set must be deterministic, got %d and %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("permission set must be deterministic: %q != %q", first[i], second[i])
		}
	}
}

func TestAllPermissionsNoDuplicates(t *testing.T) {
	seen := make(map[trainerroles.Permission]bool)
	for _, permission := range trainerroles.AllPermissions() {
		if seen[permission] {
			t.Fatalf("duplicate permission %q", permission)
		}
		seen[permission] = true
	}
}

func TestAllPermissionsRecognized(t *testing.T) {
	for _, permission := range trainerroles.AllPermissions() {
		if !trainerroles.IsKnownPermission(permission) {
			t.Fatalf("expected %q to be recognized as official", permission)
		}
	}
}

func TestValidPermissionsGranted(t *testing.T) {
	for _, permission := range trainerroles.AllPermissions() {
		if !trainerroles.HasPermission(permission) {
			t.Fatalf("valid permission %q must be granted", permission)
		}
	}
}

func TestUnknownPermissionsDenied(t *testing.T) {
	for _, value := range []string{
		"billing",
		"trainer.billing",
		"client.profile",
		"trainer.pack",
		"TRAINER.PROFILE",
		"trainer.profile ",
		"",
	} {
		if trainerroles.HasPermission(trainerroles.Permission(value)) {
			t.Fatalf("unknown permission %q must not be granted", value)
		}
	}
}

func TestPermissionFromString(t *testing.T) {
	for _, permission := range trainerroles.AllPermissions() {
		parsed, err := trainerroles.PermissionFromString(string(permission))
		if err != nil {
			t.Fatalf("PermissionFromString(%q): %v", permission, err)
		}
		if parsed != permission {
			t.Fatalf("expected %q, got %q", permission, parsed)
		}
	}
}

func TestPermissionFromStringRejectsUnknown(t *testing.T) {
	for _, value := range []string{
		"billing",
		"trainer.billing",
		"trainer.pack",
		"TRAINER.PROFILE",
		"",
	} {
		if _, err := trainerroles.PermissionFromString(value); !errors.Is(err, trainerroles.ErrUnknownPermission) {
			t.Fatalf("PermissionFromString(%q): expected ErrUnknownPermission, got %v", value, err)
		}
	}
}

func TestInitialGrantMappingCoversOfficialSet(t *testing.T) {
	for _, permission := range trainerroles.AllPermissions() {
		if !trainerroles.HasPermission(permission) {
			t.Fatalf("initial grant mapping must include %q", permission)
		}
	}
}
