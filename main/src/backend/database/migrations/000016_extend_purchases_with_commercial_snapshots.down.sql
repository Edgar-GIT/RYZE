ALTER TABLE purchases
    DROP CONSTRAINT chk_purchases_commission_bps,
    DROP COLUMN price_minor_units,
    DROP COLUMN currency,
    DROP COLUMN commission_bps,
    DROP COLUMN platform_amount,
    DROP COLUMN trainer_amount;

ALTER TABLE purchases
    ALTER COLUMN status SET DEFAULT 'completed';

UPDATE purchases SET status = 'completed' WHERE status = 'pending';
