ALTER TABLE users
    ADD COLUMN session_version INT NOT NULL DEFAULT 0 AFTER updated_at;
