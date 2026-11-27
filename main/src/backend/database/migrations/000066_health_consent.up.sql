-- Explicit consent for health data (GDPR art. 9(2)(a); owner request,
-- 2026-11-24). Body measurements and the Premium Level 2 questionnaire are
-- health data: the person gives explicit consent before sending them, the
-- server records when, and withdrawing it erases that data.
ALTER TABLE users ADD COLUMN health_consent_at TIMESTAMPTZ NULL;
ALTER TABLE premium_level_2_intakes ADD COLUMN health_consent_at TIMESTAMPTZ NULL;
