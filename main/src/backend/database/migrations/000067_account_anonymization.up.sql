-- Accounts deleted more than 30 days ago are anonymised (GDPR art. 17;
-- owner request, 2026-11-24). Until then a re-registration with the same
-- e-mail restores the account. Sales documents (invoices, purchases) stay for
-- the legal retention period; everything that identifies the person is erased.
ALTER TABLE users ADD COLUMN anonymized_at TIMESTAMPTZ NULL;
CREATE INDEX idx_users_pending_anonymization ON users (deleted_at) WHERE deleted_at IS NOT NULL AND anonymized_at IS NULL;
