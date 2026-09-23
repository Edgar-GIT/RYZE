ALTER TABLE purchases
    DROP INDEX uq_purchases_completed_purchase,
    DROP COLUMN completed_purchase;