DROP TABLE IF EXISTS workout_exercise_sets;

ALTER TABLE workout_exercises
    DROP COLUMN notes,
    DROP COLUMN instructions;

DROP TABLE IF EXISTS exercise_alternatives;

ALTER TABLE exercises
    DROP INDEX idx_exercises_primary_muscle_group,
    DROP INDEX idx_exercises_movement_category,
    DROP INDEX idx_exercises_difficulty;

ALTER TABLE exercises
    DROP COLUMN secondary_muscle_groups,
    DROP COLUMN primary_muscle_group,
    DROP COLUMN movement_category,
    DROP COLUMN instructions;