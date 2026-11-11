-- Secure programme access, Free activation and the interactive workout
-- experience.
--
-- Free programs are activated by the authenticated client instead of going
-- through checkout, so they need their own grant. The activation is not an
-- entitlement or an assignment: it never creates a purchase and never makes the
-- program trainer-owned. `expires_at` is nullable because the product rules do
-- not define a Free access duration yet; when it is set, the read paths ignore
-- the expired row and a new activation may replace it.
CREATE TABLE IF NOT EXISTS free_program_activations (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    activated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_free_activation VARCHAR(36) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN user_id END
    ) STORED,
    CONSTRAINT uq_free_program_activations_active_user UNIQUE (active_free_activation),
    CONSTRAINT fk_free_program_activations_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_free_program_activations_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT chk_free_program_activations_expiry CHECK (
        expires_at IS NULL OR expires_at > activated_at
    )
);

CREATE INDEX idx_free_program_activations_user_id ON free_program_activations (user_id);
CREATE INDEX idx_free_program_activations_program_id ON free_program_activations (program_id);
CREATE INDEX idx_free_program_activations_expires_at ON free_program_activations (expires_at);

CREATE TRIGGER trg_free_program_activations_touch_updated_at BEFORE UPDATE ON free_program_activations
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

-- Entitlements gain an optional expiry so the access gate can enforce a
-- purchased access window without changing the existing purchase lifecycle.
-- Existing entitlements stay valid forever (NULL), which preserves current
-- behaviour. The generated one-active constraint is unchanged: an expired row
-- is reclaimed by the existing purchase restoration path rather than by a
-- second row.
ALTER TABLE entitlements
    ADD COLUMN expires_at TIMESTAMPTZ;

CREATE INDEX idx_entitlements_expires_at ON entitlements (expires_at);

-- One execution of one prescribed workout. The prescription is snapshotted into
-- the session exercise/set rows so later trainer edits never rewrite history.
-- The active_session_user generated column enforces at most one live session
-- per user at the database level, including across concurrent requests.
CREATE TABLE IF NOT EXISTS workout_sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    program_workout_id VARCHAR(36) NOT NULL,
    program_name VARCHAR(255) NOT NULL,
    week_number INTEGER NOT NULL,
    workout_position INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress',
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_resumed_at TIMESTAMPTZ,
    paused_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    abandoned_at TIMESTAMPTZ,
    accumulated_seconds INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_session_user VARCHAR(36) GENERATED ALWAYS AS (
        CASE WHEN status IN ('in_progress', 'paused') AND deleted_at IS NULL THEN user_id END
    ) STORED,
    CONSTRAINT uq_workout_sessions_active_user UNIQUE (active_session_user),
    CONSTRAINT fk_workout_sessions_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_workout_sessions_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT fk_workout_sessions_workout FOREIGN KEY (program_workout_id) REFERENCES program_workouts (id),
    CONSTRAINT chk_workout_sessions_status CHECK (
        status IN ('in_progress', 'paused', 'completed', 'abandoned')
    ),
    CONSTRAINT chk_workout_sessions_seconds CHECK (accumulated_seconds >= 0),
    CONSTRAINT chk_workout_sessions_week CHECK (week_number > 0),
    CONSTRAINT chk_workout_sessions_position CHECK (workout_position > 0)
);

CREATE INDEX idx_workout_sessions_user_id ON workout_sessions (user_id);
CREATE INDEX idx_workout_sessions_program_id ON workout_sessions (program_id);
CREATE INDEX idx_workout_sessions_workout_id ON workout_sessions (program_workout_id);
CREATE INDEX idx_workout_sessions_user_status ON workout_sessions (user_id, status);
CREATE INDEX idx_workout_sessions_started_at ON workout_sessions (started_at);

CREATE TRIGGER trg_workout_sessions_touch_updated_at BEFORE UPDATE ON workout_sessions
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

CREATE TABLE IF NOT EXISTS workout_session_exercises (
    id VARCHAR(36) PRIMARY KEY,
    workout_session_id VARCHAR(36) NOT NULL,
    exercise_id VARCHAR(36) NOT NULL,
    position INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    instructions TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_workout_session_exercises_session FOREIGN KEY (workout_session_id) REFERENCES workout_sessions (id),
    CONSTRAINT fk_workout_session_exercises_exercise FOREIGN KEY (exercise_id) REFERENCES exercises (id),
    CONSTRAINT chk_workout_session_exercises_position CHECK (position > 0)
);

CREATE INDEX idx_workout_session_exercises_session_id ON workout_session_exercises (workout_session_id);
CREATE INDEX idx_workout_session_exercises_exercise_id ON workout_session_exercises (exercise_id);

CREATE TRIGGER trg_workout_session_exercises_touch_updated_at BEFORE UPDATE ON workout_session_exercises
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

CREATE TABLE IF NOT EXISTS workout_session_sets (
    id VARCHAR(36) PRIMARY KEY,
    workout_session_exercise_id VARCHAR(36) NOT NULL,
    set_number INTEGER NOT NULL,
    set_type VARCHAR(20) NOT NULL,
    prescribed_reps INTEGER,
    prescribed_weight_kg NUMERIC(6,2),
    prescribed_rir INTEGER,
    prescribed_rpe NUMERIC(3,1),
    prescribed_rest_seconds INTEGER,
    prescribed_tempo VARCHAR(20),
    actual_reps INTEGER,
    actual_weight_kg NUMERIC(6,2),
    actual_rir INTEGER,
    actual_rpe NUMERIC(3,1),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_workout_session_sets_exercise FOREIGN KEY (workout_session_exercise_id) REFERENCES workout_session_exercises (id),
    CONSTRAINT uq_workout_session_sets_position UNIQUE (workout_session_exercise_id, set_number),
    CONSTRAINT chk_workout_session_sets_status CHECK (
        status IN ('pending', 'completed', 'skipped')
    ),
    CONSTRAINT chk_workout_session_sets_number CHECK (set_number > 0),
    CONSTRAINT chk_workout_session_sets_reps CHECK (actual_reps IS NULL OR actual_reps >= 0),
    CONSTRAINT chk_workout_session_sets_weight CHECK (actual_weight_kg IS NULL OR actual_weight_kg >= 0),
    CONSTRAINT chk_workout_session_sets_rir CHECK (actual_rir IS NULL OR actual_rir >= 0),
    CONSTRAINT chk_workout_session_sets_rpe CHECK (actual_rpe IS NULL OR (actual_rpe >= 0 AND actual_rpe <= 10))
);

CREATE INDEX idx_workout_session_sets_exercise_id ON workout_session_sets (workout_session_exercise_id);
CREATE INDEX idx_workout_session_sets_status ON workout_session_sets (status);

CREATE TRIGGER trg_workout_session_sets_touch_updated_at BEFORE UPDATE ON workout_session_sets
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
