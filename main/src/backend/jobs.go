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
	"ryze/backend/services/notifications"
)

// notificationPurgeInterval is how often expired notifications are cleared.
const notificationPurgeInterval = time.Hour

// startBackgroundJobs launches the in-process workers. Each worker claims its
// rows with row locks, so running several API instances is safe.
func startBackgroundJobs(ctx context.Context, db *gorm.DB, mailCfg config.MailConfig) {
	if mailCfg.Enabled() {
		worker := email.NewWorker(repositories.NewEmailOutboxRepository(db), email.NewSMTPSender(mailCfg))
		go worker.Run(ctx)
		fmt.Println("transactional email delivery enabled")
	} else {
		fmt.Println("transactional email delivery disabled (MAIL_SMTP_HOST, MAIL_FROM and PUBLIC_APP_URL not set); emails are recorded as skipped")
	}
	go runNotificationPurge(ctx, notifications.New(repositories.NewNotificationRepository(db), nil, nil, nil))
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
