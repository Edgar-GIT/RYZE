ALTER TABLE purchases
    ADD COLUMN price_minor_units BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    ADD COLUMN commission_bps INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN platform_amount BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN trainer_amount BIGINT NOT NULL DEFAULT 0,
    ADD CONSTRAINT chk_purchases_commission_bps CHECK (commission_bps >= 0);

UPDATE purchases SET status = 'pending' WHERE status = 'completed';

ALTER TABLE purchases
    ALTER COLUMN status SET DEFAULT 'pending';
