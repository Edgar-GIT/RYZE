-- Exercise names are unique among active rows regardless of letter case, as
-- they were under MariaDB's case-insensitive collation.
CREATE TABLE exercises (
    id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    target_muscles VARCHAR(255),
    equipment VARCHAR(255),
    difficulty VARCHAR(50),
    video_url VARCHAR(500),
    image_url VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_name VARCHAR(255) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN lower(name) END) STORED,
    CONSTRAINT exercises_pkey PRIMARY KEY (id),
    CONSTRAINT uq_exercises_active_name UNIQUE (active_name)
);

CREATE INDEX idx_exercises_name ON exercises (name);

CREATE TRIGGER trg_exercises_touch_updated_at BEFORE UPDATE ON exercises
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
