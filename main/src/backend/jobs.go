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
	"ryze/backend/services/invoices"
	"ryze/backend/services/notifications"
	"ryze/backend/services/premium_level_2"
	"ryze/backend/services/test_mode"
)

// notificationPurgeInterval is how often expired notifications are cleared.
const notificationPurgeInterval = time.Hour

// invoiceSweepInterval is how often missing purchase documents are issued and
// unrecorded document emails are recorded.
const invoiceSweepInterval = 5 * time.Minute

// startBackgroundJobs launches the in-process workers. Each worker claims its
// rows with row locks, so running several API instances is safe.
func startBackgroundJobs(ctx context.Context, db *gorm.DB, mailCfg config.MailConfig, invoiceCfg config.InvoiceConfig, testModeEnabled bool, testSessionTTL time.Duration) {
	if mailCfg.Enabled() {
		worker := email.NewWorker(repositories.NewEmailOutboxRepository(db), email.NewSMTPSender(mailCfg))
		go worker.Run(ctx)
		fmt.Println("transactional email delivery enabled")
	} else {
		fmt.Println("transactional email delivery disabled (MAIL_SMTP_HOST, MAIL_FROM and PUBLIC_APP_URL not set); emails are recorded as skipped")
	}
	go runNotificationPurge(ctx, notifications.New(repositories.NewNotificationRepository(db), nil, nil, nil))
	mailer := email.NewMailer(mailCfg, repositories.NewEmailOutboxRepository(db))
	recipients := test_mode.NewRecipientResolver(testModeEnabled, repositories.NewTestSessionRepository(db), testSessionTTL)
	go runInvoiceSweep(ctx, invoices.New(repositories.NewInvoiceRepository(db), mailer, recipients, invoiceCfg))
	notifier := notifications.New(repositories.NewNotificationRepository(db), repositories.NewUserRepository(db),
		repositories.NewProgramRepository(db), mailer, notifications.WithTestRecipients(recipients))
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
