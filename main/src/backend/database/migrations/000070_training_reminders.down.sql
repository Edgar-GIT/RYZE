ALTER TABLE users
    DROP COLUMN IF EXISTS last_training_reminder_at,
    DROP COLUMN IF EXISTS training_reminders;
