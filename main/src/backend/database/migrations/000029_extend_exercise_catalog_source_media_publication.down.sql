DROP INDEX IF EXISTS idx_exercises_owner_trainer_id;
DROP INDEX IF EXISTS idx_exercises_source;
DROP INDEX IF EXISTS idx_exercises_status;
DROP INDEX IF EXISTS idx_exercises_body_part;

ALTER TABLE exercises
    DROP CONSTRAINT fk_exercises_owner_trainer,
    DROP CONSTRAINT chk_exercises_status,
    DROP CONSTRAINT chk_exercises_body_part,
    DROP CONSTRAINT uq_exercises_active_source;

ALTER TABLE exercises
    DROP COLUMN active_source,
    DROP COLUMN owner_trainer_id,
    DROP COLUMN status,
    DROP COLUMN media_attribution,
    DROP COLUMN source_media_id,
    DROP COLUMN animation_url,
    DROP COLUMN instruction_steps,
    DROP COLUMN body_part,
    DROP COLUMN source_id,
    DROP COLUMN source;
