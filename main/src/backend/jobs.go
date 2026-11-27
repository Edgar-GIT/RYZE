package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/email"
	"ryze/backend/repositories"
	"ryze/backend/services/account_anonymization"
	"ryze/backend/services/invoices"
	"ryze/backend/services/notifications"
	"ryze/backend/services/premium_level_2"
	uploads "ryze/backend/services/public_uploads"
	"ryze/backend/services/test_mode"
	"ryze/backend/services/training_reminders"
)

// notificationPurgeInterval is how often expired notifications are cleared.
const notificationPurgeInterval = time.Hour

// invoiceSweepInterval is how often missing purchase documents are issued and
// unrecorded document emails are recorded.
const invoiceSweepInterval = 5 * time.Minute

// startBackgroundJobs launches the in-process workers. Each worker claims its
// rows with row locks, so running several API instances is safe.
func startBackgroundJobs(ctx context.Context, db *gorm.DB, mailCfg config.MailConfig, invoiceCfg config.InvoiceConfig, uploadsCfg config.UploadsConfig, testModeEnabled bool, testSessionTTL time.Duration) {
	if mailCfg.Enabled() {
		worker := email.NewWorker(repositories.NewEmailOutboxRepository(db), email.NewSMTPSender(mailCfg))
		go worker.Run(ctx)
		fmt.Println("transactional email delivery enabled")
	} else {
		fmt.Println("transactional email delivery disabled (MAIL_SMTP_HOST, MAIL_FROM and PUBLIC_APP_URL not set); emails are recorded as skipped")
	}
	go runNotificationPurge(ctx, notifications.New(repositories.NewNotificationRepository(db), nil, nil, nil))
	go runAccountAnonymization(ctx, account_anonymization.New(repositories.NewAccountAnonymizationRepository(db),
		uploads.NewStore(uploadsCfg.Dir, uploadsCfg.MaxImageBytes)))
	mailer := email.NewMailer(mailCfg, repositories.NewEmailOutboxRepository(db))
	recipients := test_mode.NewRecipientResolver(testModeEnabled, repositories.NewTestSessionRepository(db), testSessionTTL)
	go runInvoiceSweep(ctx, invoices.New(repositories.NewInvoiceRepository(db), mailer, recipients, invoiceCfg))
	notifier := notifications.New(repositories.NewNotificationRepository(db), repositories.NewUserRepository(db),
		repositories.NewProgramRepository(db), mailer, notifications.WithTestRecipients(recipients))
	go runTrainingReminders(ctx, training_reminders.New(repositories.NewTrainingReminderRepository(db), notifier))
	go runPremiumLevel2OfferSweep(ctx, premium_level_2.NewService(repositories.NewPremiumLevel2Repository(db),
		repositories.NewProgramRepository(db), repositories.NewEntitlementRepository(db),
		premium_level_2.WithNotifier(notifier), premium_level_2.WithOffers(repositories.NewPremiumLevel2OfferRepository(db))))
}

// premiumLevel2OfferSweepInterval is how often Premium Level 2 offers past
// their 24 hours expire and waiting requests are offered to the next trainer.
const premiumLevel2OfferSweepInterval = 5 * time.Minute

func runPremiumLevel2OfferSweep(ctx context.Context, service premium_level_2.Service) {
	offers, ok := service.(premium_level_2.OfferService)
	if !ok {
		return
	}
	ticker := time.NewTicker(premiumLevel2OfferSweepInterval)
	defer ticker.Stop()
	for {
		if err := offers.SweepOffers(ctx); err != nil && ctx.Err() == nil {
			slog.Error("premium level 2 offer sweep failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// runInvoiceSweep issues the documents still missing for completed purchases
// (an issue that failed in the purchase request) and records document emails
// that were never recorded, at start and then every few minutes. Issuing is
// idempotent per purchase, so several API instances may run it at once.
func runInvoiceSweep(ctx context.Context, service invoices.Service) {
	ticker := time.NewTicker(invoiceSweepInterval)
	defer ticker.Stop()
	for {
		if err := service.Sweep(ctx); err != nil && ctx.Err() == nil {
			slog.Error("invoice sweep failed", slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// runNotificationPurge applies the notification retention policy at start and
// then every hour. The statements are idempotent, so several API instances
// may run it at once.
func runNotificationPurge(ctx context.Context, service notifications.Service) {
	ticker := time.NewTicker(notificationPurgeInterval)
	defer ticker.Stop()
	for {
		if err := service.Purge(ctx); err != nil && ctx.Err() == nil {
			slog.Error("notification purge failed", slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// accountAnonymizationInterval is how often accounts deleted more than 30
// days ago are anonymised.
const accountAnonymizationInterval = time.Hour

// runAccountAnonymization anonymises, at start and then every hour, the
// accounts whose 30-day restore period ended (GDPR art. 17, v2.131.0). Each
// account is locked in its own transaction, so several API instances may run
// it at once.
func runAccountAnonymization(ctx context.Context, service *account_anonymization.Service) {
	ticker := time.NewTicker(accountAnonymizationInterval)
	defer ticker.Stop()
	for {
		if count, err := service.Sweep(ctx); err != nil && ctx.Err() == nil {
			slog.Error("account anonymisation failed", slog.String("error", err.Error()))
		} else if count > 0 {
			slog.Info("accounts anonymised", slog.Int("count", count))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// trainingReminderInterval is how often members who stopped training are
// looked for; each is reminded at most once a week, during the day.
const trainingReminderInterval = time.Hour

// runTrainingReminders sends the training reminders (v2.131.0). The reminder
// time is recorded per member, so several API instances may run it at once
// (the notification dedupe key keeps one reminder per member and day).
func runTrainingReminders(ctx context.Context, service *training_reminders.Service) {
	ticker := time.NewTicker(trainingReminderInterval)
	defer ticker.Stop()
	for {
		if count, err := service.Sweep(ctx); err != nil && ctx.Err() == nil {
			slog.Error("training reminders failed", slog.String("error", err.Error()))
		} else if count > 0 {
			slog.Info("training reminders sent", slog.Int("count", count))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
