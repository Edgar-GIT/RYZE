CREATE TABLE trainers (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_user_id VARCHAR(36) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN user_id END) STORED,
    CONSTRAINT trainers_pkey PRIMARY KEY (id),
    CONSTRAINT uq_trainers_active_user_id UNIQUE (active_user_id),
    CONSTRAINT fk_trainers_user FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE TRIGGER trg_trainers_touch_updated_at BEFORE UPDATE ON trainers
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
