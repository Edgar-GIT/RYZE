package auth_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ryze/backend/api/auth"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/services/support"
)

// fakeSupport implements only what the stream needs; any other call panics.
type fakeSupport struct {
	support.Service
	hub      *support.Hub
	eligible bool

	mu       sync.Mutex
	messages []models.SupportMessage
	afters   []int64
}

func (f *fakeSupport) Hub() *support.Hub { return f.hub }

func (f *fakeSupport) Eligibility(context.Context, string) (support.Eligibility, error) {
	return support.Eligibility{Eligible: f.eligible, Role: models.SupportRoleClient}, nil
}

func (f *fakeSupport) CurrentSeq(context.Context) (int64, error) { return 0, nil }

func (f *fakeSupport) NewMessagesForUser(_ context.Context, _ string, after int64) ([]models.SupportMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.afters = append(f.afters, after)
	var out []models.SupportMessage
	for _, message := range f.messages {
		if message.Seq > after {
			out = append(out, message)
		}
	}
	return out, nil
}

func (f *fakeSupport) add(message models.SupportMessage) {
	f.mu.Lock()
	f.messages = append(f.messages, message)
	f.mu.Unlock()
	f.hub.Publish(streamUserID)
}

const streamUserID = "5f0c2f6e-1d7a-4c0e-9a59-0c1b8f7f3a11"

func newStreamServer(t *testing.T, fake *fakeSupport) *httptest.Server {
	t.Helper()
	handler := auth.NewSupportHandler(fake)
	router := gin.New()
	router.GET("/stream", func(c *gin.Context) {
		c.Set(authcontext.UserIDContextKey, streamUserID)
		handler.StreamMine(c)
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

func message(seq int64, body string) models.SupportMessage {
	return models.SupportMessage{ID: "m" + body, Seq: seq, ConversationID: "c1", AuthorKind: models.SupportAuthorStaff, Body: body}
}

// readUntil reads stream lines until one contains want.
func readUntil(t *testing.T, lines <-chan string, want string) []string {
	t.Helper()
	var seen []string
	deadline := time.After(5 * time.Second)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatalf("stream ended before %q; saw %v", want, seen)
			}
			seen = append(seen, line)
			if strings.Contains(line, want) {
				return seen
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %q; saw %v", want, seen)
		}
	}
}

func TestSupportStreamResumesAndPushes(t *testing.T) {
	fake := &fakeSupport{hub: support.NewHub(), eligible: true}
	fake.messages = []models.SupportMessage{message(4, "old"), message(6, "missed-one"), message(7, "missed-two")}
	server := newStreamServer(t, fake)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/stream", nil)
	// The browser sends the last id it saw when it reconnects.
	request.Header.Set("Last-Event-ID", "5")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("unexpected response %d %s", response.StatusCode, response.Header.Get("Content-Type"))
	}

	lines := make(chan string)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	seen := readUntil(t, lines, `"body":"missed-two"`)
	joined := strings.Join(seen, "\n")
	if !strings.Contains(joined, "retry: 3000") || !strings.Contains(joined, "id: 6") || !strings.Contains(joined, "id: 7") || strings.Contains(joined, `"old"`) {
		t.Fatalf("unexpected resume output:\n%s", joined)
	}
	if strings.Contains(joined, "author_admin_id") || strings.Contains(joined, "author_user_id") {
		t.Fatalf("the stream exposes author identities:\n%s", joined)
	}

	// A new message wakes the stream immediately.
	fake.add(message(8, "live"))
	seen = readUntil(t, lines, `"body":"live"`)
	if !strings.Contains(strings.Join(seen, "\n"), "id: 8") {
		t.Fatalf("expected id 8, saw %v", seen)
	}
	cancel()
	// The stream releases its hub subscription when the client leaves.
	deadline := time.Now().Add(3 * time.Second)
	for fake.hub.Subscribers() != 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if fake.hub.Subscribers() != 0 {
		t.Fatal("the stream kept its subscription after the client left")
	}
}

func TestSupportStreamRefusesIneligibleAccounts(t *testing.T) {
	fake := &fakeSupport{hub: support.NewHub(), eligible: false}
	server := newStreamServer(t, fake)
	response, err := http.Get(server.URL + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.StatusCode)
	}
	if fake.hub.Subscribers() != 0 {
		t.Fatal("an ineligible account subscribed to the stream")
	}
}
