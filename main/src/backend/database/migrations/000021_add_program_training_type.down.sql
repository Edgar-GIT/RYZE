-- Reverts the Generic Program marketplace training-type column.
DROP INDEX IF EXISTS idx_programs_training_type;

ALTER TABLE programs
    DROP COLUMN training_type;
