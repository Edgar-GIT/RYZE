CREATE TABLE workout_exercises (
    id VARCHAR(36) NOT NULL,
    program_workout_id VARCHAR(36) NOT NULL,
    exercise_id VARCHAR(36) NOT NULL,
    position INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    active_workout_exercise VARCHAR(50) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL THEN program_workout_id || ':' || position::text END
    ) STORED,
    CONSTRAINT workout_exercises_pkey PRIMARY KEY (id),
    CONSTRAINT uq_workout_exercises_active_position UNIQUE (active_workout_exercise),
    CONSTRAINT fk_workout_exercises_workout FOREIGN KEY (program_workout_id) REFERENCES program_workouts (id),
    CONSTRAINT fk_workout_exercises_exercise FOREIGN KEY (exercise_id) REFERENCES exercises (id),
    CONSTRAINT chk_workout_exercises_position CHECK (position > 0)
);

CREATE INDEX idx_workout_exercises_program_workout_id ON workout_exercises (program_workout_id);
CREATE INDEX idx_workout_exercises_exercise_id ON workout_exercises (exercise_id);

CREATE TRIGGER trg_workout_exercises_touch_updated_at BEFORE UPDATE ON workout_exercises
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
