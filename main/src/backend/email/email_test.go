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
			data := Data{Name: "Alex", Program: "Hypertrophy Foundations", Link: "https://ryze.example/x"}
			if template == TemplateInvoiceIssued {
				data.Invoice = &InvoiceData{Number: "RYZE 2026/000001", IssuedAt: "2026-11-20", SellerName: "RYZE", BuyerName: "Alex Costa", Description: "Hypertrophy Foundations", Quantity: 1, UnitPrice: 2500, Total: 2500, Currency: "EUR", TaxExemption: "M10"}
			}
			message, err := Render(template, locale, data)
			if err != nil {
				t.Fatalf("%s/%s: %v", template, locale, err)
			}
			if message.Subject == "" || !strings.Contains(message.Text, "Alex") || !strings.Contains(message.HTML, "RYZE") {
				t.Fatalf("%s/%s: incomplete message %+v", template, locale, message)
			}
		}
	}
}

func TestRecipientAddressValidation(t *testing.T) {
	for _, valid := range []string{"qa@example.com", "  qa.tester@example.com  ", "first.last+tag@sub.example.com"} {
		if !IsValidAddress(valid) {
			t.Fatalf("expected %q to be valid", valid)
		}
	}
	for _, invalid := range []string{"", "qa", "Name <qa@example.com>", "qa@example.com\r\nBcc: x@example.com", strings.Repeat("a", 250) + "@example.com"} {
		if IsValidAddress(invalid) {
			t.Fatalf("expected %q to be rejected", invalid)
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

func TestRedactAddressesRemovesRecipientsFromFailureReasons(t *testing.T) {
	got := redactAddresses("550 5.1.1 <qa.tester@example.com>: Recipient address rejected")
	if strings.Contains(got, "qa.tester") || strings.Contains(got, "example.com") {
		t.Fatalf("address not redacted: %q", got)
	}
	if !strings.Contains(got, "550 5.1.1") || !strings.Contains(got, "[address]") {
		t.Fatalf("reason lost its context: %q", got)
	}
}

func TestInvoiceEmailCarriesTheDocument(t *testing.T) {
	data := Data{Name: "Rita", Program: "Plan <b>", Link: "https://ryze.example/account/purchases/p1/receipt", Invoice: &InvoiceData{
		Number: "TEST 2026/000007", IssuedAt: "2026-11-20", Test: true, SellerName: "RYZE", BuyerName: "Rita <Costa>", BuyerTaxID: "123456789",
		Description: "Plan <b>", Quantity: 1, UnitPrice: 2500, Total: 2500, Currency: "EUR", TaxExemption: "M10",
	}}
	pt, err := Render(TemplateInvoiceIssued, LocalePortuguese, data)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"TEST 2026/000007", "25,00 €", "123456789", "artigo 53.º do CIVA", "não tem valor fiscal", "Modo de Teste", "Rita &lt;Costa&gt;"} {
		if !strings.Contains(pt.HTML, want) {
			t.Fatalf("Portuguese HTML misses %q", want)
		}
	}
	if strings.Contains(pt.HTML, "<b>") || !strings.Contains(pt.Text, "25,00 €") || !strings.Contains(pt.Subject, "TEST 2026/000007") {
		t.Fatalf("unexpected Portuguese message: %+v", pt)
	}
	data.Invoice.BuyerTaxID = ""
	en, err := Render(TemplateInvoiceIssued, LocaleEnglish, data)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"€25.00", "Final consumer", "not a tax invoice"} {
		if !strings.Contains(en.HTML, want) || !strings.Contains(en.Text, want) {
			t.Fatalf("English message misses %q", want)
		}
	}
	if FormatMoney(123456, "USD", LocaleEnglish) != "1234.56 USD" {
		t.Fatal("other currencies keep their code")
	}
}
