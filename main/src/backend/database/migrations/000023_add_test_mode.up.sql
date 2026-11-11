-- test_sessions records active ADMIN_1 Test Mode sessions. A session binds an
-- opaque bearer token to a predefined persona and preserves the original admin
-- identity so exiting Test Mode always restores ADMIN_1. Only the SHA-256 of
-- the raw token is stored, so a database leak never exposes the session cookie
-- value.
CREATE TABLE IF NOT EXISTS test_sessions (
    id VARCHAR(36) PRIMARY KEY,
    admin_identity VARCHAR(16) NOT NULL,
    persona VARCHAR(16) NOT NULL,
    persona_user_id VARCHAR(36) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    return_path VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_test_sessions_persona_user FOREIGN KEY (persona_user_id) REFERENCES users (id)
);

CREATE UNIQUE INDEX uq_test_sessions_token_hash ON test_sessions (token_hash);
CREATE INDEX idx_test_sessions_persona_user ON test_sessions (persona_user_id);

CREATE TRIGGER trg_test_sessions_touch_updated_at BEFORE UPDATE ON test_sessions
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

-- Test Mode purchases are REAL purchases (they create real entitlements) but
-- they never contact a payment provider and carry a zero price. The test
-- marker keeps them distinguishable from paid sales and from every commerc
-- path so they can never be mistaken for revenue or a trainer commission.
ALTER TABLE purchases
    ADD COLUMN test BOOLEAN NOT NULL DEFAULT FALSE;
