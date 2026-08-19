CREATE TABLE IF NOT EXISTS purchases (
    id CHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    CONSTRAINT fk_purchases_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_purchases_program FOREIGN KEY (program_id) REFERENCES programs(id),
    CONSTRAINT chk_purchases_status CHECK (status IN ('pending', 'completed', 'failed', 'refunded'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_purchases_user_id ON purchases (user_id);
CREATE INDEX idx_purchases_program_id ON purchases (program_id);

CREATE TABLE IF NOT EXISTS entitlements (
    id CHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    active_entitlement VARCHAR(73) GENERATED ALWAYS AS (
        IF(deleted_at IS NULL, CONCAT(user_id, ':', program_id), NULL)
    ) STORED,
    UNIQUE KEY uq_entitlements_active_entitlement (active_entitlement),
    CONSTRAINT fk_entitlements_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_entitlements_program FOREIGN KEY (program_id) REFERENCES programs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_entitlements_user_id ON entitlements (user_id);
CREATE INDEX idx_entitlements_program_id ON entitlements (program_id);
