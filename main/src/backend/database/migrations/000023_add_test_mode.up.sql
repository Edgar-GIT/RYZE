-- test_sessions records active ADMIN_1 Test Mode sessions. A session binds an
-- opaque bearer token to a predefined persona and preserves the original admin
-- identity so exiting Test Mode always restores ADMIN_1. Only the SHA-256 of
-- the raw token is stored, so a database leak never exposes the session cookie
-- value.
CREATE TABLE IF NOT EXISTS test_sessions (
    id CHAR(36) PRIMARY KEY,
    admin_identity VARCHAR(16) NOT NULL,
    persona VARCHAR(16) NOT NULL,
    persona_user_id VARCHAR(36) NOT NULL,
    token_hash CHAR(64) NOT NULL,
    return_path VARCHAR(512) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    CONSTRAINT fk_test_sessions_persona_user FOREIGN KEY (persona_user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE UNIQUE INDEX uq_test_sessions_token_hash ON test_sessions (token_hash);
CREATE INDEX idx_test_sessions_persona_user ON test_sessions (persona_user_id);

-- Test Mode purchases are REAL purchases (they create real entitlements) but
-- they never contact a payment provider and carry a zero price. The test
-- marker keeps them distinguishable from paid sales and from every commerc
-- path so they can never be mistaken for revenue or a trainer commission.
ALTER TABLE purchases
    ADD COLUMN test BOOLEAN NOT NULL DEFAULT FALSE AFTER status;