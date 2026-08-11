package adminroles

import (
	"errors"

	"ryze/backend/config"
)

// ErrUnknownAdminRole is returned by RoleForAdminID when the identity is not a
// configured administrator (ADMIN_1/ADMIN_2).
var ErrUnknownAdminRole = errors.New("unknown admin identity")

// Role identifies the responsibilities of a configured administrator.
type Role string

const (
	// RoleTechnicalAdministrator owns the platform's technical side: system
	// management, infrastructure, technical configuration, security and
	// development.
	RoleTechnicalAdministrator Role = "TECHNICAL_ADMINISTRATOR"
	// RoleManagementAdministrator owns the platform's business side: plans,
	// finance and marketing.
	RoleManagementAdministrator Role = "MANAGEMENT_ADMINISTRATOR"
)

// Permission is a single authorization unit granted to a role.
type Permission string

const (
	PermissionUsers                  Permission = "users"
	PermissionTrainers               Permission = "trainers"
	PermissionStatistics             Permission = "statistics"
	PermissionPlans                  Permission = "plans"
	PermissionFinance                Permission = "finance"
	PermissionMarketing              Permission = "marketing"
	PermissionSystem                 Permission = "system"
	PermissionInfrastructure         Permission = "infrastructure"
	PermissionTechnicalConfiguration Permission = "technical-configuration"
	PermissionSecurity               Permission = "security"
	PermissionDevelopment            Permission = "development"
)

// rolePermissions maps each role to its granted permissions. Users, trainers
// and statistics are shared by both roles. Technical concerns (system,
// infrastructure, technical configuration, security, development) belong
// exclusively to the Technical Administrator and business concerns (plans,
// finance, marketing) exclusively to the Management Administrator. Neither
// role is a superset of the other: the Technical Administrator does not
// automatically receive management permissions.
var rolePermissions = map[Role][]Permission{
	RoleTechnicalAdministrator: {
		PermissionUsers,
		PermissionTrainers,
		PermissionStatistics,
		PermissionSystem,
		PermissionInfrastructure,
		PermissionTechnicalConfiguration,
		PermissionSecurity,
		PermissionDevelopment,
	},
	RoleManagementAdministrator: {
		PermissionUsers,
		PermissionTrainers,
		PermissionStatistics,
		PermissionPlans,
		PermissionFinance,
		PermissionMarketing,
	},
}

// Permissions returns the permissions granted by the role.
func Permissions(role Role) []Permission {
	return rolePermissions[role]
}

// RoleForAdminID maps the configured administrator identity to its role:
// ADMIN_1 is the Technical Administrator and ADMIN_2 is the Management
// Administrator. Unknown identities are rejected so authorization can only
// ever derive from a configured admin identity.
func RoleForAdminID(adminID string) (Role, error) {
	switch adminID {
	case config.Admin1ID:
		return RoleTechnicalAdministrator, nil
	case config.Admin2ID:
		return RoleManagementAdministrator, nil
	default:
		return "", ErrUnknownAdminRole
	}
}

// HasRole reports whether the administrator identified by adminID holds any of
// the given roles. Identities that are not configured administrators never
// match.
func HasRole(adminID string, roles ...Role) bool {
	role, err := RoleForAdminID(adminID)
	if err != nil {
		return false
	}
	for _, candidate := range roles {
		if role == candidate {
			return true
		}
	}
	return false
}

// HasPermission reports whether the administrator identified by adminID holds
// the given permission.
func HasPermission(adminID string, permission Permission) bool {
	role, err := RoleForAdminID(adminID)
	if err != nil {
		return false
	}
	for _, granted := range rolePermissions[role] {
		if granted == permission {
			return true
		}
	}
	return false
}
