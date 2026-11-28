-- Who a Premium Level 1 package is for (v2.133.0): optional height, weight and
-- age ranges matched against the Level 1 questionnaire. NULL = any.
ALTER TABLE programs
    ADD COLUMN min_height_cm SMALLINT NULL,
    ADD COLUMN max_height_cm SMALLINT NULL,
    ADD COLUMN min_weight_kg SMALLINT NULL,
    ADD COLUMN max_weight_kg SMALLINT NULL,
    ADD COLUMN min_age SMALLINT NULL,
    ADD COLUMN max_age SMALLINT NULL,
    ADD CONSTRAINT chk_programs_height_range CHECK (
        (min_height_cm IS NULL OR min_height_cm BETWEEN 120 AND 230)
        AND (max_height_cm IS NULL OR max_height_cm BETWEEN 120 AND 230)
        AND (min_height_cm IS NULL OR max_height_cm IS NULL OR min_height_cm <= max_height_cm)),
    ADD CONSTRAINT chk_programs_weight_range CHECK (
        (min_weight_kg IS NULL OR min_weight_kg BETWEEN 30 AND 300)
        AND (max_weight_kg IS NULL OR max_weight_kg BETWEEN 30 AND 300)
        AND (min_weight_kg IS NULL OR max_weight_kg IS NULL OR min_weight_kg <= max_weight_kg)),
    ADD CONSTRAINT chk_programs_age_range CHECK (
        (min_age IS NULL OR min_age BETWEEN 14 AND 100)
        AND (max_age IS NULL OR max_age BETWEEN 14 AND 100)
        AND (min_age IS NULL OR max_age IS NULL OR min_age <= max_age)),
    ADD CONSTRAINT chk_programs_criteria_level1 CHECK (
        product_type = 'premium_level_1'
        OR (min_height_cm IS NULL AND max_height_cm IS NULL AND min_weight_kg IS NULL
            AND max_weight_kg IS NULL AND min_age IS NULL AND max_age IS NULL));
