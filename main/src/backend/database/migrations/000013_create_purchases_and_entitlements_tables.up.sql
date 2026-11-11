CREATE TABLE IF NOT EXISTS purchases (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_purchases_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_purchases_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT chk_purchases_status CHECK (status IN ('pending', 'completed', 'failed'))
);

CREATE INDEX idx_purchases_user_id ON purchases (user_id);
CREATE INDEX idx_purchases_program_id ON purchases (program_id);

CREATE TRIGGER trg_purchases_touch_updated_at BEFORE UPDATE ON purchases
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

CREATE TABLE IF NOT EXISTS entitlements (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_entitlement VARCHAR(73) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN user_id || ':' || program_id END
    ) STORED,
    CONSTRAINT uq_entitlements_active_entitlement UNIQUE (active_entitlement),
    CONSTRAINT fk_entitlements_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_entitlements_program FOREIGN KEY (program_id) REFERENCES programs (id)
);

CREATE INDEX idx_entitlements_user_id ON entitlements (user_id);
CREATE INDEX idx_entitlements_program_id ON entitlements (program_id);

CREATE TRIGGER trg_entitlements_touch_updated_at BEFORE UPDATE ON entitlements
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
