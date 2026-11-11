CREATE TABLE IF NOT EXISTS workout_history (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    program_workout_id VARCHAR(36) NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT workout_history_pkey PRIMARY KEY (id),
    CONSTRAINT fk_workout_history_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_workout_history_program_workout FOREIGN KEY (program_workout_id) REFERENCES program_workouts (id)
);

CREATE INDEX idx_workout_history_user_id ON workout_history (user_id);
CREATE INDEX idx_workout_history_program_workout_id ON workout_history (program_workout_id);

CREATE TRIGGER trg_workout_history_touch_updated_at BEFORE UPDATE ON workout_history
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

-- The history is an append-only execution log owned by the authenticated user.
-- A workout may be completed more than once (repeated executions are preserved),
-- so no unique constraint exists on (user_id, program_workout_id). Soft-deleted
-- rows are excluded from regular queries through GORM's DeletedAt handling.
