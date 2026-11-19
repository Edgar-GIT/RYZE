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
