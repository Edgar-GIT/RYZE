CREATE TABLE trainer_clients (
    id VARCHAR(36) NOT NULL,
    trainer_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_relation VARCHAR(80) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN trainer_id || ':' || user_id END
    ) STORED,
    CONSTRAINT trainer_clients_pkey PRIMARY KEY (id),
    CONSTRAINT uq_trainer_clients_active_relation UNIQUE (active_relation),
    CONSTRAINT fk_trainer_clients_trainer FOREIGN KEY (trainer_id) REFERENCES trainers (id),
    CONSTRAINT fk_trainer_clients_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_trainer_clients_not_self CHECK (trainer_id <> user_id)
);

CREATE INDEX idx_trainer_clients_trainer_id ON trainer_clients (trainer_id);
CREATE INDEX idx_trainer_clients_user_id ON trainer_clients (user_id);

CREATE TRIGGER trg_trainer_clients_touch_updated_at BEFORE UPDATE ON trainer_clients
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
