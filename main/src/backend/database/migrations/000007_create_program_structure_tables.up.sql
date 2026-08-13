CREATE TABLE program_weeks (
    id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    week_number INT NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    active_week VARCHAR(50) GENERATED ALWAYS AS (IF(deleted_at IS NULL, CONCAT(program_id, ':', week_number), NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_program_weeks_active_week (active_week),
    KEY idx_program_weeks_program_id (program_id),
    CONSTRAINT fk_program_weeks_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT chk_program_weeks_week_number CHECK (week_number > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
