package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/routes"
)

func main() {
	config.LoadEnvFile()

	cfg, err := config.Load()
	if err != nil {
		fail("configuration error: %v", err)
	}

	jwtCfg, err := config.LoadJWT()
	if err != nil {
		fail("configuration error: %v", err)
	}

	corsCfg, err := config.LoadCORS()
	if err != nil {
		fail("configuration error: %v", err)
	}

	adminCfg, err := config.LoadAdmin()
	if err != nil {
		fail("configuration error: %v", err)
	}

	pricingCfg, err := config.LoadPricing()
	if err != nil {
		fail("configuration error: %v", err)
	}

	commissionCfg, err := config.LoadCommission()
	if err != nil {
		fail("configuration error: %v", err)
	}

	stripeCfg := config.LoadStripe()

	paypalCfg := config.LoadPayPal()

	webhookCfg := config.LoadWebhooks()

	testModeCfg, err := config.LoadTestMode()
	if err != nil {
		fail("configuration error: %v", err)
	}

	serverCfg, err := config.LoadServer()
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

	fmt.Printf("database connection established successfully (host=%s port=%s db=%s)\n", cfg.Host, cfg.Port, cfg.Name)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := routes.Setup(db, jwtCfg, corsCfg, adminCfg, pricingCfg, commissionCfg, stripeCfg, paypalCfg, webhookCfg, testModeCfg, serverCfg)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: serverCfg.ReadHeaderTimeout,
		ReadTimeout:       serverCfg.ReadTimeout,
		WriteTimeout:      serverCfg.WriteTimeout,
		IdleTimeout:       serverCfg.IdleTimeout,
	}

	serveUntilShutdown(server, sqlDB, serverCfg.ShutdownTimeout)
}

// serveUntilShutdown runs the HTTP server until it fails or a termination
// signal (SIGINT/SIGTERM) arrives. On a signal it stops accepting new
// connections, drains the in-flight requests within shutdownTimeout, and closes
// the database handle, so a deploy or restart never drops an in-flight request.
func serveUntilShutdown(server *http.Server, sqlDB *sql.DB, shutdownTimeout time.Duration) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		_ = sqlDB.Close()
		fail("http server error: %v", err)
	case <-ctx.Done():
		fmt.Println("shutdown signal received, draining in-flight requests")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "graceful shutdown did not complete: %v\n", err)
		_ = server.Close()
	}

	if err := sqlDB.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "database close error: %v\n", err)
	}
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
