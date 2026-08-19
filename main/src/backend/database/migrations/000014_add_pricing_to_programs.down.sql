DROP INDEX idx_programs_currency ON programs;

ALTER TABLE programs
DROP COLUMN currency,
DROP COLUMN price_minor_units;
