package notifications

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"ryze/backend/config"
	"ryze/backend/database"
	"ryze/backend/email"
	"ryze/backend/models"
	"ryze/backend/repositories"
)

type harness struct {
	tx    *gorm.DB
	svc   *service
	users repositories.UserRepository
	clock time.Time
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

	users := repositories.NewUserRepository(tx)
	mailCfg := config.MailConfig{SMTPHost: "smtp.example", SMTPPort: "587", From: "RYZE <no-reply@ryze.example>", PublicAppURL: "https://ryze.example"}
	mailer := email.NewMailer(mailCfg, repositories.NewEmailOutboxRepository(tx))
	h := &harness{tx: tx, users: users, clock: time.Now().UTC().Truncate(time.Microsecond)}
	h.svc = New(repositories.NewNotificationRepository(tx), users, repositories.NewProgramRepository(tx), mailer).(*service)
	h.svc.now = func() time.Time { return h.clock }
	return h
}

func (h *harness) seedUser(t *testing.T, persona bool) *models.User {
	t.Helper()
	user := &models.User{Email: fmt.Sprintf("notify-%d@ryze.local", time.Now().UnixNano()), PasswordHash: "hash", FirstName: "Rita", LastName: "Costa"}
	if persona {
		value := "client"
		user.TestPersona = &value
	}
	if err := h.users.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func (h *harness) emails(t *testing.T, userID string) []models.EmailOutbox {
	t.Helper()
	var rows []models.EmailOutbox
	if err := h.tx.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestNotifyRecordsOncePerEvent(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	user := h.seedUser(t, false)
	event := Event{
		UserID: user.ID, Kind: models.NotificationPL2Delivered, Program: "Built around you",
		Link: "/services/premium-level-2/requests/abc", DedupeKey: "pl2_delivered:test-" + user.ID,
		EmailTemplate: email.TemplateServiceDelivered,
	}
	h.svc.Notify(ctx, event)
	h.svc.Notify(ctx, event) // a repeated delivery of the same event

	list, err := h.svc.List(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Notifications) != 1 || list.Unread != 1 {
		t.Fatalf("expected one unread notification, got %d (unread %d)", len(list.Notifications), list.Unread)
	}
	got := list.Notifications[0]
	if got.Kind != models.NotificationPL2Delivered || got.Link == nil || *got.Link != event.Link || string(got.Params) != `{"program": "Built around you"}` {
		t.Fatalf("unexpected notification %+v params=%s", got, got.Params)
	}
	if !got.ExpiresAt.Equal(h.clock.Add(24 * time.Hour)) {
		t.Fatalf("expected a 24 hour lifetime, got %v", got.ExpiresAt.Sub(h.clock))
	}
	if emails := h.emails(t, user.ID); len(emails) != 1 || emails[0].Template != email.TemplateServiceDelivered {
		t.Fatalf("expected exactly one email, got %+v", emails)
	}
}

func TestNotificationsBelongToTheirOwner(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	owner := h.seedUser(t, false)
	other := h.seedUser(t, false)
	h.svc.Notify(ctx, Event{UserID: owner.ID, Kind: models.NotificationAccessGranted, DedupeKey: "own-" + owner.ID})
	h.svc.Notify(ctx, Event{UserID: owner.ID, Kind: models.NotificationAccessGranted, DedupeKey: "own2-" + owner.ID})

	list, _ := h.svc.List(ctx, owner.ID)
	if len(list.Notifications) != 2 {
		t.Fatalf("expected two notifications, got %d", len(list.Notifications))
	}
	if otherList, _ := h.svc.List(ctx, other.ID); len(otherList.Notifications) != 0 {
		t.Fatal("another account sees the owner's notifications")
	}
	id := list.Notifications[0].ID
	if err := h.svc.MarkRead(ctx, other.ID, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected another account to be refused, got %v", err)
	}
	if err := h.svc.MarkRead(ctx, owner.ID, id); err != nil {
		t.Fatal(err)
	}
	if list, _ = h.svc.List(ctx, owner.ID); list.Unread != 1 {
		t.Fatalf("expected one unread after marking one, got %d", list.Unread)
	}
	if err := h.svc.MarkAllRead(ctx, owner.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ = h.svc.List(ctx, owner.ID); list.Unread != 0 {
		t.Fatalf("expected none unread, got %d", list.Unread)
	}
}

func TestNotificationsExpireAndAreRetainedOnlyAsTombstones(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	user := h.seedUser(t, false)
	key := "access_granted:expiry-" + user.ID
	h.svc.Notify(ctx, Event{UserID: user.ID, Kind: models.NotificationAccessGranted, Program: "Plan", Link: "/services/my-programs", DedupeKey: key})

	start := h.clock
	h.clock = start.Add(24*time.Hour + time.Second)
	list, _ := h.svc.List(ctx, user.ID)
	if len(list.Notifications) != 0 {
		t.Fatal("an expired notification is still listed")
	}
	if err := h.svc.MarkRead(ctx, user.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	if err := h.svc.Purge(ctx); err != nil {
		t.Fatal(err)
	}
	var tombstone models.Notification
	if err := h.tx.Where("dedupe_key = ?", key).Take(&tombstone).Error; err != nil {
		t.Fatalf("expected the tombstone to remain: %v", err)
	}
	if tombstone.Link != nil || string(tombstone.Params) != "{}" {
		t.Fatalf("expected cleared content, got link=%v params=%s", tombstone.Link, tombstone.Params)
	}
	// A redelivered event after expiry does not notify again.
	h.svc.Notify(ctx, Event{UserID: user.ID, Kind: models.NotificationAccessGranted, DedupeKey: key})
	if list, _ := h.svc.List(ctx, user.ID); len(list.Notifications) != 0 {
		t.Fatal("a repeated event notified again after expiry")
	}

	h.clock = start.Add(24*time.Hour + 31*24*time.Hour)
	if err := h.svc.Purge(ctx); err != nil {
		t.Fatal(err)
	}
	var count int64
	h.tx.Model(&models.Notification{}).Where("dedupe_key = ?", key).Count(&count)
	if count != 0 {
		t.Fatal("expected the tombstone to be deleted after 30 days")
	}
}

func TestPersonasGetNoEmail(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	persona := h.seedUser(t, true)
	h.svc.Notify(ctx, Event{UserID: persona.ID, Kind: models.NotificationPL2Delivered, Program: "Plan", Link: "/x", DedupeKey: "persona-" + persona.ID, EmailTemplate: email.TemplateServiceDelivered})
	if list, _ := h.svc.List(ctx, persona.ID); len(list.Notifications) != 1 {
		t.Fatal("expected the in-app notification")
	}
	if emails := h.emails(t, persona.ID); len(emails) != 0 {
		t.Fatalf("expected no email for a Test Mode persona, got %d", len(emails))
	}
}

func TestUnsafeLinksAreDropped(t *testing.T) {
	for _, link := range []string{"https://evil.example", "//evil.example", "/\\evil", "/ok\r\nx"} {
		if safeLink(link) != "" {
			t.Errorf("safeLink(%q) kept an unsafe link", link)
		}
	}
	if safeLink("/services/my-programs") != "/services/my-programs" {
		t.Error("safeLink dropped a valid path")
	}
}

func TestPurchaseFulfilmentNotifiesOncePerPurchase(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	buyer := h.seedUser(t, false)
	program := &models.Program{Name: "Strength base", Type: models.ProgramTypePremium, ProductType: models.ProgramProductTypeGeneric, Status: models.ProgramStatusPublished, PriceMinorUnits: 1500, Currency: "EUR"}
	if err := h.tx.Omit("TrainerID").Create(program).Error; err != nil {
		t.Fatalf("create program: %v", err)
	}
	// No completed purchase yet: nothing to report.
	if err := h.svc.FulfillPurchase(ctx, buyer.ID, program.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := h.svc.List(ctx, buyer.ID); len(list.Notifications) != 0 {
		t.Fatal("notified without a completed purchase")
	}
	purchase := &models.Purchase{UserID: buyer.ID, ProgramID: program.ID, PriceMinorUnits: 1500, Currency: "EUR", Status: models.PurchaseStatusCompleted}
	if err := h.tx.Create(purchase).Error; err != nil {
		t.Fatalf("create purchase: %v", err)
	}
	for i := 0; i < 3; i++ { // webhook, capture callback and a redelivery
		if err := h.svc.FulfillPurchase(ctx, buyer.ID, program.ID); err != nil {
			t.Fatal(err)
		}
	}
	list, _ := h.svc.List(ctx, buyer.ID)
	if len(list.Notifications) != 1 || list.Notifications[0].Kind != models.NotificationAccessGranted || *list.Notifications[0].Link != "/services/my-programs" {
		t.Fatalf("expected one access notification, got %+v", list.Notifications)
	}
	if emails := h.emails(t, buyer.ID); len(emails) != 1 || emails[0].Template != email.TemplateAccessGranted {
		t.Fatalf("expected one access email, got %+v", emails)
	}
}
