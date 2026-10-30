package contact_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ryze/backend/services/contact"
)

type recordingNotifier struct {
	message contact.Message
	err     error
}

func (n *recordingNotifier) Send(_ context.Context, message contact.Message) error {
	n.message = message
	return n.err
}

func validMessage() contact.Message {
	return contact.Message{
		Name:    "  Ada Lovelace  ",
		Email:   "ada@example.com",
		Subject: "  Training plan question  ",
		Body:    "  How should I start?  ",
	}
}

func TestSubmitNormalizesAndDelivers(t *testing.T) {
	notifier := &recordingNotifier{}
	service := contact.NewService(notifier)

	if err := service.Submit(context.Background(), validMessage()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if notifier.message.Name != "Ada Lovelace" || notifier.message.Subject != "Training plan question" || notifier.message.Body != "How should I start?" {
		t.Fatalf("unexpected normalized message: %+v", notifier.message)
	}
}

func TestSubmitRejectsInvalidFields(t *testing.T) {
	service := contact.NewService(&recordingNotifier{})

	cases := map[string]contact.Message{
		"empty name":    {Name: "", Email: "ada@example.com", Subject: "Hi", Body: "Body"},
		"header name":   {Name: "Ada\r\nBcc: x", Email: "ada@example.com", Subject: "Hi", Body: "Body"},
		"invalid email": {Name: "Ada", Email: "not-an-email", Subject: "Hi", Body: "Body"},
		"display email": {Name: "Ada", Email: "Ada <ada@example.com>", Subject: "Hi", Body: "Body"},
		"empty subject": {Name: "Ada", Email: "ada@example.com", Subject: " ", Body: "Body"},
		"empty body":    {Name: "Ada", Email: "ada@example.com", Subject: "Hi", Body: " "},
		"long body":     {Name: "Ada", Email: "ada@example.com", Subject: "Hi", Body: strings.Repeat("a", contact.MaxMessageLength+1)},
	}

	for name, message := range cases {
		t.Run(name, func(t *testing.T) {
			if err := service.Submit(context.Background(), message); !errors.Is(err, contact.ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestSubmitValidatesAttachments(t *testing.T) {
	service := contact.NewService(&recordingNotifier{})

	valid := validMessage()
	valid.Attachment = &contact.Attachment{
		Filename: "../../brief.pdf",
		Data:     []byte("%PDF-1.4\n"),
	}
	notifier := &recordingNotifier{}
	service = contact.NewService(notifier)
	if err := service.Submit(context.Background(), valid); err != nil {
		t.Fatalf("valid attachment: %v", err)
	}
	if notifier.message.Attachment == nil || notifier.message.Attachment.Filename != "brief.pdf" {
		t.Fatalf("expected sanitized attachment, got %+v", notifier.message.Attachment)
	}

	invalidType := validMessage()
	invalidType.Attachment = &contact.Attachment{Filename: "script.sh", Data: []byte("#!/bin/sh")}
	if err := service.Submit(context.Background(), invalidType); !errors.Is(err, contact.ErrInvalidInput) {
		t.Fatalf("expected invalid attachment type, got %v", err)
	}

	oversized := validMessage()
	oversized.Attachment = &contact.Attachment{
		Filename: "large.pdf",
		Data:     make([]byte, contact.MaxAttachmentBytes+1),
	}
	if err := service.Submit(context.Background(), oversized); !errors.Is(err, contact.ErrInvalidInput) {
		t.Fatalf("expected oversized attachment rejection, got %v", err)
	}
}

func TestSubmitNeverReportsSuccessWhenDeliveryFails(t *testing.T) {
	service := contact.NewService(&recordingNotifier{err: errors.New("smtp down")})

	if err := service.Submit(context.Background(), validMessage()); !errors.Is(err, contact.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

func TestSubmitWithoutNotifierIsUnavailable(t *testing.T) {
	service := contact.NewService(nil)

	if err := service.Submit(context.Background(), validMessage()); !errors.Is(err, contact.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}
