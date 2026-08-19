ALTER TABLE programs
ADD COLUMN price_minor_units BIGINT NOT NULL DEFAULT 0 AFTER status,
ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'EUR' AFTER price_minor_units;

CREATE INDEX idx_programs_currency ON programs (currency);
