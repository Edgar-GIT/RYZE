CREATE TABLE trainer_clients (
    id VARCHAR(36) NOT NULL,
    trainer_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    active_relation VARCHAR(80) GENERATED ALWAYS AS (IF(deleted_at IS NULL, CONCAT(trainer_id, ':', user_id), NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_trainer_clients_active_relation (active_relation),
    KEY idx_trainer_clients_trainer_id (trainer_id),
    KEY idx_trainer_clients_user_id (user_id),
    CONSTRAINT fk_trainer_clients_trainer FOREIGN KEY (trainer_id) REFERENCES trainers (id),
    CONSTRAINT fk_trainer_clients_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_trainer_clients_not_self CHECK (trainer_id <> user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
