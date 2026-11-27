-- Training reminders (owner request, 2026-11-24): a member with a plan who has
-- not trained for a few days gets an occasional nudge (in the app and by
-- e-mail), at most once a week. It can be turned off in the account settings.
ALTER TABLE users
    ADD COLUMN training_reminders BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN last_training_reminder_at TIMESTAMPTZ NULL;
