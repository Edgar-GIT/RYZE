-- Reverts the normalised, versioned nutrition plan tables introduced by this
-- migration. Child rows go first: the foreign keys cascade from nutrition_plans,
-- but dropping the parent first would fail while children still reference it.
--
-- nutrition_assignments.configuration is intentionally left in place. It is the
-- legacy snapshot column written before plans were normalised, and removing it
-- would destroy data that may still back a delivered plan.
DROP VIEW IF EXISTS v_nutrition_plan_meal_item;
DROP VIEW IF EXISTS v_nutrition_plan_meal;
DROP VIEW IF EXISTS v_nutrition_plan_exclusion;
DROP VIEW IF EXISTS v_nutrition_plan;

DROP TABLE IF EXISTS nutrition_plan_exclusions;
DROP TABLE IF EXISTS nutrition_plan_meal_items;
DROP TABLE IF EXISTS nutrition_plan_meals;
DROP TABLE IF EXISTS nutrition_plans;
