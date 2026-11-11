package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ryze/backend/config"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 25
	connMaxIdleTime = 5 * time.Minute
	connMaxLifetime = 2 * time.Hour
	connectTimeout  = 5 * time.Second

	// slowQueryThreshold is how long a statement may run before it is logged
	// as slow, matching the GORM default.
	slowQueryThreshold = 200 * time.Millisecond
)

// quoteDSNValue quotes a value for a libpq keyword/value connection string,
// so passwords and names containing spaces or quotes stay intact.
func quoteDSNValue(value string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value)
	return "'" + escaped + "'"
}

// connConfig builds the pgx connection configuration. The DSN is never exposed
// to keep credentials out of errors and logs. Sessions run in UTC.
func connConfig(cfg config.DatabaseConfig) (*pgx.ConnConfig, error) {
	parts := []string{
		"host=" + quoteDSNValue(cfg.Host),
		"port=" + quoteDSNValue(cfg.Port),
		"dbname=" + quoteDSNValue(cfg.Name),
		"user=" + quoteDSNValue(cfg.User),
		"password=" + quoteDSNValue(cfg.Password),
		"sslmode=" + quoteDSNValue(cfg.SSLMode),
		fmt.Sprintf("connect_timeout=%d", int(connectTimeout.Seconds())),
		"timezone=UTC",
	}
	parsed, err := pgx.ParseConfig(strings.Join(parts, " "))
	if err != nil {
		// pgx errors can echo the connection string; never wrap them.
		return nil, fmt.Errorf("invalid database configuration for %q at %s:%s", cfg.Name, cfg.Host, cfg.Port)
	}
	return parsed, nil
}

// scanTimestamptzInUTC makes every scanned TIMESTAMPTZ value a UTC time.Time,
// as the previous MariaDB driver did. The instant is unchanged; only the
// location of the returned value is fixed, so JSON output keeps its "Z" form
// whatever the host time zone is.
func scanTimestamptzInUTC(_ context.Context, conn *pgx.Conn) error {
	conn.TypeMap().RegisterType(&pgtype.Type{
		Name:  "timestamptz",
		OID:   pgtype.TimestamptzOID,
		Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC},
	})
	return nil
}

// openSQL opens a database/sql handle backed by pgx for the configuration.
func openSQL(cfg config.DatabaseConfig) (*sql.DB, error) {
	parsed, err := connConfig(cfg)
	if err != nil {
		return nil, err
	}
	return stdlib.OpenDB(*parsed, stdlib.OptionAfterConnect(scanTimestamptzInUTC)), nil
}

// Connect establishes a PostgreSQL connection from the given configuration.
// The connection is verified with a ping so an unavailable database fails
// immediately with a meaningful error.
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	// ParameterizedQueries keeps the GORM logger from interpolating values into
	// the SQL it emits: error and slow-query logs stay structural, so request
	// data can never leak into log files through a logged statement.
	dbLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:        slowQueryThreshold,
			LogLevel:             logger.Warn,
			ParameterizedQueries: true,
		},
	)

	sqlDB, err := openSQL(cfg)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{Logger: dbLogger})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database %q at %s:%s is unreachable: %w", cfg.Name, cfg.Host, cfg.Port, err)
	}

	return db, nil
}
