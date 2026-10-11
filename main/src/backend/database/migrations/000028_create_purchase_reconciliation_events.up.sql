-- Purchase reconciliation events make the webhook settlement verdicts durable.
--
-- Before this table, a verified late settlement that could not grant a sale
-- (a pair already resolved, the purchase moving on concurrently, an entitlement
-- integrity violation) left only an unstructured log line plus the failed
-- purchase row: the purchase record alone cannot tell an operator whether the
-- payment was reconciled, conflicted, mismatched or silently retried. This
-- table records one row per provider delivery containing only the minimum safe
-- diagnostic data needed to investigate an unresolved verified settlement.
--
-- Safety rules enforced here:
--   - The row is written only by the trusted webhook layer, never by a client
--     route, so no unauthorised caller can attach or read outcomes.
--   - Only the purchase id, provider, provider event id, an outcome code and
--     the first-seen timestamp are stored. No signatures, secrets, tokens,
--     provider payloads, questionnaire answers or nutrition data are persisted.
--   - UNIQUE (provider, event_id) makes a provider retry idempotent at the
--     audit layer: one delivery produces at most one durable record, and the
--     repository upserts the provisional outcome (e.g. transient_failure) with
--     the retry's final verdict instead of creating a duplicate row.
--   - The outcome vocabulary is a fixed set of documented codes; it is
--     deliberately not an open-ended free-text field.
--
-- The row is append-only and never soft-deleted: audit records must survive
-- exactly as written, so the soft-delete convention intentionally does not
-- apply here.
CREATE TABLE purchase_reconciliation_events (
    id CHAR(36) NOT NULL,
    purchase_id CHAR(36) NOT NULL,
    provider VARCHAR(16) NOT NULL,
    event_id VARCHAR(128) NOT NULL,
    outcome VARCHAR(32) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_purchase_reconciliation_events_delivery (provider, event_id),
    KEY idx_purchase_reconciliation_events_purchase (purchase_id, created_at),
    CONSTRAINT fk_purchase_reconciliation_events_purchase
        FOREIGN KEY (purchase_id) REFERENCES purchases (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;