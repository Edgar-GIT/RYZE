-- Reverts the Premium Level 1 product family and its questionnaire-driven
-- nutrition assignment tables. Views are dropped first because they read the
-- tables this migration creates.
DROP VIEW IF EXISTS v_nutrition_assignment;
DROP VIEW IF EXISTS v_nutrition_questionnaire_summary;

DROP TABLE IF EXISTS nutrition_assignments;
DROP TABLE IF EXISTS nutrition_questionnaires;

ALTER TABLE programs
    DROP INDEX idx_programs_product_type,
    DROP CHECK chk_programs_product_type,
    DROP COLUMN product_type;
