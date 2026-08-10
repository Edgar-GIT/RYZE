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

func fail(message string, args ...any) {
	fmt.Fprintf(os.Stderr, message+"\n", args...)
	os.Exit(1)
}
