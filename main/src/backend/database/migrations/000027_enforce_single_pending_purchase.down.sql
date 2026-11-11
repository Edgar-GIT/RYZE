ALTER TABLE purchases
    DROP CONSTRAINT uq_purchases_pending_purchase,
    DROP COLUMN pending_purchase;
