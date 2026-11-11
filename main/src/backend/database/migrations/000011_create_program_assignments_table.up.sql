CREATE TABLE IF NOT EXISTS program_assignments (
    id VARCHAR(36) PRIMARY KEY,
    trainer_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_assignment VARCHAR(97) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN trainer_id || ':' || user_id END
    ) STORED,
    CONSTRAINT uq_program_assignments_active_assignment UNIQUE (active_assignment),
    CONSTRAINT fk_program_assignments_trainer FOREIGN KEY (trainer_id) REFERENCES trainers (id),
    CONSTRAINT fk_program_assignments_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT fk_program_assignments_user FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE INDEX idx_program_assignments_trainer_id ON program_assignments (trainer_id);
CREATE INDEX idx_program_assignments_program_id ON program_assignments (program_id);
CREATE INDEX idx_program_assignments_user_id ON program_assignments (user_id);

CREATE TRIGGER trg_program_assignments_touch_updated_at BEFORE UPDATE ON program_assignments
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

-- The generated column keeps the one-active-assignment rule in the database
-- itself: a trainer can have at most one active assigned program per client.
-- Soft-deleted assignments become NULL so the unique key never fires on them,
-- while the rows are preserved for history.
