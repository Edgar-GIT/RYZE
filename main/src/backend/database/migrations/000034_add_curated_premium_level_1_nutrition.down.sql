DROP VIEW IF EXISTS v_curated_nutrition_plan_meal_item;
DROP VIEW IF EXISTS v_curated_nutrition_plan_meal;
DROP VIEW IF EXISTS v_curated_nutrition_plan;

DROP TABLE IF EXISTS curated_nutrition_plan_meal_items;
DROP TABLE IF EXISTS curated_nutrition_plan_meals;
DROP TABLE IF EXISTS curated_nutrition_plans;

ALTER TABLE programs
    DROP CONSTRAINT chk_programs_content_version,
    DROP CONSTRAINT chk_programs_product_metadata,
    DROP CONSTRAINT chk_programs_nutrition_mode,
    DROP CONSTRAINT chk_programs_objective,
    DROP INDEX idx_programs_nutrition_mode,
    DROP INDEX idx_programs_objective,
    DROP COLUMN content_version,
    DROP COLUMN nutrition_mode,
    DROP COLUMN objective;
