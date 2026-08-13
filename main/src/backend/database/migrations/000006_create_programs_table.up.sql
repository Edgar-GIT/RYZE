CREATE TABLE programs (
    id VARCHAR(36) NOT NULL,
    trainer_id VARCHAR(36) NULL DEFAULT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL DEFAULT NULL,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    KEY idx_programs_trainer_id (trainer_id),
    CONSTRAINT fk_programs_trainer FOREIGN KEY (trainer_id) REFERENCES trainers (id),
    CONSTRAINT chk_programs_type CHECK (type IN ('free', 'premium', 'personalized')),
    CONSTRAINT chk_programs_status CHECK (status IN ('draft', 'published'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
