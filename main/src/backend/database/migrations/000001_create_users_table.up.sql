-- PostgreSQL schema (converted from the original MariaDB migrations).
--
-- Shared database objects used by every later migration:
--   * citext gives users.email the case-insensitive comparison and uniqueness
--     the MariaDB utf8mb4_unicode_ci collation provided.
--   * ryze_touch_updated_at() replaces MariaDB's
--     "ON UPDATE CURRENT_TIMESTAMP(6)": an UPDATE that does not set updated_at
--     itself gets the current time.
CREATE EXTENSION IF NOT EXISTS citext;

CREATE OR REPLACE FUNCTION ryze_touch_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.updated_at IS NOT DISTINCT FROM OLD.updated_at THEN
        NEW.updated_at := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TABLE users (
    id VARCHAR(36) NOT NULL,
    email CITEXT NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_email CITEXT GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN email END) STORED,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT uq_users_active_email UNIQUE (active_email),
    CONSTRAINT chk_users_email_length CHECK (char_length(email) <= 255)
);

CREATE TRIGGER trg_users_touch_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
