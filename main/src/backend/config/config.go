package config

import "time"

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

type JWTConfig struct {
	Secret         string
	AccessTokenTTL time.Duration
	CookieSecure   bool
}

type CORSConfig struct {
	AllowedOrigins []string
}

// Admin1ID and Admin2ID are the identifiers of the configured platform
// administrators. They are the only identities accepted by the admin
// authentication middleware, so the token subject is never trusted as an admin
// identity without passing this check.
const (
	Admin1ID = "ADMIN_1"
	Admin2ID = "ADMIN_2"
)

// IsValidAdminIdentity reports whether id is one of the configured
// administrator identities.
func IsValidAdminIdentity(id string) bool {
	return id == Admin1ID || id == Admin2ID
}

// Admin identifies one configured administrator. Administrators are platform
// accounts defined exclusively through configuration; they are never stored in
// the database.
type Admin struct {
	ID         string
	Username   string
	Password   string
	AccessCode string
}

type AdminConfig struct {
	Admins []Admin
}
