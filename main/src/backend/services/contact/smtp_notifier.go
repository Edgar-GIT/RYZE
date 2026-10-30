package contact

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
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

const smtpTimeout = 15 * time.Second

// SMTPNotifier delivers contact messages through the configured SMTP relay.
// The recipient comes only from configuration, never from the request.
type SMTPNotifier struct {
	config config.ContactConfig
}

func NewSMTPNotifier(cfg config.ContactConfig) *SMTPNotifier {
	return &SMTPNotifier{config: cfg}
}

// Send opens one SMTP connection per message, upgrades to STARTTLS when
// advertised, and refuses to send credentials over an unencrypted connection.
func (n *SMTPNotifier) Send(ctx context.Context, message Message) error {
	if !n.config.Enabled() {
		return ErrUnavailable
	}

	from, err := mail.ParseAddress(n.config.From)
	if err != nil {
		return fmt.Errorf("invalid contact sender: %w", err)
	}
	to, err := mail.ParseAddress(n.config.Recipient)
	if err != nil {
		return fmt.Errorf("invalid contact recipient: %w", err)
	}

	payload, err := buildMessage(from, to, message)
	if err != nil {
		return err
	}

	address := net.JoinHostPort(n.config.SMTPHost, n.config.SMTPPort)
	dialer := net.Dialer{Timeout: smtpTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(smtpTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	_ = conn.SetDeadline(deadline)

	client, err := smtp.NewClient(conn, n.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: n.config.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	} else if n.config.SMTPUsername != "" {
		return fmt.Errorf("smtp server does not offer starttls")
	}

	if n.config.SMTPUsername != "" {
		auth := smtp.PlainAuth("", n.config.SMTPUsername, n.config.SMTPPassword, n.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to.Address); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}

func buildMessage(from, to *mail.Address, message Message) ([]byte, error) {
	headers := textproto.MIMEHeader{}
	headers.Set("From", from.String())
	headers.Set("To", to.String())
	headers.Set("Reply-To", (&mail.Address{Name: message.Name, Address: message.Email}).String())
	headers.Set("Subject", mime.QEncoding.Encode("utf-8", message.Subject))
	headers.Set("Date", time.Now().Format(time.RFC1123Z))
	headers.Set("MIME-Version", "1.0")

	var body bytes.Buffer
	if message.Attachment == nil {
		headers.Set("Content-Type", "text/plain; charset=utf-8")
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
		writeHeaders(&body, headers)
		writer := quotedprintable.NewWriter(&body)
		if _, err := writer.Write([]byte(message.Body)); err != nil {
			return nil, fmt.Errorf("encode contact body: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("close contact body: %w", err)
		}
		return body.Bytes(), nil
	}

	mixed := multipart.NewWriter(&body)
	headers.Set("Content-Type", "multipart/mixed; boundary="+mixed.Boundary())
	writeHeaders(&body, headers)

	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=utf-8")
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	textPart, err := mixed.CreatePart(textHeader)
	if err != nil {
		return nil, fmt.Errorf("create contact text part: %w", err)
	}
	textWriter := quotedprintable.NewWriter(textPart)
	if _, err := textWriter.Write([]byte(message.Body)); err != nil {
		return nil, fmt.Errorf("encode contact body: %w", err)
	}
	if err := textWriter.Close(); err != nil {
		return nil, fmt.Errorf("close contact body: %w", err)
	}

	attachmentHeader := textproto.MIMEHeader{}
	attachmentHeader.Set("Content-Type", message.Attachment.ContentType)
	attachmentHeader.Set("Content-Transfer-Encoding", "base64")
	attachmentHeader.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": message.Attachment.Filename}))
	attachmentPart, err := mixed.CreatePart(attachmentHeader)
	if err != nil {
		return nil, fmt.Errorf("create contact attachment part: %w", err)
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(message.Attachment.Data)))
	base64.StdEncoding.Encode(encoded, message.Attachment.Data)
	if _, err := attachmentPart.Write(encoded); err != nil {
		return nil, fmt.Errorf("write contact attachment: %w", err)
	}
	if err := mixed.Close(); err != nil {
		return nil, fmt.Errorf("close contact message: %w", err)
	}
	return body.Bytes(), nil
}

func writeHeaders(buffer *bytes.Buffer, headers textproto.MIMEHeader) {
	for key, values := range headers {
		for _, value := range values {
			fmt.Fprintf(buffer, "%s: %s\r\n", key, value)
		}
	}
	buffer.WriteString("\r\n")
}
