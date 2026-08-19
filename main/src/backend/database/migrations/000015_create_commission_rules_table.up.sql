CREATE TABLE commission_rules (
    id VARCHAR(36) NOT NULL,
    trainer_id VARCHAR(36) NOT NULL,
    commission_bps INT UNSIGNED NOT NULL,
    valid_from DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    valid_until DATETIME(6) NULL DEFAULT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    active_commission_rule VARCHAR(39) AS (IF(deleted_at IS NULL, trainer_id, NULL)) STORED,
    PRIMARY KEY (id),
    KEY idx_commission_rules_trainer_id (trainer_id),
    CONSTRAINT fk_commission_rules_trainer FOREIGN KEY (trainer_id) REFERENCES trainers (id),
    CONSTRAINT chk_commission_rules_bps CHECK (commission_bps <= 10000),
    UNIQUE KEY uk_active_commission_rule (active_commission_rule)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
