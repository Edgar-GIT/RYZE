-- Test Mode persona ownership (v2.96.0).
--
-- A persona account must be provably owned by the platform before Test Mode
-- may mint a session for it. Without a durable marker the only link between a
-- fixed persona email and the feature is the email string itself, so a public
-- registration could squat the address and be adopted as the persona.
--
-- test_persona is written only by the Test Mode provisioning service. The
-- generated active_test_persona column makes ownership unique among active
-- accounts while allowing a deleted identity to be replaced, matching the
-- soft-delete lifecycle used by every other account. Existing rows stay NULL:
-- an account that predates this migration is never retroactively trusted as a
-- persona. It must be reviewed and provisioned explicitly instead.
ALTER TABLE users
    ADD COLUMN test_persona VARCHAR(16) NULL DEFAULT NULL AFTER session_version,
    ADD COLUMN active_test_persona VARCHAR(16) GENERATED ALWAYS AS (IF(deleted_at IS NULL, test_persona, NULL)) STORED,
    ADD CONSTRAINT chk_users_test_persona CHECK (test_persona IS NULL OR test_persona IN ('client', 'trainer')),
    ADD UNIQUE KEY uq_users_active_test_persona (active_test_persona);
