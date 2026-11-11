-- Exercise Library structured fields
ALTER TABLE exercises
    ADD COLUMN instructions TEXT,
    ADD COLUMN primary_muscle_group VARCHAR(80),
    ADD COLUMN secondary_muscle_groups VARCHAR(255),
    ADD COLUMN movement_category VARCHAR(60);

CREATE INDEX idx_exercises_primary_muscle_group ON exercises (primary_muscle_group);
CREATE INDEX idx_exercises_movement_category ON exercises (movement_category);
CREATE INDEX idx_exercises_difficulty ON exercises (difficulty);

-- Exercise alternatives: directed links inside the global catalog. A pair may
-- appear only once while active; soft deleting a link frees the pair. No
-- cascade in either direction keeps historical references coherent when an
-- exercise is soft-deleted.
CREATE TABLE exercise_alternatives (
    id VARCHAR(36) NOT NULL,
    exercise_id VARCHAR(36) NOT NULL,
    alternative_exercise_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_alternative VARCHAR(80) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN exercise_id || ':' || alternative_exercise_id END
    ) STORED,
    CONSTRAINT exercise_alternatives_pkey PRIMARY KEY (id),
    CONSTRAINT uq_exercise_alternatives_active_pair UNIQUE (active_alternative),
    CONSTRAINT fk_exercise_alternatives_exercise FOREIGN KEY (exercise_id) REFERENCES exercises (id),
    CONSTRAINT fk_exercise_alternatives_alternative FOREIGN KEY (alternative_exercise_id) REFERENCES exercises (id),
    CONSTRAINT chk_exercise_alternatives_distinct CHECK (exercise_id <> alternative_exercise_id)
);

CREATE INDEX idx_exercise_alternatives_exercise_id ON exercise_alternatives (exercise_id);
CREATE INDEX idx_exercise_alternatives_alternative_id ON exercise_alternatives (alternative_exercise_id);

CREATE TRIGGER trg_exercise_alternatives_touch_updated_at BEFORE UPDATE ON exercise_alternatives
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

-- Per-assignment training data on a workout exercise
ALTER TABLE workout_exercises
    ADD COLUMN instructions TEXT,
    ADD COLUMN notes TEXT;

-- Per-assignment sets: each row is one set of the exercise inside its workout.
-- set_type uses the documented controlled vocabulary and tempo a short string
-- (for example "2010"). All training fields are optional so a set can be
-- progressively filled in.
CREATE TABLE workout_exercise_sets (
    id VARCHAR(36) NOT NULL,
    workout_exercise_id VARCHAR(36) NOT NULL,
    set_number INTEGER NOT NULL,
    reps INTEGER,
    weight_kg NUMERIC(6,2),
    rir INTEGER,
    rpe NUMERIC(3,1),
    rest_seconds INTEGER,
    tempo VARCHAR(20),
    set_type VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_set VARCHAR(80) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN workout_exercise_id || ':' || set_number::text END
    ) STORED,
    CONSTRAINT workout_exercise_sets_pkey PRIMARY KEY (id),
    CONSTRAINT uq_workout_exercise_sets_active_number UNIQUE (active_set),
    CONSTRAINT fk_workout_exercise_sets_workout_exercise FOREIGN KEY (workout_exercise_id) REFERENCES workout_exercises (id),
    CONSTRAINT chk_workout_exercise_sets_number CHECK (set_number > 0),
    CONSTRAINT chk_workout_exercise_sets_reps CHECK (reps IS NULL OR reps > 0),
    CONSTRAINT chk_workout_exercise_sets_weight CHECK (weight_kg IS NULL OR weight_kg >= 0),
    CONSTRAINT chk_workout_exercise_sets_rir CHECK (rir IS NULL OR rir >= 0),
    CONSTRAINT chk_workout_exercise_sets_rpe CHECK (rpe IS NULL OR (rpe >= 1.0 AND rpe <= 10.0)),
    CONSTRAINT chk_workout_exercise_sets_rest CHECK (rest_seconds IS NULL OR rest_seconds >= 0),
    CONSTRAINT chk_workout_exercise_sets_type CHECK (set_type IS NULL OR set_type IN ('warmup', 'working', 'drop', 'backoff', 'failure'))
);

CREATE INDEX idx_workout_exercise_sets_workout_exercise_id ON workout_exercise_sets (workout_exercise_id);

CREATE TRIGGER trg_workout_exercise_sets_touch_updated_at BEFORE UPDATE ON workout_exercise_sets
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
