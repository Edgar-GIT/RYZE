package email

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"ryze/backend/models"
	"ryze/backend/repositories"
)

const (
	workerInterval = 15 * time.Second
	workerBatch    = 20
	maxAttempts    = 6
	staleSending   = 10 * time.Minute
)

// retryDelay is the wait before attempt n+1 (1, 5, 15, 60 and 240 minutes).
func retryDelay(attempts int) time.Duration {
	delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 4 * time.Hour}
	if attempts-1 < len(delays) && attempts >= 1 {
		return delays[attempts-1]
	}
	return 4 * time.Hour
}

// Worker delivers due outbox emails.
type Worker struct {
	outbox repositories.EmailOutboxRepository
	sender Sender
	now    func() time.Time
}

func NewWorker(outbox repositories.EmailOutboxRepository, sender Sender) *Worker {
	return &Worker{outbox: outbox, sender: sender, now: func() time.Time { return time.Now().UTC() }}
}

// Run delivers emails until the context is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(workerInterval)
	defer ticker.Stop()
	for {
		w.RunOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunOnce performs one delivery pass. Errors are logged without recipients or
// content.
func (w *Worker) RunOnce(ctx context.Context) {
	now := w.now()
	if _, err := w.outbox.RecoverStale(ctx, now.Add(-staleSending)); err != nil {
		slog.Error("email outbox recovery failed", slog.String("error", err.Error()))
	}
	if _, err := w.outbox.ExpireOverdue(ctx, now); err != nil {
		slog.Error("email outbox expiry failed", slog.String("error", err.Error()))
	}
	due, err := w.outbox.ClaimDue(ctx, now, workerBatch)
	if err != nil {
		slog.Error("email outbox claim failed", slog.String("error", err.Error()))
		return
	}
	for _, record := range due {
		w.deliver(ctx, record)
	}
}

func (w *Worker) deliver(ctx context.Context, record models.EmailOutbox) {
	var data Data
	if err := json.Unmarshal(record.Payload, &data); err != nil {
		w.fail(ctx, record, "invalid payload")
		return
	}
	message, err := Render(record.Template, record.Locale, data)
	if err != nil {
		w.fail(ctx, record, "render failed")
		return
	}
	message.To = record.Recipient

	sendCtx, cancel := context.WithTimeout(ctx, smtpTimeout+5*time.Second)
	err = w.sender.Send(sendCtx, message)
	cancel()
	if err == nil {
		if markErr := w.outbox.MarkSent(ctx, record.ID, w.now()); markErr != nil {
			slog.Error("email sent but not recorded", slog.String("email_id", record.ID), slog.String("error", markErr.Error()))
		}
		return
	}

	if errors.Is(err, ErrPermanent) || record.Attempts >= maxAttempts {
		w.fail(ctx, record, err.Error())
		return
	}
	if markErr := w.outbox.MarkRetry(ctx, record.ID, w.now().Add(retryDelay(record.Attempts)), err.Error()); markErr != nil {
		slog.Error("email retry not recorded", slog.String("email_id", record.ID), slog.String("error", markErr.Error()))
	}
}

func (w *Worker) fail(ctx context.Context, record models.EmailOutbox, reason string) {
	slog.Warn("email delivery failed permanently",
		slog.String("email_id", record.ID),
		slog.String("template", record.Template),
		slog.String("reason", reason),
	)
	if err := w.outbox.MarkFailed(ctx, record.ID, reason); err != nil {
		slog.Error("email failure not recorded", slog.String("email_id", record.ID), slog.String("error", err.Error()))
	}
}
