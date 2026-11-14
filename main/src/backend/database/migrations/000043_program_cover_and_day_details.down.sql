ALTER TABLE program_workouts
    DROP CONSTRAINT IF EXISTS chk_program_workouts_focus,
    DROP COLUMN IF EXISTS focus,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS name;

ALTER TABLE programs
    DROP CONSTRAINT IF EXISTS chk_programs_cover_image,
    DROP COLUMN IF EXISTS cover_image;
