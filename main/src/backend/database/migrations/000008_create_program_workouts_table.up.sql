CREATE TABLE program_workouts (
    id VARCHAR(36) NOT NULL,
    program_week_id VARCHAR(36) NOT NULL,
    position INT NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    active_workout VARCHAR(50) GENERATED ALWAYS AS (IF(deleted_at IS NULL, CONCAT(program_week_id, ':', position), NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_program_workouts_active_workout (active_workout),
    KEY idx_program_workouts_program_week_id (program_week_id),
    CONSTRAINT fk_program_workouts_week FOREIGN KEY (program_week_id) REFERENCES program_weeks (id),
    CONSTRAINT chk_program_workouts_position CHECK (position > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
