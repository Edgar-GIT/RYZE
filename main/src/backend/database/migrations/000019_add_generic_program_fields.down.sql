-- Reverts the Generic Program metadata columns and their integrity guards.
DROP INDEX IF EXISTS idx_programs_frequency_per_week;
DROP INDEX IF EXISTS idx_programs_duration_weeks;
DROP INDEX IF EXISTS idx_programs_level;

ALTER TABLE programs
    DROP CONSTRAINT chk_programs_duration_weeks,
    DROP CONSTRAINT chk_programs_frequency_per_week,
    DROP COLUMN frequency_per_week,
    DROP COLUMN duration_weeks,
    DROP COLUMN level;
