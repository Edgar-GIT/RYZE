package email

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/models"
)

func TestRenderEveryTemplateInBothLocales(t *testing.T) {
	for template := range copies {
		for _, locale := range []string{LocaleEnglish, LocalePortuguese} {
			message, err := Render(template, locale, Data{Name: "Alex", Program: "Hypertrophy Foundations", Link: "https://ryze.example/x"})
			if err != nil {
				t.Fatalf("%s/%s: %v", template, locale, err)
			}
			if message.Subject == "" || !strings.Contains(message.Text, "Alex") || !strings.Contains(message.HTML, "RYZE") {
				t.Fatalf("%s/%s: incomplete message %+v", template, locale, message)
			}
		}
	}
}

func TestRenderEscapesHTMLAndRequiresLinks(t *testing.T) {
	message, err := Render(TemplateAccessGranted, LocaleEnglish, Data{Name: "<b>x</b>", Program: "P", Link: "https://ryze.example/a"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(message.HTML, "<b>x</b>") {
		t.Fatal("HTML part must escape user values")
	}
	if _, err := Render(TemplatePasswordReset, LocaleEnglish, Data{Name: "A"}); err == nil {
		t.Fatal("a link template without a link must fail")
	}
	if _, err := Render("nope", LocaleEnglish, Data{}); err == nil {
		t.Fatal("unknown template must fail")
	}
	if NormalizeLocale("PT") != LocalePortuguese || NormalizeLocale("de") != LocaleEnglish {
		t.Fatal("locale normalisation is wrong")
	}
}

func TestJoinAppLinkStaysOnTheApp(t *testing.T) {
	link, err := joinAppLink("https://ryze.example", "/reset-password#token=abc")
	if err != nil || link != "https://ryze.example/reset-password#token=abc" {
		t.Fatalf("unexpected link %q (%v)", link, err)
	}
	for _, path := range []string{"https://evil.example/x", "//evil.example/x", "reset"} {
		if _, err := joinAppLink("https://ryze.example", path); err == nil {
			t.Fatalf("path %q must be rejected", path)
		}
	}
	if plainText("a\r\nb\tc", 3) != "a b" {
		t.Fatalf("control characters must be removed, got %q", plainText("a\r\nb\tc", 3))
	}
}

type fakeOutbox struct {
	enqueued []*models.EmailOutbox
	claim    []models.EmailOutbox
	sent     []string
	retried  []string
	failed   []string
}

func (f *fakeOutbox) Enqueue(_ context.Context, email *models.EmailOutbox) (bool, error) {
	for _, existing := range f.enqueued {
		if existing.DedupeKey != nil && email.DedupeKey != nil && *existing.DedupeKey == *email.DedupeKey {
			return false, nil
		}
	}
	f.enqueued = append(f.enqueued, email)
	return true, nil
}
func (f *fakeOutbox) ClaimDue(context.Context, time.Time, int) ([]models.EmailOutbox, error) {
	claimed := f.claim
	f.claim = nil
	return claimed, nil
}
func (f *fakeOutbox) MarkSent(_ context.Context, id string, _ time.Time) error {
	f.sent = append(f.sent, id)
	return nil
}
func (f *fakeOutbox) MarkRetry(_ context.Context, id string, _ time.Time, _ string) error {
	f.retried = append(f.retried, id)
	return nil
}
func (f *fakeOutbox) MarkFailed(_ context.Context, id string, _ string) error {
	f.failed = append(f.failed, id)
	return nil
}
func (f *fakeOutbox) ExpireOverdue(context.Context, time.Time) (int64, error) { return 0, nil }
func (f *fakeOutbox) RecoverStale(context.Context, time.Time) (int64, error)  { return 0, nil }

type fakeSender struct {
	err  error
	sent []Message
}

func (f *fakeSender) Send(_ context.Context, message Message) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, message)
	return nil
}

func enabledConfig() config.MailConfig {
	return config.MailConfig{SMTPHost: "smtp.example", SMTPPort: "587", From: "RYZE <no-reply@ryze.example>", PublicAppURL: "https://ryze.example"}
}

func TestMailerDeduplicatesAndSkipsWhenDisabled(t *testing.T) {
	outbox := &fakeOutbox{}
	mailer := NewMailer(enabledConfig(), outbox)
	request := Request{UserID: "u1", To: "a@ryze.example", Template: TemplateAccessGranted, Locale: "pt", Name: "A", Program: "P", Path: "/services/my-programs", DedupeKey: "access:1"}
	for i := 0; i < 2; i++ {
		if err := mailer.Enqueue(context.Background(), request); err != nil {
			t.Fatal(err)
		}
	}
	if len(outbox.enqueued) != 1 || outbox.enqueued[0].Status != models.EmailStatusPending || outbox.enqueued[0].Locale != "pt" {
		t.Fatalf("expected one pending Portuguese email, got %+v", outbox.enqueued)
	}
	var data Data
	_ = json.Unmarshal(outbox.enqueued[0].Payload, &data)
	if data.Link != "https://ryze.example/services/my-programs" {
		t.Fatalf("unexpected link %q", data.Link)
	}

	disabled := &fakeOutbox{}
	if err := NewMailer(config.MailConfig{}, disabled).Enqueue(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(disabled.enqueued) != 1 || disabled.enqueued[0].Status != models.EmailStatusSkipped || string(disabled.enqueued[0].Payload) != "{}" {
		t.Fatalf("disabled delivery must record a skipped email without payload, got %+v", disabled.enqueued)
	}
}

func claimed(id string, attempts int) models.EmailOutbox {
	payload, _ := json.Marshal(Data{Name: "A", Program: "P", Link: "https://ryze.example/x"})
	return models.EmailOutbox{ID: id, Recipient: "a@ryze.example", Template: TemplateAccessGranted, Locale: "en", Payload: payload, Attempts: attempts}
}

func TestWorkerSendsRetriesAndFails(t *testing.T) {
	outbox := &fakeOutbox{claim: []models.EmailOutbox{claimed("ok", 1)}}
	sender := &fakeSender{}
	NewWorker(outbox, sender).RunOnce(context.Background())
	if len(outbox.sent) != 1 || len(sender.sent) != 1 || sender.sent[0].To != "a@ryze.example" {
		t.Fatalf("expected one delivery, got sent=%v", outbox.sent)
	}

	outbox = &fakeOutbox{claim: []models.EmailOutbox{claimed("temp", 1)}}
	NewWorker(outbox, &fakeSender{err: errors.New("smtp relay unreachable")}).RunOnce(context.Background())
	if len(outbox.retried) != 1 || len(outbox.failed) != 0 {
		t.Fatalf("a temporary failure must be retried, got retried=%v failed=%v", outbox.retried, outbox.failed)
	}

	outbox = &fakeOutbox{claim: []models.EmailOutbox{claimed("perm", 1)}}
	NewWorker(outbox, &fakeSender{err: ErrPermanent}).RunOnce(context.Background())
	if len(outbox.failed) != 1 || len(outbox.retried) != 0 {
		t.Fatalf("a permanent failure must not be retried, got failed=%v retried=%v", outbox.failed, outbox.retried)
	}

	outbox = &fakeOutbox{claim: []models.EmailOutbox{claimed("last", maxAttempts)}}
	NewWorker(outbox, &fakeSender{err: errors.New("smtp relay unreachable")}).RunOnce(context.Background())
	if len(outbox.failed) != 1 || len(outbox.retried) != 0 {
		t.Fatalf("an email out of attempts must fail, got failed=%v retried=%v", outbox.failed, outbox.retried)
	}
}
