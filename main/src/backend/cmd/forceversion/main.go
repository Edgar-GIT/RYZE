// Command forceversion is a temporary repair utility used to clear a dirty
// golang-migrate state after a failed migration run. It is not part of the
// backend and is deleted once the migration state is repaired.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migrate_mysql "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"ryze/backend/config"
	"ryze/backend/database"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: forceversion <version>")
		os.Exit(1)
	}
	target, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	sqlDB, err := sql.Open("mysql", database.DSN(cfg))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	driver, err := migrate_mysql.WithInstance(sqlDB, &migrate_mysql.Config{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	m, err := migrate.NewWithDatabaseInstance("file://database/migrations", "mysql", driver)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := m.Force(target); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	version, dirty, err := m.Version()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("forced to version: %d (dirty=%v)\n", version, dirty)
}
