ALTER TABLE programs
    DROP CONSTRAINT IF EXISTS chk_programs_criteria_level1,
    DROP CONSTRAINT IF EXISTS chk_programs_age_range,
    DROP CONSTRAINT IF EXISTS chk_programs_weight_range,
    DROP CONSTRAINT IF EXISTS chk_programs_height_range,
    DROP COLUMN IF EXISTS max_age,
    DROP COLUMN IF EXISTS min_age,
    DROP COLUMN IF EXISTS max_weight_kg,
    DROP COLUMN IF EXISTS min_weight_kg,
    DROP COLUMN IF EXISTS max_height_cm,
    DROP COLUMN IF EXISTS min_height_cm;
