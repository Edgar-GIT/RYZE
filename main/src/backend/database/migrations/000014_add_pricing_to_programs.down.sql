DROP INDEX IF EXISTS idx_programs_currency;

ALTER TABLE programs
    DROP COLUMN currency,
    DROP COLUMN price_minor_units;
