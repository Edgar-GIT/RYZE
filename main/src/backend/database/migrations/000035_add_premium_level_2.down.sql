DROP VIEW IF EXISTS v_premium_level_2_nutrition_plan_meal_item;
DROP VIEW IF EXISTS v_premium_level_2_nutrition_plan_meal;
DROP VIEW IF EXISTS v_premium_level_2_nutrition_plan;
DROP VIEW IF EXISTS v_premium_level_2_intake;
DROP VIEW IF EXISTS v_premium_level_2_request;

DROP TABLE IF EXISTS premium_level_2_request_events;
DROP TABLE IF EXISTS premium_level_2_nutrition_plan_meal_items;
DROP TABLE IF EXISTS premium_level_2_nutrition_plan_meals;
DROP TABLE IF EXISTS premium_level_2_nutrition_plans;
DROP TABLE IF EXISTS premium_level_2_intakes;
DROP TABLE IF EXISTS premium_level_2_requests;

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
    );

ALTER TABLE programs
    DROP CONSTRAINT chk_programs_product_type,
    ADD CONSTRAINT chk_programs_product_type CHECK (
        product_type IN ('generic', 'premium_level_1')
    );
