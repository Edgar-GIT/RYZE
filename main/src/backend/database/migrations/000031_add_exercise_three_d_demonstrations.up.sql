-- Three-dimensional demonstration keys (v2.94.0).
--
-- Marks the platform exercises that ship a procedural 3D demonstration in the
-- frontend viewer. The key selects the scene and animation the viewer
-- implements; the check constraint keeps the column limited to the scenes the
-- product actually ships, so a stored value can never surface a 3D entry that
-- does not render. NULL means the exercise has no 3D demonstration and the
-- regular media experience applies.
--
-- The initial scenes attach to the four canonical exercises: Push-Up
-- (source 0662) and Walking Lunge (source 1460) exist as enriched starter
-- catalogue rows; dumbbell biceps curl (source 0294) and barbell full squat
-- (source 0043) are imported dataset rows. Source identity, not row id, keeps
-- the attachment stable across catalogues.
ALTER TABLE exercises
    ADD COLUMN three_d_key VARCHAR(64) NULL DEFAULT NULL AFTER animation_url,
    ADD CONSTRAINT chk_exercises_three_d_key CHECK (three_d_key IS NULL OR three_d_key IN ('squat', 'push_up', 'biceps_curl', 'lunge'));

UPDATE exercises SET three_d_key = 'squat' WHERE source = 'gymvisual' AND source_id = '0043';
UPDATE exercises SET three_d_key = 'push_up' WHERE source = 'gymvisual' AND source_id = '0662';
UPDATE exercises SET three_d_key = 'biceps_curl' WHERE source = 'gymvisual' AND source_id = '0294';
UPDATE exercises SET three_d_key = 'lunge' WHERE source = 'gymvisual' AND source_id = '1460';
