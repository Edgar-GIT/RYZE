-- Reverts the 3D demonstration keys. The attached scenes are derived curation
-- data, so dropping the column loses nothing that cannot be re-derived from
-- this migration.
ALTER TABLE exercises
    DROP CONSTRAINT chk_exercises_three_d_key,
    DROP COLUMN three_d_key;
