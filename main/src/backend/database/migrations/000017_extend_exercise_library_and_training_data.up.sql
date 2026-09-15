-- Exercise Library structured fields
ALTER TABLE exercises
    ADD COLUMN instructions TEXT NULL DEFAULT NULL AFTER description,
    ADD COLUMN primary_muscle_group VARCHAR(80) NULL DEFAULT NULL AFTER target_muscles,
    ADD COLUMN secondary_muscle_groups VARCHAR(255) NULL DEFAULT NULL AFTER primary_muscle_group,
    ADD COLUMN movement_category VARCHAR(60) NULL DEFAULT NULL AFTER difficulty,
    ADD KEY idx_exercises_primary_muscle_group (primary_muscle_group),
    ADD KEY idx_exercises_movement_category (movement_category),
    ADD KEY idx_exercises_difficulty (difficulty);

-- Exercise alternatives: directed links inside the global catalog. A pair may
-- appear only once while active; soft deleting a link frees the pair. No
-- cascade in either direction keeps historical references coherent when an
-- exercise is soft-deleted.
CREATE TABLE exercise_alternatives (
    id VARCHAR(36) NOT NULL,
    exercise_id VARCHAR(36) NOT NULL,
    alternative_exercise_id VARCHAR(36) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    active_alternative VARCHAR(80) GENERATED ALWAYS AS (IF(deleted_at IS NULL, CONCAT(exercise_id, ':', alternative_exercise_id), NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_exercise_alternatives_active_pair (active_alternative),
    KEY idx_exercise_alternatives_exercise_id (exercise_id),
    KEY idx_exercise_alternatives_alternative_id (alternative_exercise_id),
    CONSTRAINT fk_exercise_alternatives_exercise FOREIGN KEY (exercise_id) REFERENCES exercises (id),
    CONSTRAINT fk_exercise_alternatives_alternative FOREIGN KEY (alternative_exercise_id) REFERENCES exercises (id),
    CONSTRAINT chk_exercise_alternatives_distinct CHECK (exercise_id <> alternative_exercise_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Per-assignment training data on a workout exercise
ALTER TABLE workout_exercises
    ADD COLUMN instructions TEXT NULL DEFAULT NULL AFTER position,
    ADD COLUMN notes TEXT NULL DEFAULT NULL AFTER instructions;

-- Per-assignment sets: each row is one set of the exercise inside its workout.
-- set_type uses the documented controlled vocabulary and tempo a short string
-- (for example "2010"). All training fields are optional so a set can be
-- progressively filled in.
CREATE TABLE workout_exercise_sets (
    id VARCHAR(36) NOT NULL,
    workout_exercise_id VARCHAR(36) NOT NULL,
    set_number INT NOT NULL,
    reps INT NULL DEFAULT NULL,
    weight_kg DECIMAL(6,2) NULL DEFAULT NULL,
    rir INT NULL DEFAULT NULL,
    rpe DECIMAL(3,1) NULL DEFAULT NULL,
    rest_seconds INT NULL DEFAULT NULL,
    tempo VARCHAR(20) NULL DEFAULT NULL,
    set_type VARCHAR(20) NULL DEFAULT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    active_set VARCHAR(80) GENERATED ALWAYS AS (IF(deleted_at IS NULL, CONCAT(workout_exercise_id, ':', set_number), NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_workout_exercise_sets_active_number (active_set),
    KEY idx_workout_exercise_sets_workout_exercise_id (workout_exercise_id),
    CONSTRAINT fk_workout_exercise_sets_workout_exercise FOREIGN KEY (workout_exercise_id) REFERENCES workout_exercises (id),
    CONSTRAINT chk_workout_exercise_sets_number CHECK (set_number > 0),
    CONSTRAINT chk_workout_exercise_sets_reps CHECK (reps IS NULL OR reps > 0),
    CONSTRAINT chk_workout_exercise_sets_weight CHECK (weight_kg IS NULL OR weight_kg >= 0),
    CONSTRAINT chk_workout_exercise_sets_rir CHECK (rir IS NULL OR rir >= 0),
    CONSTRAINT chk_workout_exercise_sets_rpe CHECK (rpe IS NULL OR (rpe >= 1.0 AND rpe <= 10.0)),
    CONSTRAINT chk_workout_exercise_sets_rest CHECK (rest_seconds IS NULL OR rest_seconds >= 0),
    CONSTRAINT chk_workout_exercise_sets_type CHECK (set_type IS NULL OR set_type IN ('warmup','working','drop','backoff','failure'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;