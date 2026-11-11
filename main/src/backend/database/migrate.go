package database

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migrate_pgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
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
// Each migration file is executed as one simple-protocol batch, which
// PostgreSQL runs as a single implicit transaction: a failing file leaves no
// partial schema behind.
func newMigrator(cfg config.DatabaseConfig) (*migrate.Migrate, func(), error) {
	sqlDB, err := openSQL(cfg)
	if err != nil {
		return nil, nil, err
	}

	close := func() { _ = sqlDB.Close() }

	driver, err := migrate_pgx.WithInstance(sqlDB, &migrate_pgx.Config{})
	if err != nil {
		close()
		return nil, nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		close()
		return nil, nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "pgx5", driver)
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
