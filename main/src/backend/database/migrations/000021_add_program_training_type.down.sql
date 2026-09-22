-- Reverts the Generic Program marketplace training-type column.
ALTER TABLE programs
    DROP INDEX idx_programs_training_type,
    DROP COLUMN training_type;