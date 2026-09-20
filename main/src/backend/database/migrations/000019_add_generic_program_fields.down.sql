-- Reverts the Generic Program metadata columns and their integrity guards.
ALTER TABLE programs
    DROP INDEX idx_programs_frequency_per_week,
    DROP INDEX idx_programs_duration_weeks,
    DROP INDEX idx_programs_level,
    DROP CHECK chk_programs_duration_weeks,
    DROP CHECK chk_programs_frequency_per_week,
    DROP COLUMN frequency_per_week,
    DROP COLUMN duration_weeks,
    DROP COLUMN level;