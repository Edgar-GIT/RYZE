-- Account recovery, email verification and transactional email.
--
-- users.email_verified_at records when the owner proved control of the address
-- (NULL = not verified yet; existing accounts stay unverified until they use a
-- verification link). Verification is informational: it does not change who
-- may sign in. users.preferred_language is the language of the emails RYZE
-- sends; NULL falls back to English.
ALTER TABLE users
    ADD COLUMN email_verified_at TIMESTAMPTZ,
    ADD COLUMN preferred_language VARCHAR(5),
    ADD CONSTRAINT chk_users_preferred_language CHECK (
        preferred_language IS NULL OR preferred_language IN ('en', 'pt')
    );

-- Single-use account tokens. Only the SHA-256 of the random token is stored,
-- so a database leak never exposes a usable link. A token is consumed by
-- setting used_at; it is never deleted, so the issuing history stays auditable.
CREATE TABLE user_tokens (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    purpose VARCHAR(32) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT uq_user_tokens_token_hash UNIQUE (token_hash),
    CONSTRAINT fk_user_tokens_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_user_tokens_purpose CHECK (purpose IN ('password_reset', 'email_verification')),
    CONSTRAINT chk_user_tokens_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_user_tokens_user_purpose ON user_tokens (user_id, purpose, created_at);

-- Transactional email outbox. A business event records the email here and a
-- background worker delivers it with retries, so a slow or failing relay never
-- blocks or fails the request, and the delivery state is observable. The
-- dedupe_key makes a repeated event (for example a webhook redelivery) record
-- the email only once. payload holds only the values the template needs; it is
-- cleared once the email is sent, fails permanently or expires, so a link that
-- carries a token never outlives its delivery.
CREATE TABLE email_outbox (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36),
    recipient VARCHAR(255) NOT NULL,
    template VARCHAR(48) NOT NULL,
    locale VARCHAR(5) NOT NULL DEFAULT 'en',
    payload JSONB NOT NULL DEFAULT '{}',
    dedupe_key VARCHAR(160),
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ,
    last_error VARCHAR(255),
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT email_outbox_pkey PRIMARY KEY (id),
    CONSTRAINT uq_email_outbox_dedupe_key UNIQUE (dedupe_key),
    CONSTRAINT fk_email_outbox_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_email_outbox_status CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'skipped', 'expired')),
    CONSTRAINT chk_email_outbox_locale CHECK (locale IN ('en', 'pt')),
    CONSTRAINT chk_email_outbox_attempts CHECK (attempts >= 0)
);

CREATE INDEX idx_email_outbox_due ON email_outbox (status, next_attempt_at);
CREATE INDEX idx_email_outbox_user_id ON email_outbox (user_id);

CREATE TRIGGER trg_email_outbox_touch_updated_at BEFORE UPDATE ON email_outbox
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
