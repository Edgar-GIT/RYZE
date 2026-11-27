-- Refunds (owner request, 2026-11-24): a refund is required when the digital
-- content is not in conformity, and when the right of withdrawal applies
-- (Premium Level 2). ADMIN_2 refunds a completed purchase: the purchase
-- becomes 'refunded', its access is removed at once and it no longer counts
-- as a sale (so it leaves the trainer's earnings).
ALTER TABLE purchases DROP CONSTRAINT chk_purchases_status;
ALTER TABLE purchases ADD CONSTRAINT chk_purchases_status CHECK (status IN ('pending', 'completed', 'failed', 'refunded'));
ALTER TABLE purchases
    ADD COLUMN refunded_at TIMESTAMPTZ NULL,
    ADD COLUMN refund_reason VARCHAR(32) NULL,
    ADD COLUMN refund_note VARCHAR(500) NULL,
    ADD COLUMN refunded_by VARCHAR(36) NULL,
    ADD COLUMN refund_provider_ref VARCHAR(255) NULL;
ALTER TABLE purchases ADD CONSTRAINT chk_purchases_refund_reason
    CHECK (refund_reason IS NULL OR refund_reason IN ('non_conformity', 'withdrawal', 'duplicate', 'goodwill'));
ALTER TABLE purchases ADD CONSTRAINT chk_purchases_refund_state
    CHECK ((status = 'refunded') = (refunded_at IS NOT NULL AND refund_reason IS NOT NULL));
