-- Refunded purchases go back to failed (no access), so the old status check holds.
UPDATE purchases SET status = 'failed' WHERE status = 'refunded';
ALTER TABLE purchases DROP CONSTRAINT IF EXISTS chk_purchases_refund_state;
ALTER TABLE purchases DROP CONSTRAINT IF EXISTS chk_purchases_refund_reason;
ALTER TABLE purchases
    DROP COLUMN IF EXISTS refund_provider_ref,
    DROP COLUMN IF EXISTS refunded_by,
    DROP COLUMN IF EXISTS refund_note,
    DROP COLUMN IF EXISTS refund_reason,
    DROP COLUMN IF EXISTS refunded_at;
ALTER TABLE purchases DROP CONSTRAINT chk_purchases_status;
ALTER TABLE purchases ADD CONSTRAINT chk_purchases_status CHECK (status IN ('pending', 'completed', 'failed'));
