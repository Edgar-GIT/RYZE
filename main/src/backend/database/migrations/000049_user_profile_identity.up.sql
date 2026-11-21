-- Profile identity (owner request, 2026-11-20): every account has a unique
-- @username (its public reference, also how administrators find it), an
-- optional display name people see, and an optional profile photo.

ALTER TABLE users ADD COLUMN username VARCHAR(30);
ALTER TABLE users ADD COLUMN display_name VARCHAR(40);
ALTER TABLE users ADD COLUMN avatar_key VARCHAR(80);

-- The base of a generated username: the email's local part, lowercase
-- letters, digits and underscores only, at least 3 characters.
CREATE FUNCTION ryze_username_base(address TEXT) RETURNS TEXT AS $$
DECLARE
    base TEXT;
BEGIN
    base := lower(regexp_replace(split_part(address, '@', 1), '[^a-zA-Z0-9_]', '', 'g'));
    base := regexp_replace(base, '^_+', '');
    IF length(base) < 3 THEN
        base := base || 'ryze';
    END IF;
    RETURN left(base, 24);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- A free username from the base: the base itself, else base.2, base.3, ...
-- (generated bases never contain a dot, so a suffix never meets a base).
CREATE FUNCTION ryze_free_username(base TEXT, own_id VARCHAR) RETURNS TEXT AS $$
DECLARE
    candidate TEXT := base;
    n INTEGER := 1;
BEGIN
    WHILE EXISTS (SELECT 1 FROM users WHERE username = candidate AND deleted_at IS NULL AND id <> own_id) LOOP
        n := n + 1;
        candidate := base || '.' || n;
    END LOOP;
    RETURN candidate;
END;
$$ LANGUAGE plpgsql;

-- Every creation path (registration, Google, administrators, Test Mode)
-- gets a username without choosing one.
CREATE FUNCTION ryze_users_default_username() RETURNS trigger AS $$
BEGIN
    IF NEW.username IS NULL OR NEW.username = '' THEN
        NEW.username := ryze_free_username(ryze_username_base(NEW.email::text), NEW.id);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Existing accounts, oldest first.
DO $$
DECLARE
    account RECORD;
BEGIN
    FOR account IN SELECT id, email FROM users ORDER BY created_at, id LOOP
        UPDATE users SET username = ryze_free_username(ryze_username_base(account.email::text), account.id)
        WHERE id = account.id;
    END LOOP;
END;
$$;

ALTER TABLE users ALTER COLUMN username SET NOT NULL;
ALTER TABLE users
    ADD CONSTRAINT chk_users_username CHECK (username ~ '^[a-z0-9][a-z0-9._]{2,29}$');
ALTER TABLE users
    ADD CONSTRAINT chk_users_display_name CHECK (display_name IS NULL OR length(btrim(display_name)) BETWEEN 1 AND 40);
ALTER TABLE users
    ADD CONSTRAINT chk_users_avatar_key CHECK (avatar_key IS NULL OR avatar_key ~ '^avatars/[0-9a-f-]{36}\.jpg$');
-- One active account per username (soft-deleted accounts keep theirs).
ALTER TABLE users
    ADD COLUMN active_username VARCHAR(30) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN username END) STORED;
ALTER TABLE users ADD CONSTRAINT uq_users_active_username UNIQUE (active_username);

CREATE TRIGGER trg_users_default_username BEFORE INSERT ON users
    FOR EACH ROW EXECUTE FUNCTION ryze_users_default_username();
