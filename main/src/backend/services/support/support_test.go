package support

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/notifications"
)

type recordingNotifier struct{ events []notifications.Event }

func (r *recordingNotifier) Notify(_ context.Context, event notifications.Event) {
	r.events = append(r.events, event)
}

type harness struct {
	tx       *gorm.DB
	svc      Service
	notifier *recordingNotifier
	users    repositories.UserRepository
	program  *models.Program
}

// newHarness runs against the isolated test database inside a transaction
// that is rolled back when the test ends.
func newHarness(t *testing.T) *harness {
	t.Helper()
	config.LoadEnvFile()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	tx := db.Begin()
	t.Cleanup(func() {
		tx.Rollback()
		// Each test opens its own pool; close it so repeated runs never
		// exhaust the server's connections.
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	notifier := &recordingNotifier{}
	program := &models.Program{Name: "Support test plan", Type: models.ProgramTypePremium, ProductType: models.ProgramProductTypeGeneric, Status: models.ProgramStatusPublished, PriceMinorUnits: 1000, Currency: "EUR"}
	if err := tx.Omit("TrainerID").Create(program).Error; err != nil {
		t.Fatalf("create program: %v", err)
	}
	return &harness{
		tx:       tx,
		svc:      New(repositories.NewSupportRepository(tx), notifier),
		notifier: notifier,
		users:    repositories.NewUserRepository(tx),
		program:  program,
	}
}

func (h *harness) user(t *testing.T, label string) *models.User {
	t.Helper()
	user := &models.User{Email: fmt.Sprintf("support-%s-%d@ryze.local", label, time.Now().UnixNano()), PasswordHash: "hash", FirstName: "Ana", LastName: label}
	if err := h.users.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func (h *harness) client(t *testing.T, label string) *models.User {
	t.Helper()
	user := h.user(t, label)
	if err := repositories.NewEntitlementRepository(h.tx).Create(context.Background(), user.ID, h.program.ID, &models.Entitlement{}); err != nil {
		t.Fatalf("create entitlement: %v", err)
	}
	return user
}

func TestEligibilityIsCheckedOnEveryOperation(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	visitor := h.user(t, "visitor")
	if eligibility, err := h.svc.Eligibility(ctx, visitor.ID); err != nil || eligibility.Eligible {
		t.Fatalf("an account without a programme is eligible: %+v %v", eligibility, err)
	}
	if _, err := h.svc.Open(ctx, visitor.ID, "Billing question", "Hello", ""); !errors.Is(err, ErrNotEligible) {
		t.Fatalf("expected ErrNotEligible, got %v", err)
	}
	if _, err := h.svc.ListMine(ctx, visitor.ID); !errors.Is(err, ErrNotEligible) {
		t.Fatalf("expected ErrNotEligible for the list, got %v", err)
	}

	// A Free activation counts.
	free := &models.Program{Name: "Free plan", Type: models.ProgramTypeFree, ProductType: models.ProgramProductTypeGeneric, Status: models.ProgramStatusPublished, Currency: "EUR"}
	if err := h.tx.Omit("TrainerID").Create(free).Error; err != nil {
		t.Fatal(err)
	}
	freeUser := h.user(t, "free")
	if err := h.tx.Create(&models.FreeProgramActivation{UserID: freeUser.ID, ProgramID: free.ID, ActivatedAt: time.Now()}).Error; err != nil {
		t.Fatalf("create activation: %v", err)
	}
	if eligibility, _ := h.svc.Eligibility(ctx, freeUser.ID); !eligibility.Eligible || eligibility.Role != models.SupportRoleClient {
		t.Fatalf("expected a Free activation to be eligible as client, got %+v", eligibility)
	}

	// Trainers are eligible as trainers.
	trainerUser := h.user(t, "trainer")
	if err := h.tx.Create(&models.Trainer{UserID: trainerUser.ID}).Error; err != nil {
		t.Fatal(err)
	}
	thread, err := h.svc.Open(ctx, trainerUser.ID, "Payout question", "When are payouts made?", "")
	if err != nil {
		t.Fatalf("trainer open: %v", err)
	}
	if thread.Conversation.RequesterRole != models.SupportRoleTrainer {
		t.Fatalf("expected the trainer role, got %s", thread.Conversation.RequesterRole)
	}

	// Losing the programme closes access to existing conversations too.
	client := h.client(t, "lapsed")
	opened, err := h.svc.Open(ctx, client.ID, "Access question", "Hi", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.tx.Exec("UPDATE entitlements SET deleted_at = now() WHERE user_id = ?", client.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.GetMine(ctx, client.ID, opened.Conversation.ID, 0); !errors.Is(err, ErrNotEligible) {
		t.Fatalf("expected ErrNotEligible after the entitlement ended, got %v", err)
	}
}

func TestConversationsBelongToTheirOwner(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	owner := h.client(t, "owner")
	other := h.client(t, "other")

	thread, err := h.svc.Open(ctx, owner.ID, "My programme", "  How do I log a set?\r\n ", "client-msg-0001")
	if err != nil {
		t.Fatal(err)
	}
	if thread.Messages[0].Body != "How do I log a set?" || thread.Messages[0].Seq == 0 {
		t.Fatalf("unexpected first message %+v", thread.Messages[0])
	}
	id := thread.Conversation.ID
	if _, err := h.svc.GetMine(ctx, other.ID, id, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another account read the conversation: %v", err)
	}
	if _, err := h.svc.SendMine(ctx, other.ID, id, "hi", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another account wrote to the conversation: %v", err)
	}
	if err := h.svc.MarkMineRead(ctx, other.ID, id, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another account marked the conversation: %v", err)
	}
	if list, _ := h.svc.ListMine(ctx, other.ID); len(list) != 0 {
		t.Fatal("another account lists the conversation")
	}
	// The stream of another account never carries the owner's messages.
	if messages, _ := h.svc.NewMessagesForUser(ctx, other.ID, 0); len(messages) != 0 {
		t.Fatal("another account's stream carries the owner's messages")
	}
	if messages, _ := h.svc.NewMessagesForUser(ctx, owner.ID, thread.Messages[0].Seq-1); len(messages) != 1 {
		t.Fatalf("expected the owner's stream to carry the message, got %d", len(messages))
	}
}

func TestMessagesAreValidatedAndIdempotent(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	owner := h.client(t, "writer")
	thread, err := h.svc.Open(ctx, owner.ID, "Question", "First", "")
	if err != nil {
		t.Fatal(err)
	}
	id := thread.Conversation.ID
	for _, body := range []string{"", "   ", strings.Repeat("a", 4001)} {
		if _, err := h.svc.SendMine(ctx, owner.ID, id, body, ""); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for body of %d chars, got %v", len(body), err)
		}
	}
	if _, err := h.svc.Open(ctx, owner.ID, "Hi", "Body", ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected a too-short subject to be refused, got %v", err)
	}
	if _, err := h.svc.SendMine(ctx, owner.ID, id, "x", "bad id!"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected a malformed client id to be refused, got %v", err)
	}

	first, err := h.svc.SendMine(ctx, owner.ID, id, "Resent after a dropped connection", "retry-key-12345")
	if err != nil {
		t.Fatal(err)
	}
	again, err := h.svc.SendMine(ctx, owner.ID, id, "Resent after a dropped connection", "retry-key-12345")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID || again.Seq != first.Seq {
		t.Fatalf("a resend created a second message: %+v vs %+v", first, again)
	}
	loaded, _ := h.svc.GetMine(ctx, owner.ID, id, 0)
	if len(loaded.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(loaded.Messages))
	}
}

func TestOpenConversationsAreBounded(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	owner := h.client(t, "busy")
	for i := 0; i < maxOpenConversations; i++ {
		if _, err := h.svc.Open(ctx, owner.ID, fmt.Sprintf("Question %d", i), "Body", ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.svc.Open(ctx, owner.ID, "One too many", "Body", ""); !errors.Is(err, ErrTooManyOpen) {
		t.Fatalf("expected ErrTooManyOpen, got %v", err)
	}
}

func TestStaffRepliesNotifyAndRespectClosing(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	owner := h.client(t, "reader")
	thread, err := h.svc.Open(ctx, owner.ID, "Nutrition plan", "Can I swap a meal?", "")
	if err != nil {
		t.Fatal(err)
	}
	id := thread.Conversation.ID

	staffList, err := h.svc.ListStaff(ctx, models.SupportStatusOpen, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if staffList.Unread < 1 {
		t.Fatalf("expected unread messages for staff, got %d", staffList.Unread)
	}

	reply, err := h.svc.Reply(ctx, "admin-2", id, "Yes, within the same day.", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Reply(ctx, "admin-2", id, "And keep the protein target.", ""); err != nil {
		t.Fatal(err)
	}
	if len(h.notifier.events) != 2 {
		t.Fatalf("expected two notifications, got %d", len(h.notifier.events))
	}
	if h.notifier.events[0].EmailTemplate != email.TemplateSupportReply || h.notifier.events[1].EmailTemplate != "" {
		t.Fatalf("expected an email only for the first unread reply, got %+v", h.notifier.events)
	}
	if h.notifier.events[0].UserID != owner.ID || h.notifier.events[0].DedupeKey != "support_reply:"+reply.ID {
		t.Fatalf("unexpected notification %+v", h.notifier.events[0])
	}

	mine, _ := h.svc.ListMine(ctx, owner.ID)
	if len(mine) != 1 || mine[0].Unread != 2 {
		t.Fatalf("expected two unread replies, got %+v", mine)
	}
	loaded, _ := h.svc.GetMine(ctx, owner.ID, id, 0)
	last := loaded.Messages[len(loaded.Messages)-1]
	if last.AuthorKind != models.SupportAuthorStaff || last.AuthorAdminID == nil {
		t.Fatalf("expected the reply to be stored as staff, got %+v", last)
	}
	if err := h.svc.MarkMineRead(ctx, owner.ID, id, last.Seq); err != nil {
		t.Fatal(err)
	}
	if mine, _ = h.svc.ListMine(ctx, owner.ID); mine[0].Unread != 0 {
		t.Fatalf("expected no unread replies after reading, got %d", mine[0].Unread)
	}
	// After reading, the next reply emails again.
	if _, err := h.svc.Reply(ctx, "admin-2", id, "Anything else?", ""); err != nil {
		t.Fatal(err)
	}
	if h.notifier.events[2].EmailTemplate != email.TemplateSupportReply {
		t.Fatal("expected an email for the first reply after reading")
	}

	if _, err := h.svc.SetStatus(ctx, "admin-2", id, models.SupportStatusClosed); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SendMine(ctx, owner.ID, id, "One more thing", ""); !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
	if closedList, _ := h.svc.ListStaff(ctx, models.SupportStatusClosed, 1, 20); closedList.Total < 1 {
		t.Fatal("expected the conversation in the closed list")
	}
	if _, err := h.svc.SetStatus(ctx, "admin-2", id, models.SupportStatusOpen); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SendMine(ctx, owner.ID, id, "One more thing", ""); err != nil {
		t.Fatalf("expected a reopened conversation to accept messages, got %v", err)
	}
	staffThread, err := h.svc.GetStaff(ctx, id, 0)
	if err != nil || staffThread.Requester == nil || staffThread.Requester.Email != owner.Email {
		t.Fatalf("expected staff to see the requester, got %+v %v", staffThread, err)
	}
}

func TestMessagePagesGoBackwards(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	owner := h.client(t, "history")
	thread, err := h.svc.Open(ctx, owner.ID, "Long thread", "message 0", "")
	if err != nil {
		t.Fatal(err)
	}
	id := thread.Conversation.ID
	for i := 1; i <= messagePageSize+4; i++ {
		if _, err := h.svc.SendMine(ctx, owner.ID, id, fmt.Sprintf("message %d", i), ""); err != nil {
			t.Fatal(err)
		}
	}
	latest, _ := h.svc.GetMine(ctx, owner.ID, id, 0)
	if len(latest.Messages) != messagePageSize || !latest.HasOlder || latest.Messages[len(latest.Messages)-1].Body != fmt.Sprintf("message %d", messagePageSize+4) {
		t.Fatalf("unexpected latest page: %d messages, older=%v", len(latest.Messages), latest.HasOlder)
	}
	older, _ := h.svc.GetMine(ctx, owner.ID, id, latest.Messages[0].Seq)
	if len(older.Messages) != 5 || older.HasOlder || older.Messages[0].Body != "message 0" {
		t.Fatalf("unexpected older page: %d messages, older=%v", len(older.Messages), older.HasOlder)
	}
}

func TestHubWakesOnlyTheOwnerAndStaff(t *testing.T) {
	hub := NewHub()
	owner, releaseOwner, err := hub.Subscribe("user-a")
	if err != nil {
		t.Fatal(err)
	}
	other, releaseOther, _ := hub.Subscribe("user-b")
	staff, releaseStaff, _ := hub.Subscribe(StaffScope)
	hub.Publish("user-a")
	hub.Publish("user-a") // coalesced, never blocks
	for name, ch := range map[string]<-chan struct{}{"owner": owner, "staff": staff} {
		select {
		case <-ch:
		default:
			t.Fatalf("expected the %s stream to wake", name)
		}
	}
	select {
	case <-other:
		t.Fatal("another account's stream was woken")
	default:
	}
	releaseOwner()
	releaseOwner() // releasing twice is harmless
	releaseOther()
	releaseStaff()
	if hub.Subscribers() != 0 {
		t.Fatal("expected every subscriber to be released")
	}
}

func TestHubBoundsStreamsPerAccount(t *testing.T) {
	hub := NewHub()
	var releases []func()
	for i := 0; i < maxStreamsPerScope; i++ {
		_, release, err := hub.Subscribe("user-a")
		if err != nil {
			t.Fatalf("stream %d refused: %v", i, err)
		}
		releases = append(releases, release)
	}
	if _, _, err := hub.Subscribe("user-a"); !errors.Is(err, ErrTooManyStreams) {
		t.Fatalf("expected ErrTooManyStreams, got %v", err)
	}
	if _, release, err := hub.Subscribe("user-b"); err != nil {
		t.Fatalf("another account was refused: %v", err)
	} else {
		release()
	}
	releases[0]()
	if _, _, err := hub.Subscribe("user-a"); err != nil {
		t.Fatalf("expected a slot after a release, got %v", err)
	}
}

func TestOpenIsIdempotentAndClientIDsBelongToTheirAuthor(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	owner := h.client(t, "retry-open")

	first, err := h.svc.Open(ctx, owner.ID, "Retried question", "Hello", "open-key-123456")
	if err != nil {
		t.Fatal(err)
	}
	again, err := h.svc.Open(ctx, owner.ID, "Retried question", "Hello", "open-key-123456")
	if err != nil {
		t.Fatal(err)
	}
	if again.Conversation.ID != first.Conversation.ID {
		t.Fatal("a retried open created a second conversation")
	}
	if list, _ := h.svc.ListMine(ctx, owner.ID); len(list) != 1 {
		t.Fatalf("expected one conversation, got %d", len(list))
	}

	// A staff reply's client id cannot be replayed by the account to read it
	// back as its own message.
	if _, err := h.svc.Reply(ctx, "admin-2", first.Conversation.ID, "Staff text", "staff-key-123456"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SendMine(ctx, owner.ID, first.Conversation.ID, "mine", "staff-key-123456"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for another author's client id, got %v", err)
	}
}
