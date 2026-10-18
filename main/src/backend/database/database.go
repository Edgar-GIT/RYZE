package database

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
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

// mysqlConfig builds the go-sql-driver connection configuration. The DSN is
// never exposed to keep credentials out of errors and logs. MultiStatements is
// enabled because the migration runner executes every migration file as a
// single multi-statement batch (golang-migrate requires the connection to have
// multiStatements=true); GORM itself never uses multi-statement batches.
func mysqlConfig(cfg config.DatabaseConfig) *gomysql.Config {
	return &gomysql.Config{
		User:                 cfg.User,
		Passwd:               cfg.Password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(cfg.Host, cfg.Port),
		DBName:               cfg.Name,
		ParseTime:            true,
		Loc:                  time.UTC,
		Timeout:              connectTimeout,
		ReadTimeout:          connectTimeout,
		WriteTimeout:         connectTimeout,
		AllowNativePasswords: true,
		MultiStatements:      true,
	}
}

// DSN returns the MySQL DSN for the given configuration.
func DSN(cfg config.DatabaseConfig) string {
	return mysqlConfig(cfg).FormatDSN()
}

// Connect establishes a MySQL/MariaDB connection from the given configuration.
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

	db, err := gorm.Open(mysql.Open(DSN(cfg)), &gorm.Config{Logger: dbLogger})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve database handle: %w", err)
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database %q at %s:%s is unreachable: %w", cfg.Name, cfg.Host, cfg.Port, err)
	}

	return db, nil
}
