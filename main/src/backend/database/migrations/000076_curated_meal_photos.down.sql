DROP VIEW IF EXISTS v_curated_nutrition_plan_meal;
CREATE VIEW v_curated_nutrition_plan_meal AS
SELECT
    m.id,
    m.plan_id,
    m.position,
    m.label,
    m.kind,
    m.notes,
    m.created_at
FROM curated_nutrition_plan_meals m;
ALTER TABLE curated_nutrition_plan_meals
    DROP COLUMN IF EXISTS alternatives_image_key,
    DROP COLUMN IF EXISTS alternatives,
    DROP COLUMN IF EXISTS image_key;
