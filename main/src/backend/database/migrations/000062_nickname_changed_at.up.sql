-- When the nickname (display_name) last changed: it can change once every 7
-- days (owner decision, 2026-11-23). NULL: never changed since this migration.
ALTER TABLE users ADD COLUMN display_name_changed_at TIMESTAMPTZ NULL;
