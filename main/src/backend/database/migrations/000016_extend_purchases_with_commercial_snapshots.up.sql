ALTER TABLE purchases
    ADD COLUMN price_minor_units BIGINT NOT NULL DEFAULT 0 AFTER program_id,
    ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'EUR' AFTER price_minor_units,
    ADD COLUMN commission_bps INT UNSIGNED NOT NULL DEFAULT 0 AFTER currency,
    ADD COLUMN platform_amount BIGINT NOT NULL DEFAULT 0 AFTER commission_bps,
    ADD COLUMN trainer_amount BIGINT NOT NULL DEFAULT 0 AFTER platform_amount;

UPDATE purchases SET status = 'pending' WHERE status = 'completed';

ALTER TABLE purchases
    ALTER COLUMN status SET DEFAULT 'pending';
