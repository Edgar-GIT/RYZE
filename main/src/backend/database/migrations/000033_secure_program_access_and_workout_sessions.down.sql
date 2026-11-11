DROP TABLE IF EXISTS workout_session_sets;
DROP TABLE IF EXISTS workout_session_exercises;
DROP TABLE IF EXISTS workout_sessions;

DROP INDEX IF EXISTS idx_entitlements_expires_at;

ALTER TABLE entitlements
    DROP COLUMN expires_at;

DROP TABLE IF EXISTS free_program_activations;
