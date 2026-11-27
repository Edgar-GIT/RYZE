-- Discount codes (owner request, 2026-11-24). ADMIN_1 or ADMIN_2 create a
-- code with a percentage off and how long it can be used: at most 30 days
-- (or a number of hours). A buyer enters it at checkout; the server applies
-- it to the pending purchase before the payment opens. A code counts a use
-- per completed purchase and can be used once per person.
CREATE TABLE discount_codes (
    id VARCHAR(36) NOT NULL,
    code VARCHAR(32) NOT NULL,
    percent_off INT NOT NULL,
    applies_to VARCHAR(20) NOT NULL DEFAULT 'all',
    starts_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    max_uses INT NULL,
    disabled_at TIMESTAMPTZ NULL,
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT discount_codes_pkey PRIMARY KEY (id),
    CONSTRAINT uq_discount_codes_code UNIQUE (code),
    CONSTRAINT chk_discount_codes_code CHECK (code ~ '^[A-Z0-9][A-Z0-9_-]{2,31}$'),
    CONSTRAINT chk_discount_codes_percent CHECK (percent_off BETWEEN 1 AND 90),
    CONSTRAINT chk_discount_codes_applies_to CHECK (applies_to IN ('all', 'premium_level_1', 'premium_level_2', 'trainer_plans')),
    CONSTRAINT chk_discount_codes_window CHECK (expires_at > starts_at AND expires_at <= starts_at + INTERVAL '30 days'),
    CONSTRAINT chk_discount_codes_max_uses CHECK (max_uses IS NULL OR max_uses > 0)
);

CREATE TRIGGER trg_discount_codes_touch_updated_at BEFORE UPDATE ON discount_codes
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

-- The discount on a purchase: the list price before it and the amount off.
ALTER TABLE purchases
    ADD COLUMN discount_code_id VARCHAR(36) NULL,
    ADD COLUMN original_price_minor_units BIGINT NULL,
    ADD COLUMN discount_minor_units BIGINT NOT NULL DEFAULT 0,
    ADD CONSTRAINT fk_purchases_discount_code FOREIGN KEY (discount_code_id) REFERENCES discount_codes (id),
    ADD CONSTRAINT chk_purchases_discount CHECK (
        (discount_code_id IS NULL AND discount_minor_units = 0)
        OR (discount_code_id IS NOT NULL AND original_price_minor_units IS NOT NULL AND discount_minor_units > 0
            AND price_minor_units = original_price_minor_units - discount_minor_units)
    );
CREATE INDEX idx_purchases_discount_code ON purchases (discount_code_id) WHERE discount_code_id IS NOT NULL;
