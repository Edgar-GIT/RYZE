-- External sign-in identities (Google Sign-In).
--
-- users.password_set records whether the owner knows a password for the
-- account. Every existing account registered with a password (TRUE). An
-- account created through Google stores an unusable random hash and FALSE
-- until the owner sets a password through the reset link; while FALSE the
-- Google identity cannot be disconnected, so the account never loses its only
-- way to sign in.
ALTER TABLE users
    ADD COLUMN password_set BOOLEAN NOT NULL DEFAULT TRUE;

-- One row per external identity linked to an account. subject is the
-- provider's stable account id (the ID token "sub" claim); it never changes
-- even when the provider email does. A provider account links to one RYZE
-- account, and a RYZE account links at most one account per provider.
-- email is the provider address at the last sign-in, shown to the owner only.
CREATE TABLE user_identities (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    provider VARCHAR(16) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMPTZ,
    CONSTRAINT user_identities_pkey PRIMARY KEY (id),
    CONSTRAINT uq_user_identities_provider_subject UNIQUE (provider, subject),
    CONSTRAINT uq_user_identities_user_provider UNIQUE (user_id, provider),
    CONSTRAINT fk_user_identities_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_user_identities_provider CHECK (provider IN ('google'))
);
