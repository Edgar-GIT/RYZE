package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = "3306"
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
