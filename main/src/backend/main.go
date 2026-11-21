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
	"ryze/backend/services/achievements"
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

	evidenceCfg, err := config.LoadEvidence()
	if err != nil {
		fail("configuration error: %v", err)
	}

	uploadsCfg, err := config.LoadUploads()
	if err != nil {
		fail("configuration error: %v", err)
	}

	contactCfg, err := config.LoadContact()
	if err != nil {
		fail("configuration error: %v", err)
	}

	mailCfg, err := config.LoadMail()
	if err != nil {
		fail("configuration error: %v", err)
	}
	googleCfg, err := config.LoadGoogle()
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

	if len(os.Args) > 1 && os.Args[1] == "set-xp" {
		runSetXPCommand(db, os.Args[2:])
		_ = sqlDB.Close()
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	invoiceCfg := config.LoadInvoice()
	router := routes.Setup(db, invoiceCfg, jwtCfg, corsCfg, adminCfg, pricingCfg, commissionCfg, stripeCfg, paypalCfg, webhookCfg, testModeCfg, serverCfg, rateLimitCfg, mediaCfg, evidenceCfg, uploadsCfg, contactCfg, mailCfg, googleCfg)

	// Background jobs (email delivery, retention) stop with the server.
	jobsCtx, stopJobs := context.WithCancel(context.Background())
	defer stopJobs()
	startBackgroundJobs(jobsCtx, db, mailCfg, invoiceCfg, testModeCfg.Enabled, jwtCfg.AccessTokenTTL)

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

	fmt.Printf("records=%d unique=%d created=%d matched_by_source=%d matched_by_name=%d updated=%d duplicate_source_ids_skipped=%d duplicate_names_skipped=%d name_conflicts=%d\n",
		report.Total, report.Unique, report.Created, report.MatchedBySource, report.MatchedByName, report.Updated,
		len(report.SkippedDuplicateSourceIDs), len(report.SkippedDuplicateNames), len(report.Conflicts))
	if len(report.SkippedDuplicateSourceIDs) > 0 {
		fmt.Printf("duplicate source ids skipped: %v\n", report.SkippedDuplicateSourceIDs)
	}
	if len(report.SkippedDuplicateNames) > 0 {
		fmt.Printf("duplicate names skipped: %v\n", report.SkippedDuplicateNames)
	}
	if len(report.Conflicts) > 0 {
		fmt.Printf("name conflicts left unchanged: %v\n", report.Conflicts)
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

// runSetXPCommand sets an account's total EXP by recording an EXP adjustment
// (owner request, 2026-11-21: put a test account one EXP short of the top
// level to see the level-up animation). --short-of-top N puts the account N
// EXP below the highest level; --xp N sets the total directly.
func runSetXPCommand(db *gorm.DB, args []string) {
	flags := flag.NewFlagSet("set-xp", flag.ContinueOnError)
	username := flags.String("username", "", "the account's @username")
	target := flags.Int("xp", -1, "the total EXP to reach")
	shortOfTop := flags.Int("short-of-top", 0, "EXP missing to reach the highest level")
	if err := flags.Parse(args); err != nil {
		fail("invalid set-xp arguments: %v", err)
	}
	if *shortOfTop > 0 {
		*target = achievements.Levels[len(achievements.Levels)-1].MinXP - *shortOfTop
	}
	if *username == "" || *target < 0 {
		fail("usage: go run . set-xp --username <username> (--xp <total> | --short-of-top <n>)")
	}
	ctx := context.Background()
	var userID string
	if err := db.WithContext(ctx).Raw(`SELECT id FROM users WHERE username = ? AND deleted_at IS NULL`, *username).Scan(&userID).Error; err != nil || userID == "" {
		var similar []string
		_ = db.WithContext(ctx).Raw(`SELECT username || ' (' || first_name || ' ' || last_name || ')' FROM users WHERE deleted_at IS NULL AND (username ILIKE ? OR first_name ILIKE ?) ORDER BY username LIMIT 10`,
			"%"+*username+"%", "%"+*username+"%").Scan(&similar).Error
		fail("no active account with the @username %q (similar: %v)", *username, similar)
	}
	repo := repositories.NewAchievementsRepository(db)
	summary, err := achievements.NewService(repo).Summary(ctx, userID)
	if err != nil {
		fail("failed to compute EXP: %v", err)
	}
	if delta := *target - summary.XP; delta != 0 {
		adder := repo.(interface {
			AddXPAdjustment(ctx context.Context, userID string, xp int, reason string) error
		})
		if err := adder.AddXPAdjustment(ctx, userID, delta, fmt.Sprintf("set-xp command: total set to %d", *target)); err != nil {
			fail("%v", err)
		}
	}
	level, next := achievements.LevelFor(*target)
	fmt.Printf("@%s: %d EXP -> %d EXP (level %d %s", *username, summary.XP, *target, level.Rank, level.Key)
	if next != nil {
		fmt.Printf(", %d EXP to %s", next.MinXP-*target, next.Key)
	}
	fmt.Println(")")
}
