package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ryze/backend/models"
)

var (
	// ErrSupportConversationNotFound covers unknown conversations and, for an
	// account, conversations it does not own.
	ErrSupportConversationNotFound = errors.New("support conversation not found")
	// ErrSupportConversationClosed indicates the conversation takes no new
	// messages from the account.
	ErrSupportConversationClosed = errors.New("support conversation closed")
	// ErrSupportTooManyOpen indicates the account reached its open limit.
	ErrSupportTooManyOpen = errors.New("too many open support conversations")
	// ErrSupportClientIDTaken indicates the client message id belongs to a
	// message by another author.
	ErrSupportClientIDTaken = errors.New("client message id already used")
)

// SupportEligibility tells whether an account may use support, and as what.
type SupportEligibility struct {
	Trainer bool
	Client  bool
}

// SupportConversationSummary is a conversation with its unread count for the
// reader and, for staff, the requester's name.
type SupportConversationSummary struct {
	models.SupportConversation
	Unread         int64  `gorm:"column:unread"`
	FirstName      string `gorm:"column:first_name"`
	LastName       string `gorm:"column:last_name"`
	Email          string `gorm:"column:email"`
	LastMessageSeq int64  `gorm:"column:last_message_seq"`
}

// SupportRepository stores support conversations and messages.
type SupportRepository interface {
	Eligibility(ctx context.Context, userID string, now time.Time) (SupportEligibility, error)
	// CreateConversation creates a conversation with its first message. It
	// refuses with ErrSupportTooManyOpen when the account already has maxOpen
	// open conversations (counted under the insert lock).
	CreateConversation(ctx context.Context, conversation *models.SupportConversation, first *models.SupportMessage, maxOpen int64) error
	// FindOpenedByClientID returns the conversation the account opened with
	// this client message id, if any (a retried "open").
	FindOpenedByClientID(ctx context.Context, userID, clientMessageID string) (*models.SupportConversation, error)
	FindConversation(ctx context.Context, id string) (*models.SupportConversation, error)
	ListForUser(ctx context.Context, userID string) ([]SupportConversationSummary, error)
	ListForStaff(ctx context.Context, status string, page, limit int) ([]SupportConversationSummary, int64, error)
	// AddMessage appends a message. A repeated client_message_id returns the
	// stored message (created=false). A user message to a closed conversation
	// fails with ErrSupportConversationClosed.
	AddMessage(ctx context.Context, message *models.SupportMessage) (created bool, err error)
	Messages(ctx context.Context, conversationID string, beforeSeq int64, limit int) ([]models.SupportMessage, error)
	// MessagesSince returns messages after a sequence, oldest first, for one
	// account's conversations (userID set) or for every conversation.
	MessagesSince(ctx context.Context, userID string, afterSeq int64, limit int) ([]models.SupportMessage, error)
	MaxSeq(ctx context.Context) (int64, error)
	MarkRead(ctx context.Context, conversationID string, staff bool, seq int64) error
	SetStatus(ctx context.Context, conversationID, status, actor string, now time.Time) (*models.SupportConversation, error)
	UnreadForStaff(ctx context.Context) (int64, error)
	// UnreadStaffMessages counts staff messages the account has not read in
	// one conversation.
	UnreadStaffMessages(ctx context.Context, conversationID string) (int64, error)
}

type supportRepository struct {
	db *gorm.DB
}

func NewSupportRepository(db *gorm.DB) SupportRepository {
	return &supportRepository{db: db}
}

func (r *supportRepository) Eligibility(ctx context.Context, userID string, now time.Time) (SupportEligibility, error) {
	var row struct {
		Trainer bool
		Client  bool
	}
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			EXISTS (SELECT 1 FROM trainers WHERE user_id = @user AND deleted_at IS NULL) AS trainer,
			(EXISTS (SELECT 1 FROM entitlements WHERE user_id = @user AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > @now))
			 OR EXISTS (SELECT 1 FROM free_program_activations WHERE user_id = @user AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > @now))) AS client`,
		map[string]any{"user": userID, "now": now},
	).Scan(&row).Error
	if err != nil {
		return SupportEligibility{}, fmt.Errorf("failed to check support eligibility: %w", err)
	}
	return SupportEligibility{Trainer: row.Trainer, Client: row.Client}, nil
}

// lockMessageOrder serializes message inserts until commit, so sequence order
// equals commit order: a live stream that has read sequence N can never later
// see a message with a smaller sequence commit. Support traffic is low enough
// for one transaction-scoped advisory lock.
func lockMessageOrder(tx *gorm.DB) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", supportMessageLockKey).Error
}

// supportMessageLockKey is the fixed advisory-lock key of support message
// inserts ("RYZESUPP" as an int64).
const supportMessageLockKey int64 = 0x52595a4553555050

// loadSeq reads the database-assigned sequence and time of a new message.
func loadSeq(tx *gorm.DB, message *models.SupportMessage) error {
	return tx.Raw("SELECT seq, created_at FROM support_messages WHERE id = ?", message.ID).
		Row().Scan(&message.Seq, &message.CreatedAt)
}

func (r *supportRepository) CreateConversation(ctx context.Context, conversation *models.SupportConversation, first *models.SupportMessage, maxOpen int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMessageOrder(tx); err != nil {
			return fmt.Errorf("failed to order message: %w", err)
		}
		// Counted under the lock, so parallel requests cannot pass the limit.
		var open int64
		if err := tx.Model(&models.SupportConversation{}).
			Where("user_id = ? AND status = ?", conversation.UserID, models.SupportStatusOpen).
			Count(&open).Error; err != nil {
			return fmt.Errorf("failed to count open conversations: %w", err)
		}
		if open >= maxOpen {
			return ErrSupportTooManyOpen
		}
		if err := tx.Create(conversation).Error; err != nil {
			return fmt.Errorf("failed to create conversation: %w", err)
		}
		first.ConversationID = conversation.ID
		if err := tx.Omit("Seq").Create(first).Error; err != nil {
			return fmt.Errorf("failed to create first message: %w", err)
		}
		if err := loadSeq(tx, first); err != nil {
			return fmt.Errorf("failed to read message sequence: %w", err)
		}
		// The author has read their own message.
		conversation.UserLastReadSeq = first.Seq
		return tx.Model(&models.SupportConversation{}).Where("id = ?", conversation.ID).
			Updates(map[string]any{"user_last_read_seq": first.Seq, "last_message_at": first.CreatedAt}).Error
	})
}

func (r *supportRepository) FindConversation(ctx context.Context, id string) (*models.SupportConversation, error) {
	var conversation models.SupportConversation
	err := r.db.WithContext(ctx).Take(&conversation, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSupportConversationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation: %w", err)
	}
	return &conversation, nil
}

const summarySelect = `c.*,
	u.first_name, u.last_name, u.email,
	COALESCE((SELECT MAX(m.seq) FROM support_messages m WHERE m.conversation_id = c.id), 0) AS last_message_seq`

func (r *supportRepository) ListForUser(ctx context.Context, userID string) ([]SupportConversationSummary, error) {
	var rows []SupportConversationSummary
	err := r.db.WithContext(ctx).Table("support_conversations AS c").
		Select(summarySelect+`,
			(SELECT COUNT(*) FROM support_messages m WHERE m.conversation_id = c.id AND m.author_kind = 'staff' AND m.seq > c.user_last_read_seq) AS unread`).
		Joins("JOIN users u ON u.id = c.user_id").
		Where("c.user_id = ?", userID).
		Order("c.last_message_at DESC, c.id DESC").
		Limit(100).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	return rows, nil
}

func (r *supportRepository) ListForStaff(ctx context.Context, status string, page, limit int) ([]SupportConversationSummary, int64, error) {
	base := r.db.WithContext(ctx).Table("support_conversations AS c").Joins("JOIN users u ON u.id = c.user_id")
	if status != "" {
		base = base.Where("c.status = ?", status)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count conversations: %w", err)
	}
	var rows []SupportConversationSummary
	err := base.Session(&gorm.Session{}).
		Select(summarySelect + `,
			(SELECT COUNT(*) FROM support_messages m WHERE m.conversation_id = c.id AND m.author_kind = 'user' AND m.seq > c.staff_last_read_seq) AS unread`).
		Order("c.last_message_at DESC, c.id DESC").
		Offset((page - 1) * limit).Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conversations: %w", err)
	}
	return rows, total, nil
}

func (r *supportRepository) AddMessage(ctx context.Context, message *models.SupportMessage) (bool, error) {
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMessageOrder(tx); err != nil {
			return fmt.Errorf("failed to order message: %w", err)
		}
		var conversation models.SupportConversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&conversation, "id = ?", message.ConversationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrSupportConversationNotFound
			}
			return fmt.Errorf("failed to lock conversation: %w", err)
		}
		if message.ClientMessageID != nil {
			var existing models.SupportMessage
			err := tx.Where("conversation_id = ? AND client_message_id = ?", message.ConversationID, *message.ClientMessageID).Take(&existing).Error
			if err == nil {
				// Only the same author's resend is recognised; another
				// author's id is refused rather than echoed back.
				if existing.AuthorKind != message.AuthorKind {
					return ErrSupportClientIDTaken
				}
				*message = existing
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("failed to check message: %w", err)
			}
		}
		if message.AuthorKind == models.SupportAuthorUser && conversation.Status != models.SupportStatusOpen {
			return ErrSupportConversationClosed
		}
		if err := tx.Omit("Seq").Create(message).Error; err != nil {
			return fmt.Errorf("failed to add message: %w", err)
		}
		if err := loadSeq(tx, message); err != nil {
			return fmt.Errorf("failed to read message sequence: %w", err)
		}
		updates := map[string]any{"last_message_at": message.CreatedAt}
		// The author has read up to their own message.
		if message.AuthorKind == models.SupportAuthorStaff {
			updates["staff_last_read_seq"] = message.Seq
		} else {
			updates["user_last_read_seq"] = message.Seq
		}
		if err := tx.Model(&models.SupportConversation{}).Where("id = ?", conversation.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update conversation: %w", err)
		}
		created = true
		return nil
	})
	return created, err
}

func (r *supportRepository) Messages(ctx context.Context, conversationID string, beforeSeq int64, limit int) ([]models.SupportMessage, error) {
	query := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID)
	if beforeSeq > 0 {
		query = query.Where("seq < ?", beforeSeq)
	}
	var messages []models.SupportMessage
	if err := query.Order("seq DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	// Newest page first from the database, oldest first for the reader.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (r *supportRepository) MessagesSince(ctx context.Context, userID string, afterSeq int64, limit int) ([]models.SupportMessage, error) {
	query := r.db.WithContext(ctx).Table("support_messages AS m").Select("m.*").Where("m.seq > ?", afterSeq)
	if userID != "" {
		query = query.Joins("JOIN support_conversations c ON c.id = m.conversation_id").Where("c.user_id = ?", userID)
	}
	var messages []models.SupportMessage
	if err := query.Order("m.seq").Limit(limit).Scan(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to list new messages: %w", err)
	}
	return messages, nil
}

func (r *supportRepository) MaxSeq(ctx context.Context) (int64, error) {
	var seq int64
	if err := r.db.WithContext(ctx).Raw("SELECT COALESCE(MAX(seq), 0) FROM support_messages").Row().Scan(&seq); err != nil {
		return 0, fmt.Errorf("failed to read message sequence: %w", err)
	}
	return seq, nil
}

func (r *supportRepository) MarkRead(ctx context.Context, conversationID string, staff bool, seq int64) error {
	column := "user_last_read_seq"
	if staff {
		column = "staff_last_read_seq"
	}
	// Never move backwards, and never past the conversation's last message.
	result := r.db.WithContext(ctx).Exec(`
		UPDATE support_conversations SET `+column+` = GREATEST(`+column+`, LEAST(?,
			COALESCE((SELECT MAX(seq) FROM support_messages WHERE conversation_id = ?), 0)))
		WHERE id = ?`, seq, conversationID, conversationID)
	if result.Error != nil {
		return fmt.Errorf("failed to mark conversation read: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSupportConversationNotFound
	}
	return nil
}

func (r *supportRepository) SetStatus(ctx context.Context, conversationID, status, actor string, now time.Time) (*models.SupportConversation, error) {
	updates := map[string]any{"status": status, "closed_at": nil, "closed_by": nil}
	if status == models.SupportStatusClosed {
		updates["closed_at"] = now
		updates["closed_by"] = actor
	}
	result := r.db.WithContext(ctx).Model(&models.SupportConversation{}).Where("id = ?", conversationID).Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update conversation status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrSupportConversationNotFound
	}
	return r.FindConversation(ctx, conversationID)
}

func (r *supportRepository) UnreadForStaff(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM support_messages m
		JOIN support_conversations c ON c.id = m.conversation_id
		WHERE c.status = 'open' AND m.author_kind = 'user' AND m.seq > c.staff_last_read_seq`).Row().Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread messages: %w", err)
	}
	return count, nil
}

func (r *supportRepository) UnreadStaffMessages(ctx context.Context, conversationID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) FROM support_messages m
		JOIN support_conversations c ON c.id = m.conversation_id
		WHERE c.id = ? AND m.author_kind = 'staff' AND m.seq > c.user_last_read_seq`, conversationID).Row().Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread replies: %w", err)
	}
	return count, nil
}

func (r *supportRepository) FindOpenedByClientID(ctx context.Context, userID, clientMessageID string) (*models.SupportConversation, error) {
	var conversation models.SupportConversation
	err := r.db.WithContext(ctx).Table("support_conversations AS c").Select("c.*").
		Joins("JOIN support_messages m ON m.conversation_id = c.id").
		Where("c.user_id = ? AND m.author_kind = ? AND m.client_message_id = ?", userID, models.SupportAuthorUser, clientMessageID).
		Order("m.seq").Limit(1).
		Scan(&conversation).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find conversation: %w", err)
	}
	if conversation.ID == "" {
		return nil, ErrSupportConversationNotFound
	}
	return &conversation, nil
}
