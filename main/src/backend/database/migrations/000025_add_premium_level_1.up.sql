-- Premium Level 1: the product family a catalogue program belongs to. `type`
-- stays the commercial/pricing category (free, premium, personalized) and is
-- left untouched, because Generic Programs and Premium Level 1 both sell under
-- it. `product_type` is the orthogonal dimension that decides which parts of
-- the purchase lifecycle a program participates in: a Premium Level 1 program
-- requires a submitted nutrition questionnaire before checkout and receives a
-- server-owned nutrition assignment once the purchase is verified.
--
-- The default is 'generic' so every pre-existing program keeps its current
-- behaviour without a backfill. `premium_level_2` is deliberately absent: it is
-- not implemented, and a later migration widens the constraint when it is.
ALTER TABLE programs
    ADD COLUMN product_type VARCHAR(32) NOT NULL DEFAULT 'generic',
    ADD CONSTRAINT chk_programs_product_type CHECK (
        product_type IN ('generic', 'premium_level_1')
    );

CREATE INDEX idx_programs_product_type ON programs (product_type);

-- The collected client and health intake for a Premium Level 1 program. Exactly
-- one active questionnaire exists per (user, program): resubmitting rewrites the
-- row and increments `version`, so the nutrition assignment always records the
-- questionnaire revision it was generated from. Answers are stored normalized
-- and validated, never as raw client input, and are treated as sensitive data:
-- they are never logged and never projected into public, trainer or commerce
-- responses.
CREATE TABLE IF NOT EXISTS nutrition_questionnaires (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    answers JSONB NOT NULL,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_questionnaire VARCHAR(73) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN user_id || ':' || program_id END
    ) STORED,
    CONSTRAINT uq_nutrition_questionnaires_active UNIQUE (active_questionnaire),
    CONSTRAINT fk_nutrition_questionnaires_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_nutrition_questionnaires_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT chk_nutrition_questionnaires_version CHECK (version > 0)
);

CREATE INDEX idx_nutrition_questionnaires_user_id ON nutrition_questionnaires (user_id);
CREATE INDEX idx_nutrition_questionnaires_program_id ON nutrition_questionnaires (program_id);

CREATE TRIGGER trg_nutrition_questionnaires_touch_updated_at BEFORE UPDATE ON nutrition_questionnaires
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

-- The server-owned nutrition configuration produced for a purchased Premium
-- Level 1 program. `status` is the observable state machine the client polls:
-- a purchase never implies a finished plan, and only 'completed' rows may be
-- rendered as a plan. `configuration` is the generated plan snapshot and
-- `questionnaire_version` pins the intake it was derived from, so a later
-- questionnaire revision can be detected as out of date instead of silently
-- serving a stale plan.
--
-- Generation is deterministic and server-side, so a failed or interrupted run
-- is recoverable in place through the idempotent retry path rather than by
-- creating duplicate plans. `failure_reason` is internal diagnostics: it is
-- never returned to the client and never logged with questionnaire content.
CREATE TABLE IF NOT EXISTS nutrition_assignments (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    questionnaire_id VARCHAR(36) NOT NULL,
    questionnaire_version INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    version INTEGER NOT NULL DEFAULT 1,
    configuration JSONB,
    failure_reason VARCHAR(255),
    generated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_assignment VARCHAR(73) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN user_id || ':' || program_id END
    ) STORED,
    CONSTRAINT uq_nutrition_assignments_active UNIQUE (active_assignment),
    CONSTRAINT fk_nutrition_assignments_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_nutrition_assignments_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT fk_nutrition_assignments_questionnaire FOREIGN KEY (questionnaire_id) REFERENCES nutrition_questionnaires (id),
    CONSTRAINT chk_nutrition_assignments_status CHECK (
        status IN ('pending', 'processing', 'completed', 'failed')
    ),
    CONSTRAINT chk_nutrition_assignments_version CHECK (version > 0),
    CONSTRAINT chk_nutrition_assignments_questionnaire_version CHECK (questionnaire_version > 0)
);

CREATE INDEX idx_nutrition_assignments_user_id ON nutrition_assignments (user_id);
CREATE INDEX idx_nutrition_assignments_program_id ON nutrition_assignments (program_id);
CREATE INDEX idx_nutrition_assignments_questionnaire_id ON nutrition_assignments (questionnaire_id);
CREATE INDEX idx_nutrition_assignments_status ON nutrition_assignments (status);

CREATE TRIGGER trg_nutrition_assignments_touch_updated_at BEFORE UPDATE ON nutrition_assignments
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

-- Safe read surface for the Premium Level 1 client journey. Repositories read
-- through this view instead of the questionnaire base table so the write model
-- (internal revision bookkeeping) is never exposed as an API contract.
CREATE OR REPLACE VIEW v_nutrition_questionnaire_summary AS
SELECT
    q.id,
    q.user_id,
    q.program_id,
    q.version,
    q.submitted_at,
    q.created_at,
    q.updated_at
FROM nutrition_questionnaires q
WHERE q.deleted_at IS NULL;

CREATE OR REPLACE VIEW v_nutrition_assignment AS
SELECT
    a.id,
    a.user_id,
    a.program_id,
    a.questionnaire_id,
    a.questionnaire_version,
    a.status,
    a.version,
    a.configuration,
    a.generated_at,
    a.created_at,
    a.updated_at
FROM nutrition_assignments a
WHERE a.deleted_at IS NULL;
