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
    id CHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    activated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    expires_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    active_free_activation VARCHAR(36) GENERATED ALWAYS AS (
        IF(deleted_at IS NULL, user_id, NULL)
    ) STORED,
    UNIQUE KEY uq_free_program_activations_active_user (active_free_activation),
    CONSTRAINT fk_free_program_activations_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_free_program_activations_program FOREIGN KEY (program_id) REFERENCES programs(id),
    CONSTRAINT chk_free_program_activations_expiry CHECK (
        expires_at IS NULL OR expires_at > activated_at
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_free_program_activations_user_id ON free_program_activations (user_id);
CREATE INDEX idx_free_program_activations_program_id ON free_program_activations (program_id);
CREATE INDEX idx_free_program_activations_expires_at ON free_program_activations (expires_at);

-- Entitlements gain an optional expiry so the access gate can enforce a
-- purchased access window without changing the existing purchase lifecycle.
-- Existing entitlements stay valid forever (NULL), which preserves current
-- behaviour. The generated one-active constraint is unchanged: an expired row
-- is reclaimed by the existing purchase restoration path rather than by a
-- second row.
ALTER TABLE entitlements
    ADD COLUMN expires_at DATETIME(6) NULL AFTER program_id,
    ADD KEY idx_entitlements_expires_at (expires_at);

-- One execution of one prescribed workout. The prescription is snapshotted into
-- the session exercise/set rows so later trainer edits never rewrite history.
-- The active_session_user generated column enforces at most one live session
-- per user at the database level, including across concurrent requests.
CREATE TABLE IF NOT EXISTS workout_sessions (
    id CHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    program_workout_id VARCHAR(36) NOT NULL,
    program_name VARCHAR(255) NOT NULL,
    week_number INT NOT NULL,
    workout_position INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress',
    started_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    last_resumed_at DATETIME(6) NULL,
    paused_at DATETIME(6) NULL,
    completed_at DATETIME(6) NULL,
    abandoned_at DATETIME(6) NULL,
    accumulated_seconds INT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    active_session_user VARCHAR(36) GENERATED ALWAYS AS (
        IF(status IN ('in_progress', 'paused') AND deleted_at IS NULL, user_id, NULL)
    ) STORED,
    UNIQUE KEY uq_workout_sessions_active_user (active_session_user),
    CONSTRAINT fk_workout_sessions_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_workout_sessions_program FOREIGN KEY (program_id) REFERENCES programs(id),
    CONSTRAINT fk_workout_sessions_workout FOREIGN KEY (program_workout_id) REFERENCES program_workouts(id),
    CONSTRAINT chk_workout_sessions_status CHECK (
        status IN ('in_progress', 'paused', 'completed', 'abandoned')
    ),
    CONSTRAINT chk_workout_sessions_seconds CHECK (accumulated_seconds >= 0),
    CONSTRAINT chk_workout_sessions_week CHECK (week_number > 0),
    CONSTRAINT chk_workout_sessions_position CHECK (workout_position > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_workout_sessions_user_id ON workout_sessions (user_id);
CREATE INDEX idx_workout_sessions_program_id ON workout_sessions (program_id);
CREATE INDEX idx_workout_sessions_workout_id ON workout_sessions (program_workout_id);
CREATE INDEX idx_workout_sessions_user_status ON workout_sessions (user_id, status);
CREATE INDEX idx_workout_sessions_started_at ON workout_sessions (started_at);

CREATE TABLE IF NOT EXISTS workout_session_exercises (
    id CHAR(36) PRIMARY KEY,
    workout_session_id CHAR(36) NOT NULL,
    exercise_id VARCHAR(36) NOT NULL,
    position INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    instructions TEXT NULL,
    notes TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    CONSTRAINT fk_workout_session_exercises_session FOREIGN KEY (workout_session_id) REFERENCES workout_sessions(id),
    CONSTRAINT fk_workout_session_exercises_exercise FOREIGN KEY (exercise_id) REFERENCES exercises(id),
    CONSTRAINT chk_workout_session_exercises_position CHECK (position > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_workout_session_exercises_session_id ON workout_session_exercises (workout_session_id);
CREATE INDEX idx_workout_session_exercises_exercise_id ON workout_session_exercises (exercise_id);

CREATE TABLE IF NOT EXISTS workout_session_sets (
    id CHAR(36) PRIMARY KEY,
    workout_session_exercise_id CHAR(36) NOT NULL,
    set_number INT NOT NULL,
    set_type VARCHAR(20) NOT NULL,
    prescribed_reps INT NULL,
    prescribed_weight_kg DECIMAL(6,2) NULL,
    prescribed_rir INT NULL,
    prescribed_rpe DECIMAL(3,1) NULL,
    prescribed_rest_seconds INT NULL,
    prescribed_tempo VARCHAR(20) NULL,
    actual_reps INT NULL,
    actual_weight_kg DECIMAL(6,2) NULL,
    actual_rir INT NULL,
    actual_rpe DECIMAL(3,1) NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    completed_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    CONSTRAINT fk_workout_session_sets_exercise FOREIGN KEY (workout_session_exercise_id) REFERENCES workout_session_exercises(id),
    UNIQUE KEY uq_workout_session_sets_position (workout_session_exercise_id, set_number),
    CONSTRAINT chk_workout_session_sets_status CHECK (
        status IN ('pending', 'completed', 'skipped')
    ),
    CONSTRAINT chk_workout_session_sets_number CHECK (set_number > 0),
    CONSTRAINT chk_workout_session_sets_reps CHECK (actual_reps IS NULL OR actual_reps >= 0),
    CONSTRAINT chk_workout_session_sets_weight CHECK (actual_weight_kg IS NULL OR actual_weight_kg >= 0),
    CONSTRAINT chk_workout_session_sets_rir CHECK (actual_rir IS NULL OR actual_rir >= 0),
    CONSTRAINT chk_workout_session_sets_rpe CHECK (actual_rpe IS NULL OR (actual_rpe >= 0 AND actual_rpe <= 10))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_workout_session_sets_exercise_id ON workout_session_sets (workout_session_exercise_id);
CREATE INDEX idx_workout_session_sets_status ON workout_session_sets (status);
