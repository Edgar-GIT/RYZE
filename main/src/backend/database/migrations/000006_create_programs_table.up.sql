CREATE TABLE programs (
    id VARCHAR(36) NOT NULL,
    trainer_id VARCHAR(36),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT programs_pkey PRIMARY KEY (id),
    CONSTRAINT fk_programs_trainer FOREIGN KEY (trainer_id) REFERENCES trainers (id),
    CONSTRAINT chk_programs_type CHECK (type IN ('free', 'premium', 'personalized')),
    CONSTRAINT chk_programs_status CHECK (status IN ('draft', 'published'))
);

CREATE INDEX idx_programs_trainer_id ON programs (trainer_id);

CREATE TRIGGER trg_programs_touch_updated_at BEFORE UPDATE ON programs
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
