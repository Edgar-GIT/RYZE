DROP INDEX IF EXISTS idx_users_pending_anonymization;
ALTER TABLE users DROP COLUMN IF EXISTS anonymized_at;
