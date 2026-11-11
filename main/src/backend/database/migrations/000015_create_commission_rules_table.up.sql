CREATE TABLE commission_rules (
    id VARCHAR(36) NOT NULL,
    trainer_id VARCHAR(36) NOT NULL,
    commission_bps INTEGER NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_commission_rule VARCHAR(39) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN trainer_id END
    ) STORED,
    CONSTRAINT commission_rules_pkey PRIMARY KEY (id),
    CONSTRAINT fk_commission_rules_trainer FOREIGN KEY (trainer_id) REFERENCES trainers (id),
    CONSTRAINT chk_commission_rules_bps CHECK (commission_bps >= 0 AND commission_bps <= 10000),
    CONSTRAINT uk_active_commission_rule UNIQUE (active_commission_rule)
);

CREATE INDEX idx_commission_rules_trainer_id ON commission_rules (trainer_id);

CREATE TRIGGER trg_commission_rules_touch_updated_at BEFORE UPDATE ON commission_rules
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
