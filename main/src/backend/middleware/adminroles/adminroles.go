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
	// PermissionUsers is the coarse user-management permission umbrella held by
	// both roles. Read-only and lifecycle responsibilities are expressed by the
	// finer-grained PermissionUsersRead / PermissionUsersManage permissions.
	PermissionUsers       Permission = "users"
	PermissionUsersRead   Permission = "users.read"
	PermissionUsersManage Permission = "users.manage"
	// PermissionTrainers is the coarse trainer-management permission umbrella
	// held by both roles. Read-only and lifecycle responsibilities are
	// expressed by the finer-grained PermissionTrainersRead /
	// PermissionTrainersManage permissions.
	PermissionTrainers       Permission = "trainers"
	PermissionTrainersRead   Permission = "trainers.read"
	PermissionTrainersManage Permission = "trainers.manage"
	// PermissionTrainerApplications is the coarse trainer-application
	// management permission umbrella held by both roles. Reviewing trainer
	// applications is shared between the Technical and the Management
	// Administrator, so the read and manage sub-permissions are both granted
	// to every role.
	PermissionTrainerApplications       Permission = "trainer_applications"
	PermissionTrainerApplicationsRead   Permission = "trainer_applications.read"
	PermissionTrainerApplicationsManage Permission = "trainer_applications.manage"
	PermissionStatistics                Permission = "statistics"
	PermissionPlans                     Permission = "plans"
	PermissionFinance                   Permission = "finance"
	PermissionMarketing                 Permission = "marketing"
	PermissionSystem                    Permission = "system"
	PermissionInfrastructure            Permission = "infrastructure"
	PermissionTechnicalConfiguration    Permission = "technical-configuration"
	PermissionSecurity                  Permission = "security"
	PermissionDevelopment               Permission = "development"
)

// rolePermissions maps each role to its granted permissions. Users, trainers,
// trainer applications and statistics are shared by both roles. User management
// and trainer management are each split into a read-only view (users.read /
// trainers.read, shared by both roles) and a full lifecycle capability
// (users.manage / trainers.manage: create, update, soft-delete and reactivate,
// granted only to the Technical Administrator). Trainer-application review is
// shared by both roles: trainer_applications.read and
// trainer_applications.manage are both granted to every role. Technical
// concerns (system, infrastructure, technical configuration, security,
// development) belong exclusively to the Technical Administrator and business
// concerns (plans, finance, marketing) exclusively to the Management
// Administrator. Neither role is a superset of the other: the Technical
// Administrator does not automatically receive management permissions.
var rolePermissions = map[Role][]Permission{
	RoleTechnicalAdministrator: {
		PermissionUsers,
		PermissionUsersRead,
		PermissionUsersManage,
		PermissionTrainers,
		PermissionTrainersRead,
		PermissionTrainersManage,
		PermissionTrainerApplications,
		PermissionTrainerApplicationsRead,
		PermissionTrainerApplicationsManage,
		PermissionStatistics,
		PermissionSystem,
		PermissionInfrastructure,
		PermissionTechnicalConfiguration,
		PermissionSecurity,
		PermissionDevelopment,
	},
	RoleManagementAdministrator: {
		PermissionUsers,
		PermissionUsersRead,
		PermissionTrainers,
		PermissionTrainersRead,
		PermissionTrainerApplications,
		PermissionTrainerApplicationsRead,
		PermissionTrainerApplicationsManage,
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
