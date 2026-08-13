CREATE TABLE workout_exercises (
    id VARCHAR(36) NOT NULL,
    program_workout_id VARCHAR(36) NOT NULL,
    exercise_id VARCHAR(36) NOT NULL,
    position INT NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    active_workout_exercise VARCHAR(50) GENERATED ALWAYS AS (IF(deleted_at IS NULL, CONCAT(program_workout_id, ':', position), NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_workout_exercises_active_position (active_workout_exercise),
    KEY idx_workout_exercises_program_workout_id (program_workout_id),
    KEY idx_workout_exercises_exercise_id (exercise_id),
    CONSTRAINT fk_workout_exercises_workout FOREIGN KEY (program_workout_id) REFERENCES program_workouts (id),
    CONSTRAINT fk_workout_exercises_exercise FOREIGN KEY (exercise_id) REFERENCES exercises (id),
    CONSTRAINT chk_workout_exercises_position CHECK (position > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
