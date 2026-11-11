ALTER TABLE programs
    ADD COLUMN price_minor_units BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'EUR';

CREATE INDEX idx_programs_currency ON programs (currency);
