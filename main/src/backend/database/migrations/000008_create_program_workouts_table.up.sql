CREATE TABLE program_workouts (
    id VARCHAR(36) NOT NULL,
    program_week_id VARCHAR(36) NOT NULL,
    position INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_workout VARCHAR(50) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN program_week_id || ':' || position::text END
    ) STORED,
    CONSTRAINT program_workouts_pkey PRIMARY KEY (id),
    CONSTRAINT uq_program_workouts_active_workout UNIQUE (active_workout),
    CONSTRAINT fk_program_workouts_week FOREIGN KEY (program_week_id) REFERENCES program_weeks (id),
    CONSTRAINT chk_program_workouts_position CHECK (position > 0)
);

CREATE INDEX idx_program_workouts_program_week_id ON program_workouts (program_week_id);

CREATE TRIGGER trg_program_workouts_touch_updated_at BEFORE UPDATE ON program_workouts
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
