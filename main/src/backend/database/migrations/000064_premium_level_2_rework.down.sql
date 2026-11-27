DROP VIEW v_premium_level_2_request;
CREATE VIEW v_premium_level_2_request AS
SELECT
    r.id, r.user_id, r.program_id, r.trainer_id, r.training_program_id, r.status,
    r.intake_version, r.delivered_at, r.created_at, r.updated_at,
    COALESCE(o.name, '') AS offer_name,
    COALESCE(o.description, '') AS offer_description,
    COALESCE(o.price_minor_units, 0) AS offer_price_minor_units,
    COALESCE(o.currency, 'EUR') AS offer_currency,
    COALESCE(u.first_name, '') AS client_first_name,
    COALESCE(u.last_name, '') AS client_last_name,
    COALESCE(u.email, '') AS client_email,
    t.user_id AS trainer_user_id,
    tu.first_name AS trainer_first_name,
    tu.last_name AS trainer_last_name,
    tp.name AS training_program_name,
    np.id AS nutrition_plan_id,
    np.version AS nutrition_plan_version,
    np.status AS nutrition_plan_status
FROM premium_level_2_requests r
LEFT JOIN programs o ON o.id = r.program_id
LEFT JOIN users u ON u.id = r.user_id
LEFT JOIN trainers t ON t.id = r.trainer_id
LEFT JOIN users tu ON tu.id = t.user_id
LEFT JOIN programs tp ON tp.id = r.training_program_id
LEFT JOIN premium_level_2_nutrition_plans np ON np.request_id = r.id AND np.deleted_at IS NULL
WHERE r.deleted_at IS NULL;

DROP VIEW v_premium_level_2_nutrition_plan_meal;
CREATE VIEW v_premium_level_2_nutrition_plan_meal AS
SELECT m.id, m.plan_id, m.position, m.label, m.kind, m.notes, m.created_at
FROM premium_level_2_nutrition_plan_meals m;

ALTER TABLE premium_level_2_nutrition_plan_meals DROP CONSTRAINT IF EXISTS chk_premium_level_2_meal_image;
ALTER TABLE premium_level_2_nutrition_plan_meals DROP COLUMN IF EXISTS image_key;
ALTER TABLE premium_level_2_requests DROP CONSTRAINT IF EXISTS chk_premium_level_2_requests_one_handler;
ALTER TABLE premium_level_2_requests DROP COLUMN IF EXISTS ryze_admin_id;
ALTER TABLE premium_level_2_requests DROP COLUMN IF EXISTS handled_by_ryze;
