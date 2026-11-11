-- RYZE purchases are final: a user can complete at most one purchase per
-- program. The unique constraint is scoped to active COMPLETED purchases
-- only, mirroring the active_entitlement pattern. A pending purchase in
-- progress still blocks a second purchase through the service layer, and a
-- failed purchase must remain retryable, so only the definitive terminal
-- state (completed) is locked at the database level.
ALTER TABLE purchases
    ADD COLUMN completed_purchase VARCHAR(73) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL AND status = 'completed' THEN user_id || ':' || program_id END
    ) STORED,
    ADD CONSTRAINT uq_purchases_completed_purchase UNIQUE (completed_purchase);
