-- The converted programmes are not turned back (their buyers and any meal plan
-- added since would be inconsistent); only the constraint is restored for the
-- rows that satisfy it.
ALTER TABLE programs
    DROP CONSTRAINT chk_programs_product_metadata,
    ADD CONSTRAINT chk_programs_product_metadata CHECK (
        (product_type = 'generic' AND objective IS NULL AND nutrition_mode IS NULL)
        OR
        (product_type = 'premium_level_1' AND (
            (nutrition_mode = 'questionnaire_generated' AND objective IS NULL)
            OR
            (nutrition_mode = 'curated' AND objective IS NOT NULL)
        ))
        OR
        (product_type = 'premium_level_2' AND objective IS NULL AND nutrition_mode IS NULL)
    ) NOT VALID;
