-- Meal photo and food alternatives in curated meal plans (v2.133.0): each meal
-- may have a picture, and a line of alternatives with an optional picture.
-- Keys point to public uploads under programs/meals/.
ALTER TABLE curated_nutrition_plan_meals
    ADD COLUMN image_key VARCHAR(255) NULL,
    ADD COLUMN alternatives VARCHAR(500) NULL,
    ADD COLUMN alternatives_image_key VARCHAR(255) NULL;

CREATE OR REPLACE VIEW v_curated_nutrition_plan_meal AS
SELECT
    m.id,
    m.plan_id,
    m.position,
    m.label,
    m.kind,
    m.notes,
    m.created_at,
    m.image_key,
    m.alternatives,
    m.alternatives_image_key
FROM curated_nutrition_plan_meals m;
