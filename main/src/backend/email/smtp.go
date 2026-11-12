// Package email delivers RYZE transactional emails. Business events record an
// email in the outbox (Mailer.Enqueue); the Worker renders and sends due emails
// through an SMTP relay that must offer STARTTLS, with retries and backoff.
package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"time"

	"ryze/backend/config"
)

const smtpTimeout = 20 * time.Second

// Message is one rendered email.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Sender delivers one rendered message.
type Sender interface {
	Send(ctx context.Context, message Message) error
}

// ErrPermanent marks a delivery failure that a retry cannot fix (an invalid
// recipient or a rejected address).
var ErrPermanent = errors.New("permanent email delivery failure")

// SMTPSender delivers through the configured relay. Every connection must
// upgrade with STARTTLS before credentials or content are sent; a relay that
// does not offer it is refused rather than downgraded to plaintext.
type SMTPSender struct {
	config config.MailConfig
}

func NewSMTPSender(cfg config.MailConfig) *SMTPSender {
	return &SMTPSender{config: cfg}
}

func (s *SMTPSender) Send(ctx context.Context, message Message) error {
	from, err := mail.ParseAddress(s.config.From)
	if err != nil {
		return fmt.Errorf("%w: invalid sender", ErrPermanent)
	}
	to, err := mail.ParseAddress(message.To)
	if err != nil {
		return fmt.Errorf("%w: invalid recipient", ErrPermanent)
	}
	payload, err := buildMessage(from, to, message)
	if err != nil {
		return err
	}

	dialer := net.Dialer{Timeout: smtpTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(s.config.SMTPHost, s.config.SMTPPort))
	if err != nil {
		return errors.New("smtp relay unreachable")
	}
	defer conn.Close()
	deadline := time.Now().Add(smtpTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	_ = conn.SetDeadline(deadline)

	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return errors.New("smtp handshake failed")
	}
	defer client.Close()
	if err := client.Hello("localhost"); err != nil {
		return errors.New("smtp hello failed")
	}
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return errors.New("smtp relay does not offer starttls")
	}
	if err := client.StartTLS(&tls.Config{ServerName: s.config.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
		return errors.New("smtp starttls failed")
	}
	if s.config.SMTPUsername != "" {
		if err := client.Auth(smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)); err != nil {
			return errors.New("smtp authentication failed")
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("%w: sender rejected", ErrPermanent)
	}
	if err := client.Rcpt(to.Address); err != nil {
		// 5xx replies are permanent; anything else may succeed later.
		var smtpErr *textproto.Error
		if errors.As(err, &smtpErr) && smtpErr.Code >= 500 {
			return fmt.Errorf("%w: recipient rejected", ErrPermanent)
		}
		return errors.New("smtp recipient temporarily rejected")
	}
	writer, err := client.Data()
	if err != nil {
		return errors.New("smtp data refused")
	}
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return errors.New("smtp write failed")
	}
	if err := writer.Close(); err != nil {
		return errors.New("smtp message not accepted")
	}
	return client.Quit()
}

// buildMessage writes a multipart/alternative message with a plain-text and an
// HTML part.
func buildMessage(from, to *mail.Address, message Message) ([]byte, error) {
	var body bytes.Buffer
	alternative := multipart.NewWriter(&body)

	headers := textproto.MIMEHeader{}
	headers.Set("From", from.String())
	headers.Set("To", to.String())
	headers.Set("Subject", mime.QEncoding.Encode("utf-8", message.Subject))
	headers.Set("Date", time.Now().Format(time.RFC1123Z))
	headers.Set("MIME-Version", "1.0")
	headers.Set("Content-Type", "multipart/alternative; boundary="+alternative.Boundary())

	var out bytes.Buffer
	for key, values := range headers {
		for _, value := range values {
			fmt.Fprintf(&out, "%s: %s\r\n", key, value)
		}
	}
	out.WriteString("\r\n")

	for _, part := range []struct {
		contentType string
		content     string
	}{
		{"text/plain; charset=utf-8", message.Text},
		{"text/html; charset=utf-8", message.HTML},
	} {
		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Type", part.contentType)
		partHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		writer, err := alternative.CreatePart(partHeader)
		if err != nil {
			return nil, fmt.Errorf("create email part: %w", err)
		}
		encoder := quotedprintable.NewWriter(writer)
		if _, err := encoder.Write([]byte(part.content)); err != nil {
			return nil, fmt.Errorf("encode email part: %w", err)
		}
		if err := encoder.Close(); err != nil {
			return nil, fmt.Errorf("close email part: %w", err)
		}
	}
	if err := alternative.Close(); err != nil {
		return nil, fmt.Errorf("close email: %w", err)
	}
	out.Write(body.Bytes())
	return out.Bytes(), nil
}
