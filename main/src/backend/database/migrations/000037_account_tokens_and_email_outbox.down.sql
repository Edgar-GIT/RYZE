DROP TABLE IF EXISTS email_outbox;
DROP TABLE IF EXISTS user_tokens;

ALTER TABLE users
    DROP CONSTRAINT chk_users_preferred_language,
    DROP COLUMN preferred_language,
    DROP COLUMN email_verified_at;
