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
    ADD COLUMN source VARCHAR(40),
    ADD COLUMN source_id VARCHAR(40),
    ADD COLUMN body_part VARCHAR(40),
    ADD COLUMN instruction_steps JSONB,
    ADD COLUMN animation_url VARCHAR(500),
    ADD COLUMN source_media_id VARCHAR(40),
    ADD COLUMN media_attribution VARCHAR(255),
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'published',
    ADD COLUMN owner_trainer_id VARCHAR(36),
    ADD COLUMN active_source VARCHAR(90) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL AND source_id IS NOT NULL THEN source || ':' || source_id END
    ) STORED,
    ADD CONSTRAINT uq_exercises_active_source UNIQUE (active_source),
    ADD CONSTRAINT fk_exercises_owner_trainer FOREIGN KEY (owner_trainer_id) REFERENCES trainers (id),
    ADD CONSTRAINT chk_exercises_status CHECK (status IN ('draft', 'published', 'archived')),
    ADD CONSTRAINT chk_exercises_body_part CHECK (body_part IS NULL OR body_part IN ('back', 'cardio', 'chest', 'lower arms', 'lower legs', 'neck', 'shoulders', 'upper arms', 'upper legs', 'waist'));

CREATE INDEX idx_exercises_body_part ON exercises (body_part);
CREATE INDEX idx_exercises_status ON exercises (status);
CREATE INDEX idx_exercises_source ON exercises (source, source_id);
CREATE INDEX idx_exercises_owner_trainer_id ON exercises (owner_trainer_id);
