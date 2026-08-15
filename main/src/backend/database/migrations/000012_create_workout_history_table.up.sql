CREATE TABLE IF NOT EXISTS workout_history (
    id CHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    program_workout_id VARCHAR(36) NOT NULL,
    completed_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    KEY idx_workout_history_user_id (user_id),
    KEY idx_workout_history_program_workout_id (program_workout_id),
    CONSTRAINT fk_workout_history_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_workout_history_program_workout FOREIGN KEY (program_workout_id) REFERENCES program_workouts (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- The history is an append-only execution log owned by the authenticated user.
-- A workout may be completed more than once (repeated executions are preserved),
-- so no unique constraint exists on (user_id, program_workout_id). Soft-deleted
-- rows are excluded from regular queries through GORM's DeletedAt handling.
