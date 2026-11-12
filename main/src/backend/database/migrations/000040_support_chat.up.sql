-- Support chat between an account and RYZE support (answered by the
-- Management Administrator, ADMIN_2).
--
-- Support is not coaching: a conversation never links a client to a trainer
-- and creates no coaching relationship. Eligible accounts are trainers and
-- clients with an active entitlement or Free activation; the API checks this
-- on every operation.
--
-- Conversations and messages are the permanent support history (they are not
-- the 24-hour notifications). Messages are append-only.
CREATE TABLE support_conversations (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    requester_role VARCHAR(16) NOT NULL,
    subject VARCHAR(120) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'open',
    -- Highest message sequence each side has read; unread counts compare
    -- message sequences with these.
    user_last_read_seq BIGINT NOT NULL DEFAULT 0,
    staff_last_read_seq BIGINT NOT NULL DEFAULT 0,
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMPTZ,
    closed_by VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT support_conversations_pkey PRIMARY KEY (id),
    CONSTRAINT fk_support_conversations_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_support_conversations_role CHECK (requester_role IN ('client', 'trainer')),
    CONSTRAINT chk_support_conversations_status CHECK (status IN ('open', 'closed')),
    CONSTRAINT chk_support_conversations_subject CHECK (char_length(subject) BETWEEN 3 AND 120),
    CONSTRAINT chk_support_conversations_closed CHECK ((status = 'closed') = (closed_at IS NOT NULL))
);

CREATE INDEX idx_support_conversations_user ON support_conversations (user_id, last_message_at DESC);
CREATE INDEX idx_support_conversations_status ON support_conversations (status, last_message_at DESC);

CREATE TRIGGER trg_support_conversations_touch_updated_at BEFORE UPDATE ON support_conversations
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

-- seq is a global, increasing message number: live streams resume from the
-- last sequence a client saw (Server-Sent Events Last-Event-ID), so no message
-- is lost across a reconnection. client_message_id lets a client resend after
-- a dropped connection without creating a duplicate.
CREATE TABLE support_messages (
    id VARCHAR(36) NOT NULL,
    seq BIGINT GENERATED ALWAYS AS IDENTITY,
    conversation_id VARCHAR(36) NOT NULL,
    author_kind VARCHAR(8) NOT NULL,
    author_user_id VARCHAR(36),
    author_admin_id VARCHAR(64),
    body TEXT NOT NULL,
    client_message_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT support_messages_pkey PRIMARY KEY (id),
    CONSTRAINT uq_support_messages_seq UNIQUE (seq),
    CONSTRAINT uq_support_messages_client_id UNIQUE (conversation_id, client_message_id),
    CONSTRAINT fk_support_messages_conversation FOREIGN KEY (conversation_id) REFERENCES support_conversations (id),
    CONSTRAINT fk_support_messages_user FOREIGN KEY (author_user_id) REFERENCES users (id),
    CONSTRAINT chk_support_messages_author CHECK (
        (author_kind = 'user' AND author_user_id IS NOT NULL AND author_admin_id IS NULL)
        OR (author_kind = 'staff' AND author_admin_id IS NOT NULL AND author_user_id IS NULL)
    ),
    CONSTRAINT chk_support_messages_body CHECK (char_length(body) BETWEEN 1 AND 4000)
);

CREATE INDEX idx_support_messages_conversation_seq ON support_messages (conversation_id, seq);
