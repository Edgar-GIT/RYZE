package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultHost          = "127.0.0.1"
	defaultPort          = "3306"
	defaultTokenTTL      = 15 * time.Minute
	minJWTSecretLength   = 32
	defaultAllowedOrigin = "http://localhost:5173,http://127.0.0.1:5173"
	minAdminPasswordLen  = 6
)

// LoadEnvFile loads the repository .env file so environment variables defined
// there are available to the process. It searches the working directory first
// and then walks upwards until the repository root (identified by .git) is found.
func LoadEnvFile() {
	_ = godotenv.Load()

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			_ = godotenv.Load(filepath.Join(dir, ".env"))
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

// Load reads the database configuration from environment variables.
// DB_HOST and DB_PORT have defaults; DB_NAME and DB_USER are required.
func Load() (DatabaseConfig, error) {
	cfg := DatabaseConfig{
		Host:     valueOr("DB_HOST", defaultHost),
		Port:     valueOr("DB_PORT", defaultPort),
		Name:     os.Getenv("DB_NAME"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
	}

	if cfg.Name == "" {
		return DatabaseConfig{}, fmt.Errorf("DB_NAME is required")
	}
	if cfg.User == "" {
		return DatabaseConfig{}, fmt.Errorf("DB_USER is required")
	}
	if port, err := strconv.Atoi(cfg.Port); err != nil || port < 1 || port > 65535 {
		return DatabaseConfig{}, fmt.Errorf("DB_PORT must be a valid port number, got %q", cfg.Port)
	}

	return cfg, nil
}

func valueOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// LoadJWT reads the JWT configuration from environment variables. The signing
// secret is required and must be strong enough (minimum length) so the server
// never silently falls back to an insecure default. The access-token lifetime
// is optional and defaults to 15 minutes when not provided. The auth cookie is
// Secure by default; local HTTP development may disable it with COOKIE_SECURE.
func LoadJWT() (JWTConfig, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < minJWTSecretLength {
		return JWTConfig{}, fmt.Errorf("JWT_SECRET must be set and at least %d characters long", minJWTSecretLength)
	}

	ttl := defaultTokenTTL
	if raw := strings.TrimSpace(os.Getenv("JWT_ACCESS_TOKEN_TTL")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return JWTConfig{}, fmt.Errorf("JWT_ACCESS_TOKEN_TTL must be a positive duration, got %q", raw)
		}
		ttl = parsed
	}

	cookieSecure := true
	if raw := strings.TrimSpace(os.Getenv("COOKIE_SECURE")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return JWTConfig{}, fmt.Errorf("COOKIE_SECURE must be a boolean, got %q", raw)
		}
		cookieSecure = parsed
	}

	return JWTConfig{Secret: secret, AccessTokenTTL: ttl, CookieSecure: cookieSecure}, nil
}

// LoadAdmin reads the configured administrators from environment variables.
// Every admin requires a username and a password of at least
// minAdminPasswordLen characters; missing or invalid values fail at startup
// instead of silently disabling an administrator. Usernames must be unique so
// credentials always resolve to a single admin identity.
func LoadAdmin() (AdminConfig, error) {
	admins := []Admin{
		{ID: "ADMIN_1"},
		{ID: "ADMIN_2"},
	}

	seenUsernames := map[string]string{}
	for i, admin := range admins {
		username := strings.TrimSpace(os.Getenv(admin.ID + "_USERNAME"))
		password := os.Getenv(admin.ID + "_PASSWORD")
		if username == "" {
			return AdminConfig{}, fmt.Errorf("%s_USERNAME is required", admin.ID)
		}
		if len(password) < minAdminPasswordLen {
			return AdminConfig{}, fmt.Errorf("%s_PASSWORD is required and must be at least %d characters long", admin.ID, minAdminPasswordLen)
		}
		if previous, exists := seenUsernames[username]; exists {
			return AdminConfig{}, fmt.Errorf("admin username %q is already used by %s and %s", username, previous, admin.ID)
		}
		seenUsernames[username] = admin.ID
		admins[i].Username = username
		admins[i].Password = password
	}

	return AdminConfig{Admins: admins}, nil
}

// LoadCORS reads the cross-origin configuration from environment variables.
// CORS_ALLOWED_ORIGINS is a comma-separated list of origins allowed to call the
// API with credentials. When unset it defaults to the local Vite development
// origins so the frontend can talk to the backend without extra setup.
func LoadCORS() (CORSConfig, error) {
	origins := []string{}
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		raw = defaultAllowedOrigin
	}

	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			return CORSConfig{}, fmt.Errorf("CORS_ALLOWED_ORIGINS contains an empty origin")
		}
		origins = append(origins, origin)
	}

	return CORSConfig{AllowedOrigins: origins}, nil
}
