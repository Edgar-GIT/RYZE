-- A photo of the member on a workout (owner request, 2026-11-22, as in
-- Lyfta): added before saving the workout and shown when the workout is
-- opened from the history. Only the storage key is kept; the file lives in the
-- private session_photos folder and is served to its owner only.
ALTER TABLE workout_sessions ADD COLUMN photo_key VARCHAR(80) NULL;
ALTER TABLE workout_sessions ADD CONSTRAINT chk_workout_sessions_photo_key
    CHECK (photo_key IS NULL OR photo_key ~ '^session_photos/[0-9a-f-]{36}\.jpg$');
