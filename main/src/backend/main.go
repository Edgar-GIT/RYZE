package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/repositories"
	"ryze/backend/routes"
	"ryze/backend/services/exercise_import"
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
	// Refusing to start would break local HTTP development, so the insecure
	// cookie mode is loud instead: auth cookies must never travel over plain
	// HTTP outside a developer machine.
	if !jwtCfg.CookieSecure {
		fmt.Fprintln(os.Stderr, "warning: COOKIE_SECURE=false - authentication cookies are sent without the Secure attribute; never use this outside local development")
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

	rateLimitCfg, err := config.LoadRateLimit()
	if err != nil {
		fail("configuration error: %v", err)
	}

	mediaCfg, err := config.LoadExerciseMedia()
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

	if len(os.Args) > 1 && os.Args[1] == "import-exercises" {
		runImportExercisesCommand(db, os.Args[2:])
		_ = sqlDB.Close()
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := routes.Setup(db, jwtCfg, corsCfg, adminCfg, pricingCfg, commissionCfg, stripeCfg, paypalCfg, webhookCfg, testModeCfg, serverCfg, rateLimitCfg, mediaCfg)

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

// runImportExercisesCommand imports an external exercise dataset into the
// global catalogue. The whole import runs inside one transaction, so a failure
// never leaves a partially imported catalogue behind. Dry runs execute the
// same validation and matching work and then roll everything back.
func runImportExercisesCommand(db *gorm.DB, args []string) {
	flags := flag.NewFlagSet("import-exercises", flag.ContinueOnError)
	filePath := flags.String("file", "", "path to the exercises dataset JSON file")
	dryRun := flags.Bool("dry-run", false, "validate and report without committing")
	if err := flags.Parse(args); err != nil {
		fail("invalid import-exercises arguments: %v", err)
	}
	if *filePath == "" {
		fail("usage: go run . import-exercises --file <path> [--dry-run]")
	}

	file, err := os.Open(*filePath)
	if err != nil {
		fail("failed to open dataset: %v", err)
	}
	defer file.Close()

	tx := db.Begin()
	if tx.Error != nil {
		fail("failed to start import transaction: %v", tx.Error)
	}

	importer := exercise_import.NewImporter(repositories.NewExerciseRepository(tx))
	report, importErr := importer.Import(context.Background(), file)
	if importErr != nil {
		_ = tx.Rollback()
		fail("exercise import failed: %v", importErr)
	}

	if *dryRun {
		if err := tx.Rollback().Error; err != nil {
			fail("failed to roll back dry run: %v", err)
		}
	} else if err := tx.Commit().Error; err != nil {
		fail("failed to commit exercise import: %v", err)
	}

	fmt.Printf("records=%d unique=%d created=%d matched_by_source=%d matched_by_name=%d updated=%d duplicates_skipped=%d conflicts=%d\n",
		report.Total, report.Unique, report.Created, report.MatchedBySource, report.MatchedByName, report.Updated,
		len(report.SkippedDuplicates), len(report.Conflicts))
	if len(report.SkippedDuplicates) > 0 {
		fmt.Printf("duplicate source ids skipped: %v\n", report.SkippedDuplicates)
	}
	if len(report.Conflicts) > 0 {
		fmt.Printf("conflicting source ids left unchanged: %v\n", report.Conflicts)
	}
	if *dryRun {
		fmt.Println("dry run: no changes were committed")
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
