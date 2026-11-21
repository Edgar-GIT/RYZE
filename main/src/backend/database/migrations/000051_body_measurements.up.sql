-- Body measurements (owner request, 2026-11-20, as in Lyfta): the member logs
-- weight, body fat and circumferences over time. One value per kind and day;
-- logging the same day again replaces it. Weight is in kg, body fat in %, the
-- rest in cm.
CREATE TABLE body_measurements (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    kind VARCHAR(24) NOT NULL,
    value NUMERIC(6, 2) NOT NULL,
    measured_on DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT body_measurements_pkey PRIMARY KEY (id),
    CONSTRAINT uq_body_measurements_entry UNIQUE (user_id, kind, measured_on),
    CONSTRAINT fk_body_measurements_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_body_measurements_kind CHECK (kind IN (
        'weight', 'body_fat', 'neck', 'shoulders', 'chest', 'biceps_left', 'biceps_right', 'forearm_left',
        'forearm_right', 'waist', 'hips', 'thigh_left', 'thigh_right', 'calf_left', 'calf_right'
    )),
    CONSTRAINT chk_body_measurements_value CHECK (value > 0 AND value < 1000)
);

CREATE INDEX idx_body_measurements_user ON body_measurements (user_id, kind, measured_on DESC);

CREATE TRIGGER trg_body_measurements_touch_updated_at BEFORE UPDATE ON body_measurements
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
