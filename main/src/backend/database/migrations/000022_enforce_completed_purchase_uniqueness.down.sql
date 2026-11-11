ALTER TABLE purchases
    DROP CONSTRAINT uq_purchases_completed_purchase,
    DROP COLUMN completed_purchase;
