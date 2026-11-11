CREATE TABLE program_weeks (
    id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    week_number INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_week VARCHAR(50) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN program_id || ':' || week_number::text END
    ) STORED,
    CONSTRAINT program_weeks_pkey PRIMARY KEY (id),
    CONSTRAINT uq_program_weeks_active_week UNIQUE (active_week),
    CONSTRAINT fk_program_weeks_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT chk_program_weeks_week_number CHECK (week_number > 0)
);

CREATE INDEX idx_program_weeks_program_id ON program_weeks (program_id);

CREATE TRIGGER trg_program_weeks_touch_updated_at BEFORE UPDATE ON program_weeks
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
