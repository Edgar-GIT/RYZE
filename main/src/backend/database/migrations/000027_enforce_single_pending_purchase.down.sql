ALTER TABLE purchases
    DROP INDEX uq_purchases_pending_purchase,
    DROP COLUMN pending_purchase;