CREATE TABLE trainer_applications (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_user_id VARCHAR(36) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL AND status IN ('PENDING', 'APPROVED') THEN user_id END
    ) STORED,
    CONSTRAINT trainer_applications_pkey PRIMARY KEY (id),
    CONSTRAINT uq_trainer_applications_active_user_id UNIQUE (active_user_id),
    CONSTRAINT fk_trainer_applications_user FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE INDEX idx_trainer_applications_user_id ON trainer_applications (user_id);
CREATE INDEX idx_trainer_applications_status ON trainer_applications (status);

CREATE TRIGGER trg_trainer_applications_touch_updated_at BEFORE UPDATE ON trainer_applications
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
