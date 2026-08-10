package main

import (
	"fmt"
	"os"

	"ryze/backend/config"
	"ryze/backend/database"
)

func main() {
	config.LoadEnvFile()

	cfg, err := config.Load()
	if err != nil {
		fail("configuration error: %v", err)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrateCommand(cfg, os.Args[2:])
		return
	}

	db, err := database.Connect(cfg)
	if err != nil {
		fail("database connection failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		fail("database handle error: %v", err)
	}
	defer sqlDB.Close()

	fmt.Printf("database connection established successfully (host=%s port=%s db=%s)\n", cfg.Host, cfg.Port, cfg.Name)
}

func runMigrateCommand(cfg config.DatabaseConfig, args []string) {
	if len(args) < 1 {
		fail("usage: go run . migrate <up|down|version>")
	}

	switch args[0] {
	case "up":
		applied, err := database.MigrateUp(cfg)
		if err != nil {
			fail("migration up failed: %v", err)
		}
		if applied {
			fmt.Println("migrations applied successfully")
		} else {
			fmt.Println("no pending migrations")
		}
	case "down":
		applied, err := database.MigrateDown(cfg)
		if err != nil {
			fail("migration down failed: %v", err)
		}
		if applied {
			fmt.Println("migrations rolled back successfully")
		} else {
			fmt.Println("no migrations to roll back")
		}
	case "version":
		if err := database.MigrateVersion(cfg); err != nil {
			fail("migration version check failed: %v", err)
		}
	default:
		fail("unknown migrate command %q (expected up, down or version)", args[0])
	}
}

func fail(message string, args ...any) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
	os.Exit(1)
}
