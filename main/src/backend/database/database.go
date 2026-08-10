package database

import (
	"fmt"
	"net"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"ryze/backend/config"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 25
	connMaxIdleTime = 5 * time.Minute
	connMaxLifetime = 2 * time.Hour
	connectTimeout  = 5 * time.Second
)

// Connect establishes a MySQL/MariaDB connection from the given configuration.
// The connection is verified with a ping so an unavailable database fails
// immediately with a meaningful error.
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	mysqlConfig := gomysql.Config{
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
	}

	db, err := gorm.Open(mysql.Open(mysqlConfig.FormatDSN()), &gorm.Config{})
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
