CREATE TABLE trainer_applications (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    active_user_id VARCHAR(36) GENERATED ALWAYS AS (IF(deleted_at IS NULL AND status IN ('PENDING','APPROVED'), user_id, NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_trainer_applications_active_user_id (active_user_id),
    KEY idx_trainer_applications_user_id (user_id),
    KEY idx_trainer_applications_status (status),
    CONSTRAINT fk_trainer_applications_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
