package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migrate_mysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"ryze/backend/config"
)

// Migrations are embedded so deployment artefacts need only the binary. A
// container image or systemd install that ships the binary without the source
// tree can still apply the full schema reproducibly.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// newMigrator creates a golang-migrate instance bound to the configured
// database and the embedded migrations. The returned close function must be
// called when the migrator is no longer needed.
func newMigrator(cfg config.DatabaseConfig) (*migrate.Migrate, func(), error) {
	sqlDB, err := sql.Open("mysql", DSN(cfg))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	close := func() { _ = sqlDB.Close() }

	driver, err := migrate_mysql.WithInstance(sqlDB, &migrate_mysql.Config{})
	if err != nil {
		close()
		return nil, nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		close()
		return nil, nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		_ = source.Close()
		close()
		return nil, nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return m, close, nil
}

// MigrateUp applies all pending migrations. It reports whether any migration
// was applied.
func MigrateUp(cfg config.DatabaseConfig) (bool, error) {
	m, close, err := newMigrator(cfg)
	if err != nil {
		return false, err
	}
	defer close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return false, nil
		}
		return false, fmt.Errorf("migration up failed: %w", err)
	}

	return true, nil
}

// MigrateDown rolls back all applied migrations. It reports whether any
// migration was rolled back.
func MigrateDown(cfg config.DatabaseConfig) (bool, error) {
	m, close, err := newMigrator(cfg)
	if err != nil {
		return false, err
	}
	defer close()

	if err := m.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return false, nil
		}
		return false, fmt.Errorf("migration down failed: %w", err)
	}

	return true, nil
}

// MigrateVersion prints the current migration version and dirty state.
func MigrateVersion(cfg config.DatabaseConfig) error {
	m, close, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer close()

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("no migrations applied")
			return nil
		}
		return fmt.Errorf("migration version check failed: %w", err)
	}

	fmt.Printf("current migration version: %d (dirty=%v)\n", version, dirty)
	return nil
}
