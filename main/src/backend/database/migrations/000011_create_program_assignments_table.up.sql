CREATE TABLE IF NOT EXISTS program_assignments (
    id CHAR(36) PRIMARY KEY,
    trainer_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    active_assignment VARCHAR(97) GENERATED ALWAYS AS (
        IF(deleted_at IS NULL, CONCAT(trainer_id, ':', user_id), NULL)
    ) STORED,
    UNIQUE KEY uq_program_assignments_active_assignment (active_assignment),
    CONSTRAINT fk_program_assignments_trainer FOREIGN KEY (trainer_id) REFERENCES trainers(id),
    CONSTRAINT fk_program_assignments_program FOREIGN KEY (program_id) REFERENCES programs(id),
    CONSTRAINT fk_program_assignments_user FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Indexes
CREATE INDEX idx_program_assignments_trainer_id ON program_assignments (trainer_id);
CREATE INDEX idx_program_assignments_program_id ON program_assignments (program_id);
CREATE INDEX idx_program_assignments_user_id ON program_assignments (user_id);

-- The generated column keeps the one-active-assignment rule in the database
-- itself: a trainer can have at most one active assigned program per client.
-- Soft-deleted assignments become NULL so the unique key never fires on them,
-- while the rows are preserved for history.
