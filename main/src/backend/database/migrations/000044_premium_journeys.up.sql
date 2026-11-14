-- Premium Level 1 and Level 2 journeys (owner request, 2026-11-14).
--
-- Premium Level 1: the client chooses an objective and gives weight, age and
-- height; RYZE picks one published curated package with that objective at
-- random, never one the client already owns. Each pick is recorded (the body
-- data stays with the client's account for the later nutrition work).
-- A fourth objective, sport-specific training, joins the vocabulary.
--
-- Premium Level 2: the long questionnaire is answered before payment. The
-- answers are kept as a per-offer draft until the purchase completes; the
-- completed purchase turns them into the request's first intake.
ALTER TABLE programs DROP CONSTRAINT chk_programs_objective;
ALTER TABLE programs
    ADD CONSTRAINT chk_programs_objective CHECK (
        objective IS NULL OR objective IN ('gain_muscle', 'lose_fat', 'strength_conditioning', 'sport_specific')
    );

CREATE TABLE premium_level_1_matches (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    objective VARCHAR(32) NOT NULL,
    weight_kg NUMERIC(5, 1) NOT NULL,
    height_cm NUMERIC(5, 1) NOT NULL,
    age SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT premium_level_1_matches_pkey PRIMARY KEY (id),
    CONSTRAINT fk_premium_level_1_matches_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_premium_level_1_matches_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT chk_premium_level_1_matches_objective CHECK (
        objective IN ('gain_muscle', 'lose_fat', 'strength_conditioning', 'sport_specific')
    ),
    CONSTRAINT chk_premium_level_1_matches_weight CHECK (weight_kg BETWEEN 30 AND 300),
    CONSTRAINT chk_premium_level_1_matches_height CHECK (height_cm BETWEEN 120 AND 230),
    CONSTRAINT chk_premium_level_1_matches_age CHECK (age BETWEEN 14 AND 100)
);

CREATE INDEX idx_premium_level_1_matches_user ON premium_level_1_matches (user_id, created_at DESC);

CREATE TABLE premium_level_2_intake_drafts (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    answers JSONB NOT NULL DEFAULT '{}'::jsonb,
    complete BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT premium_level_2_intake_drafts_pkey PRIMARY KEY (id),
    CONSTRAINT uq_premium_level_2_intake_drafts_owner UNIQUE (user_id, program_id),
    CONSTRAINT fk_premium_level_2_intake_drafts_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_premium_level_2_intake_drafts_program FOREIGN KEY (program_id) REFERENCES programs (id)
);

CREATE TRIGGER trg_premium_level_2_intake_drafts_touch_updated_at BEFORE UPDATE ON premium_level_2_intake_drafts
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
