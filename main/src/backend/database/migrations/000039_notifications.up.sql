-- In-app notifications.
--
-- A notification is temporary: it is visible for 24 hours (expires_at =
-- created_at + 24 hours) and is never the record of anything. Purchases,
-- entitlements, Premium Level 2 events and audit data keep their own
-- permanent tables; nothing references this one.
--
-- kind names the message; params holds only display values (for example a
-- programme name), never answers, health data or payment details. The web app
-- writes the text in the reader's language from kind + params. link is a
-- relative path inside the app.
--
-- Retention: once expired a notification is hidden; the hourly job then
-- clears its params and link, keeping only the dedupe_key, and deletes the row
-- 30 days after expiry. The kept key stops a repeated event (a payment
-- provider redelivering a webhook days later) from notifying twice.
CREATE TABLE notifications (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    kind VARCHAR(48) NOT NULL,
    params JSONB NOT NULL DEFAULT '{}',
    link VARCHAR(255),
    dedupe_key VARCHAR(160),
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT notifications_pkey PRIMARY KEY (id),
    CONSTRAINT uq_notifications_dedupe_key UNIQUE (dedupe_key),
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_notifications_expiry CHECK (expires_at > created_at),
    CONSTRAINT chk_notifications_link CHECK (link IS NULL OR (link LIKE '/%' AND link NOT LIKE '//%'))
);

CREATE INDEX idx_notifications_user_expires ON notifications (user_id, expires_at);
CREATE INDEX idx_notifications_expires ON notifications (expires_at);
