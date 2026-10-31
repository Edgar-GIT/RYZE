package contact_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"ryze/backend/config"
	"ryze/backend/services/contact"
)

// TestSMTPNotifierRefusesRelayWithoutSTARTTLS proves that a relay which does
// not advertise STARTTLS never receives envelope data or message content. The
// contact form carries personal data, so a silent plaintext downgrade would be
// a security regression rather than a compatibility fallback.
func TestSMTPNotifierRefusesRelayWithoutSTARTTLS(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().(*net.TCPAddr)
	relayDone := make(chan struct{})
	mailSeen := make(chan struct{}, 1)

	go func() {
		defer close(relayDone)

		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		write := func(line string) {
			_, _ = fmt.Fprintf(conn, "%s\r\n", line)
		}

		write("220 test ESMTP")
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			command := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
				write("250-test")
				write("250 OK")
			case strings.HasPrefix(command, "MAIL FROM:"):
				select {
				case mailSeen <- struct{}{}:
				default:
				}
				write("250 OK")
			case strings.HasPrefix(command, "QUIT"):
				write("221 Bye")
				return
			default:
				write("250 OK")
			}
		}
	}()

	notifier := contact.NewSMTPNotifier(config.ContactConfig{
		SMTPHost:  "127.0.0.1",
		SMTPPort:  fmt.Sprint(address.Port),
		From:      "RYZE Test <contact@ryze.test>",
		Recipient: "Support Test <support@ryze.test>",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = notifier.Send(ctx, contact.Message{
		Name:    "Test",
		Email:   "test@example.test",
		Subject: "Test",
		Body:    "Test body",
	})
	if err == nil || !strings.Contains(err.Error(), "does not offer starttls") {
		t.Fatalf("expected STARTTLS refusal, got %v", err)
	}

	select {
	case <-mailSeen:
		t.Fatal("relay received MAIL FROM before STARTTLS was established")
	default:
	}

	select {
	case <-relayDone:
	case <-time.After(time.Second):
		t.Fatal("test relay did not terminate")
	}
}
