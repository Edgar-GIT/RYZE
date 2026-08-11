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
