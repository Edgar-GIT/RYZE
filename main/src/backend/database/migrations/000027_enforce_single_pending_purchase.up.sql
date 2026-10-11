-- RYZE allows at most one active payable purchase per program, so a buyer can
-- never hold two live intents for the same product. The pending state is the
-- only intermediate in which a provider payment can be opened (initiation), so
-- the unique constraint is scoped to active PENDING purchases only, mirroring
-- the completed_purchase pattern. A failed purchase must remain replaceable --
-- a fresh intent supersedes it and reconciliation may complete it later -- and
-- a completed purchase is already locked by uq_purchases_completed_purchase,
-- so only the volatile in-progress state is constrained here. The service
-- layer still runs the duplicate-intent check first; this index closes the
-- concurrent-writer race that check cannot see, making the creation of a
-- second live intent a deterministic database-level rejection instead of a
-- lost update.
ALTER TABLE purchases
    ADD COLUMN pending_purchase VARCHAR(73) GENERATED ALWAYS AS (
        IF(deleted_at IS NULL AND status = 'pending', CONCAT(user_id, ':', program_id), NULL)
    ) STORED,
    ADD UNIQUE KEY uq_purchases_pending_purchase (pending_purchase);