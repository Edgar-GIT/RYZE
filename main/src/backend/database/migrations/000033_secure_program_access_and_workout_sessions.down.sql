DROP TABLE IF EXISTS workout_session_sets;
DROP TABLE IF EXISTS workout_session_exercises;
DROP TABLE IF EXISTS workout_sessions;

ALTER TABLE entitlements
    DROP INDEX idx_entitlements_expires_at,
    DROP COLUMN expires_at;

DROP TABLE IF EXISTS free_program_activations;
