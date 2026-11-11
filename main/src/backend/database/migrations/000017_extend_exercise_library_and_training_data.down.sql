DROP TABLE IF EXISTS workout_exercise_sets;

ALTER TABLE workout_exercises
    DROP COLUMN notes,
    DROP COLUMN instructions;

DROP TABLE IF EXISTS exercise_alternatives;

DROP INDEX IF EXISTS idx_exercises_primary_muscle_group;
DROP INDEX IF EXISTS idx_exercises_movement_category;
DROP INDEX IF EXISTS idx_exercises_difficulty;

ALTER TABLE exercises
    DROP COLUMN secondary_muscle_groups,
    DROP COLUMN primary_muscle_group,
    DROP COLUMN movement_category,
    DROP COLUMN instructions;
