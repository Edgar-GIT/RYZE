ALTER TABLE workout_sessions DROP CONSTRAINT IF EXISTS chk_workout_sessions_photo_key;
ALTER TABLE workout_sessions DROP COLUMN IF EXISTS photo_key;
