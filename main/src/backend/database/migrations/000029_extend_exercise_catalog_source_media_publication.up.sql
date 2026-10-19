-- Canonical exercise catalogue (v2.94.0).
--
-- Extends the global platform exercise library with source identity (so the
-- imported dataset can be re-imported idempotently), body-part
-- classification, instruction steps, animation and attribution metadata, and a
-- publication lifecycle. status defaults to 'published' so existing catalog
-- rows and program references keep their current visibility. owner_trainer_id
-- reserves the separation between platform-owned content and future
-- trainer-owned content; only platform-owned content is exposed in this
-- version.
ALTER TABLE exercises
    ADD COLUMN source VARCHAR(40) NULL DEFAULT NULL AFTER name,
    ADD COLUMN source_id VARCHAR(40) NULL DEFAULT NULL AFTER source,
    ADD COLUMN body_part VARCHAR(40) NULL DEFAULT NULL AFTER source_id,
    ADD COLUMN instruction_steps JSON NULL DEFAULT NULL AFTER instructions,
    ADD COLUMN animation_url VARCHAR(500) NULL DEFAULT NULL AFTER image_url,
    ADD COLUMN source_media_id VARCHAR(40) NULL DEFAULT NULL AFTER animation_url,
    ADD COLUMN media_attribution VARCHAR(255) NULL DEFAULT NULL AFTER source_media_id,
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'published' AFTER media_attribution,
    ADD COLUMN owner_trainer_id VARCHAR(36) NULL DEFAULT NULL AFTER status,
    ADD COLUMN active_source VARCHAR(90) GENERATED ALWAYS AS (IF(deleted_at IS NULL AND source_id IS NOT NULL, CONCAT(source, ':', source_id), NULL)) STORED,
    ADD INDEX idx_exercises_body_part (body_part),
    ADD INDEX idx_exercises_status (status),
    ADD INDEX idx_exercises_source (source, source_id),
    ADD INDEX idx_exercises_owner_trainer_id (owner_trainer_id),
    ADD UNIQUE INDEX uq_exercises_active_source (active_source),
    ADD CONSTRAINT fk_exercises_owner_trainer FOREIGN KEY (owner_trainer_id) REFERENCES trainers (id),
    ADD CONSTRAINT chk_exercises_status CHECK (status IN ('draft', 'published', 'archived')),
    ADD CONSTRAINT chk_exercises_body_part CHECK (body_part IS NULL OR body_part IN ('back', 'cardio', 'chest', 'lower arms', 'lower legs', 'neck', 'shoulders', 'upper arms', 'upper legs', 'waist'));
