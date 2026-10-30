package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ryze/backend/services/contact"
)

type stubContactService struct {
	err     error
	message contact.Message
	called  bool
}

func (s *stubContactService) Submit(_ context.Context, message contact.Message) error {
	s.called = true
	s.message = message
	return s.err
}

func contactRequest(t *testing.T, fields map[string]string, attachment []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %q: %v", key, err)
		}
	}
	if attachment != nil {
		part, err := writer.CreateFormFile("attachment", "brief.pdf")
		if err != nil {
			t.Fatalf("create attachment: %v", err)
		}
		if _, err := part.Write(attachment); err != nil {
			t.Fatalf("write attachment: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/contact", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func runContactRequest(t *testing.T, service contact.Service, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/contact", NewContactHandler(service).Submit)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestContactHandlerDeliversValidSubmission(t *testing.T) {
	service := &stubContactService{}
	recorder := runContactRequest(t, service, contactRequest(t, map[string]string{
		"name":    "Ada",
		"email":   "ada@example.com",
		"subject": "Hello",
		"message": "A message",
	}, []byte("%PDF-1.4\n")))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !service.called || service.message.Name != "Ada" || service.message.Attachment == nil {
		t.Fatalf("unexpected service call: %+v", service.message)
	}
}

func TestContactHandlerReportsUnavailableWithoutFakeSuccess(t *testing.T) {
	service := &stubContactService{err: contact.ErrUnavailable}
	recorder := runContactRequest(t, service, contactRequest(t, map[string]string{
		"name":    "Ada",
		"email":   "ada@example.com",
		"subject": "Hello",
		"message": "A message",
	}, nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Success || envelope.Error.Code != "CONTACT_UNAVAILABLE" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestContactHandlerRejectsHoneypotBeforeService(t *testing.T) {
	service := &stubContactService{}
	recorder := runContactRequest(t, service, contactRequest(t, map[string]string{
		"name":    "Bot",
		"email":   "bot@example.com",
		"subject": "Hello",
		"message": "A message",
		"website": "https://spam.example",
	}, nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if service.called {
		t.Fatal("honeypot submission must not reach the service")
	}
}

func TestContactHandlerRejectsInvalidSubmission(t *testing.T) {
	service := &stubContactService{err: contact.ErrInvalidInput}
	recorder := runContactRequest(t, service, contactRequest(t, map[string]string{
		"name":    "",
		"email":   "not-an-email",
		"subject": "",
		"message": "",
	}, nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}
