package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ryze/backend/middleware/adminauthcontext"
	"ryze/backend/middleware/authcontext"
	"ryze/backend/models"
	"ryze/backend/repositories"
	"ryze/backend/services/support"
	"ryze/backend/services/token"
)

// Live stream timing. A stream ends after streamLifetime and the browser
// reconnects with Last-Event-ID, which also re-checks the session and support
// eligibility. The database is re-read every streamPoll even without a local
// wake-up (messages written by another API instance) and a comment line keeps
// proxies from closing an idle connection.
const (
	streamLifetime    = 25 * time.Minute
	streamPoll        = 15 * time.Second
	streamWriteWindow = 30 * time.Second
	streamRetryMillis = 3000
)

// SupportHandler exposes the support chat to accounts and to support staff.
type SupportHandler struct {
	support  support.Service
	tokens   token.Service
	sessions SessionVersions
}

func NewSupportHandler(service support.Service) *SupportHandler {
	return &SupportHandler{support: service}
}

// WithSessions lets account streams re-check their session while open, so a
// password reset or account deletion ends them within one poll interval.
func (h *SupportHandler) WithSessions(tokens token.Service, sessions SessionVersions) *SupportHandler {
	h.tokens = tokens
	h.sessions = sessions
	return h
}

type openConversationRequest struct {
	Subject         string `json:"subject"`
	Body            string `json:"body"`
	ClientMessageID string `json:"client_message_id"`
}

type sendMessageRequest struct {
	Body            string `json:"body"`
	ClientMessageID string `json:"client_message_id"`
}

type markReadRequest struct {
	Seq int64 `json:"seq"`
}

type conversationResponse struct {
	models.SupportConversation
	Unread         int64            `json:"unread"`
	LastMessageSeq int64            `json:"last_message_seq"`
	Requester      *requesterDetail `json:"requester,omitempty"`
}

type requesterDetail struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

func newConversationResponse(row repositories.SupportConversationSummary, staff bool) conversationResponse {
	response := conversationResponse{SupportConversation: row.SupportConversation, Unread: row.Unread, LastMessageSeq: row.LastMessageSeq}
	if staff {
		response.Requester = &requesterDetail{FirstName: row.FirstName, LastName: row.LastName, Email: row.Email}
	}
	return response
}

func threadResponse(thread *support.Thread, staff bool) gin.H {
	data := gin.H{
		"conversation": thread.Conversation,
		"messages":     nonNil(thread.Messages),
		"has_older":    thread.HasOlder,
	}
	if staff && thread.Requester != nil {
		data["requester"] = requesterDetail{FirstName: thread.Requester.FirstName, LastName: thread.Requester.LastName, Email: thread.Requester.Email}
	}
	return data
}

func respondSupportError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, support.ErrNotEligible):
		RespondError(c, http.StatusForbidden, "SUPPORT_NOT_AVAILABLE", "Support is available to accounts with an active programme and to trainers.", nil)
	case errors.Is(err, support.ErrNotFound):
		RespondError(c, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation not found.", nil)
	case errors.Is(err, support.ErrClosed):
		RespondError(c, http.StatusConflict, "CONVERSATION_CLOSED", "This conversation is closed. Start a new one.", nil)
	case errors.Is(err, support.ErrTooManyOpen):
		RespondError(c, http.StatusConflict, "TOO_MANY_OPEN_CONVERSATIONS", "You already have 5 open conversations.", nil)
	case errors.Is(err, support.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Check the subject (3 to 120 characters) and the message (1 to 4000 characters).", nil)
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
	}
}

func conversationParam(c *gin.Context) (string, bool) {
	id := c.Param("conversationID")
	if _, err := uuid.Parse(id); err != nil {
		RespondError(c, http.StatusNotFound, "CONVERSATION_NOT_FOUND", "Conversation not found.", nil)
		return "", false
	}
	return id, true
}

func beforeParam(c *gin.Context) int64 {
	value, err := strconv.ParseInt(c.Query("before"), 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func (h *SupportHandler) userID(c *gin.Context) (string, bool) {
	userID, err := authcontext.UserIDFromContext(c)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication required.", nil)
		return "", false
	}
	return userID, true
}

// Eligibility tells the account whether it may use support.
func (h *SupportHandler) Eligibility(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	eligibility, err := h.support.Eligibility(c.Request.Context(), userID)
	if err != nil {
		respondSupportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Support eligibility retrieved.", "data": gin.H{
		"eligible": eligibility.Eligible,
		"role":     eligibility.Role,
	}})
}

// ListMine lists the account's conversations.
func (h *SupportHandler) ListMine(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	rows, err := h.support.ListMine(c.Request.Context(), userID)
	if err != nil {
		respondSupportError(c, err)
		return
	}
	conversations := make([]conversationResponse, 0, len(rows))
	for _, row := range rows {
		conversations = append(conversations, newConversationResponse(row, false))
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Conversations retrieved.", "data": gin.H{"conversations": conversations}})
}

// Open starts a conversation with its first message.
func (h *SupportHandler) Open(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var req openConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	thread, err := h.support.Open(c.Request.Context(), userID, req.Subject, req.Body, req.ClientMessageID)
	if err != nil {
		respondSupportError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Conversation started.", "data": threadResponse(thread, false)})
}

// GetMine returns one of the account's conversations with its latest
// messages (?before=<seq> pages back).
func (h *SupportHandler) GetMine(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	id, ok := conversationParam(c)
	if !ok {
		return
	}
	thread, err := h.support.GetMine(c.Request.Context(), userID, id, beforeParam(c))
	if err != nil {
		respondSupportError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Conversation retrieved.", "data": threadResponse(thread, false)})
}

// SendMine adds the account's message to its conversation.
func (h *SupportHandler) SendMine(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	id, ok := conversationParam(c)
	if !ok {
		return
	}
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	message, err := h.support.SendMine(c.Request.Context(), userID, id, req.Body, req.ClientMessageID)
	if err != nil {
		respondSupportError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Message sent.", "data": message})
}

// MarkMineRead records how far the account has read.
func (h *SupportHandler) MarkMineRead(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	id, ok := conversationParam(c)
	if !ok {
		return
	}
	var req markReadRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Seq < 0 {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	if err := h.support.MarkMineRead(c.Request.Context(), userID, id, req.Seq); err != nil {
		respondSupportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Conversation marked as read.", "data": gin.H{}})
}

// StreamMine is the account's live message stream (Server-Sent Events).
func (h *SupportHandler) StreamMine(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	eligibility, err := h.support.Eligibility(c.Request.Context(), userID)
	if err != nil || !eligibility.Eligible {
		if err == nil {
			err = support.ErrNotEligible
		}
		respondSupportError(c, err)
		return
	}
	h.stream(c, userID, func(ctx context.Context, after int64) ([]models.SupportMessage, error) {
		return h.support.NewMessagesForUser(ctx, userID, after)
	}, func(ctx context.Context) bool {
		// The stream stays open only while the session is current and the
		// account still qualifies for support.
		if h.tokens != nil && h.sessions != nil && currentSessionUser(c, h.tokens, h.sessions) != userID {
			return false
		}
		still, err := h.support.Eligibility(ctx, userID)
		return err == nil && still.Eligible
	})
}

// ListStaff lists conversations for support staff (?status=open|closed).
func (h *SupportHandler) ListStaff(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	list, err := h.support.ListStaff(c.Request.Context(), c.Query("status"), page, limit)
	if err != nil {
		respondSupportError(c, err)
		return
	}
	conversations := make([]conversationResponse, 0, len(list.Conversations))
	for _, row := range list.Conversations {
		conversations = append(conversations, newConversationResponse(row, true))
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Conversations retrieved.", "data": gin.H{
		"conversations": conversations,
		"total":         list.Total,
		"unread":        list.Unread,
	}})
}

// GetStaff returns one conversation for support staff.
func (h *SupportHandler) GetStaff(c *gin.Context) {
	id, ok := conversationParam(c)
	if !ok {
		return
	}
	thread, err := h.support.GetStaff(c.Request.Context(), id, beforeParam(c))
	if err != nil {
		respondSupportError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Conversation retrieved.", "data": threadResponse(thread, true)})
}

func adminID(c *gin.Context) (string, bool) {
	id, err := adminauthcontext.AdminIdentityFromContext(c)
	if err != nil || id == "" {
		RespondError(c, http.StatusUnauthorized, "ADMIN_AUTHENTICATION_REQUIRED", "Admin authentication required.", nil)
		return "", false
	}
	return id, true
}

// Reply adds a support reply.
func (h *SupportHandler) Reply(c *gin.Context) {
	admin, ok := adminID(c)
	if !ok {
		return
	}
	id, ok := conversationParam(c)
	if !ok {
		return
	}
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	message, err := h.support.Reply(c.Request.Context(), admin, id, req.Body, req.ClientMessageID)
	if err != nil {
		respondSupportError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Reply sent.", "data": message})
}

// MarkStaffRead records how far support has read a conversation.
func (h *SupportHandler) MarkStaffRead(c *gin.Context) {
	id, ok := conversationParam(c)
	if !ok {
		return
	}
	var req markReadRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Seq < 0 {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}
	if err := h.support.MarkStaffRead(c.Request.Context(), id, req.Seq); err != nil {
		respondSupportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Conversation marked as read.", "data": gin.H{}})
}

// Close and Reopen change a conversation's status.
func (h *SupportHandler) Close(c *gin.Context)  { h.setStatus(c, models.SupportStatusClosed) }
func (h *SupportHandler) Reopen(c *gin.Context) { h.setStatus(c, models.SupportStatusOpen) }

func (h *SupportHandler) setStatus(c *gin.Context, status string) {
	admin, ok := adminID(c)
	if !ok {
		return
	}
	id, ok := conversationParam(c)
	if !ok {
		return
	}
	conversation, err := h.support.SetStatus(c.Request.Context(), admin, id, status)
	if err != nil {
		respondSupportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Conversation updated.", "data": conversation})
}

// StreamStaff is the live stream of every conversation for support staff.
func (h *SupportHandler) StreamStaff(c *gin.Context) {
	h.stream(c, support.StaffScope, func(ctx context.Context, after int64) ([]models.SupportMessage, error) {
		return h.support.NewMessagesForStaff(ctx, after)
	}, nil)
}

// streamStart resolves where a stream resumes: the Last-Event-ID the browser
// sends on reconnection, the ?after= the page passes on first connection, or
// the current sequence (only new messages).
func (h *SupportHandler) streamStart(c *gin.Context) (int64, error) {
	for _, raw := range []string{c.GetHeader("Last-Event-ID"), c.Query("after")} {
		if raw == "" {
			continue
		}
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil && value >= 0 {
			return value, nil
		}
	}
	return h.support.CurrentSeq(c.Request.Context())
}

// stream writes Server-Sent Events: one "message" event per stored message,
// with the message sequence as the event id.
// stillAllowed (optional) is re-checked on every poll; the stream ends when
// it reports false.
func (h *SupportHandler) stream(c *gin.Context, scope string, fetch func(ctx context.Context, after int64) ([]models.SupportMessage, error), stillAllowed func(ctx context.Context) bool) {
	after, err := h.streamStart(c)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.", nil)
		return
	}
	ctx := c.Request.Context()
	wake, release, err := h.support.Hub().Subscribe(scope)
	if err != nil {
		c.Header("Retry-After", "60")
		RespondError(c, http.StatusTooManyRequests, "TOO_MANY_STREAMS", "Too many live connections. Close another tab and try again.", nil)
		return
	}
	defer release()

	writer := c.Writer
	controller := http.NewResponseController(writer)
	header := writer.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-store")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)

	write := func(chunk string) bool {
		// Each write gets its own deadline, so the server-wide write timeout
		// does not cut a healthy long-lived stream.
		_ = controller.SetWriteDeadline(time.Now().Add(streamWriteWindow))
		if _, err := io.WriteString(writer, chunk); err != nil {
			return false
		}
		return controller.Flush() == nil
	}
	send := func() bool {
		for {
			messages, err := fetch(ctx, after)
			if err != nil {
				return false
			}
			if len(messages) == 0 {
				return true
			}
			for _, message := range messages {
				data, err := json.Marshal(message)
				if err != nil {
					return false
				}
				if !write(fmt.Sprintf("id: %d\nevent: message\ndata: %s\n\n", message.Seq, data)) {
					return false
				}
				after = message.Seq
			}
		}
	}

	if !write(fmt.Sprintf("retry: %d\n\n", streamRetryMillis)) || !send() {
		return
	}
	lifetime := time.NewTimer(streamLifetime)
	defer lifetime.Stop()
	poll := time.NewTicker(streamPoll)
	defer poll.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-lifetime.C:
			return
		case <-wake:
			if !send() {
				return
			}
		case <-poll.C:
			if stillAllowed != nil && !stillAllowed(ctx) {
				return
			}
			if !send() || !write(": keep-alive\n\n") {
				return
			}
		}
	}
}
