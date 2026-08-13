// Package trainerroles defines the trainer permission model.
//
// Trainer authorization is a separate concern from trainer authentication:
// authentication (`middleware/trainercontext` + `middleware.TrainerAuthenticate`)
// resolves an authenticated user into an active trainer, while this package
// centralizes the permissions an active trainer may hold. The package has no
// dependency on Gin, HTTP, the database, JWT or any repository.
package trainerroles

import "errors"

// Permission is a single authorization unit granted to a trainer.
type Permission string

// ErrUnknownPermission is returned by PermissionFromString when the string
// does not match any official trainer permission.
var ErrUnknownPermission = errors.New("unknown trainer permission")

const (
	// PermissionProfile covers the trainer's own profile and dashboard area.
	PermissionProfile Permission = "trainer.profile"
	// PermissionClients covers the trainer's client management area.
	PermissionClients Permission = "trainer.clients"
	// PermissionPrograms covers the trainer's programs area.
	PermissionPrograms Permission = "trainer.programs"
	// PermissionStatistics covers the trainer's statistics area.
	PermissionStatistics Permission = "trainer.statistics"
)

// officialPermissions is the official trainer permission set. It is also the
// initial grant mapping: every active trainer is granted every base
// permission. Future trainer tiers or roles may narrow the grant mapping
// without touching handlers or services.
var officialPermissions = []Permission{
	PermissionProfile,
	PermissionClients,
	PermissionPrograms,
	PermissionStatistics,
}

// AllPermissions returns the official trainer permission set in a stable,
// deterministic order. The returned slice is a copy and cannot mutate the
// official set.
func AllPermissions() []Permission {
	return append([]Permission(nil), officialPermissions...)
}

// IsKnownPermission reports whether the permission is part of the official
// trainer permission set. Unknown and empty permissions are never recognized.
func IsKnownPermission(permission Permission) bool {
	for _, known := range officialPermissions {
		if known == permission {
			return true
		}
	}
	return false
}

// PermissionFromString converts a string into a Permission only when the
// string is an official trainer permission. Any other value is rejected with
// ErrUnknownPermission, so no arbitrary string can become a valid trainer
// permission.
func PermissionFromString(value string) (Permission, error) {
	permission := Permission(value)
	if !IsKnownPermission(permission) {
		return "", ErrUnknownPermission
	}
	return permission, nil
}

// HasPermission reports whether an active trainer holds the permission.
//
// The initial grant mapping grants every official base permission to every
// active trainer, so official permissions return true and unknown permissions
// return false. This is the single place where the trainer grant mapping is
// expressed; the future RequireTrainerPermission middleware will rely on it
// and will never trust client-supplied permissions.
func HasPermission(permission Permission) bool {
	for _, granted := range officialPermissions {
		if granted == permission {
			return true
		}
	}
	return false
}
