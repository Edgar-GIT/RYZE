-- Premium Level 2 rework (owner request, 2026-11-24).
--
-- 1. RYZE itself (ADMIN_2) can take a request instead of a trainer.
ALTER TABLE premium_level_2_requests
    ADD COLUMN handled_by_ryze BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN ryze_admin_id VARCHAR(36) NULL;
ALTER TABLE premium_level_2_requests
    ADD CONSTRAINT chk_premium_level_2_requests_one_handler
    CHECK (NOT (handled_by_ryze AND trainer_id IS NOT NULL));

-- 2. A picture per meal in the personal nutrition plan.
ALTER TABLE premium_level_2_nutrition_plan_meals
    ADD COLUMN image_key VARCHAR(80) NULL;
ALTER TABLE premium_level_2_nutrition_plan_meals
    ADD CONSTRAINT chk_premium_level_2_meal_image
    CHECK (image_key IS NULL OR image_key ~ '^programs/meals/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(jpg|png|webp)$');

-- 3. The service is one platform product: administrators no longer create
--    Premium Level 2 offers. Make sure it exists (19.49 EUR).
INSERT INTO programs (id, trainer_id, name, description, type, product_type, status, price_minor_units, currency, content_version, created_at, updated_at)
SELECT gen_random_uuid()::text, NULL, 'RYZE Personalizado',
       'Treino e nutrição feitos só para ti: respondes ao questionário e um treinador RYZE prepara o teu plano.',
       'premium', 'premium_level_2', 'published', 1949, 'EUR', 1, NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM programs
    WHERE product_type = 'premium_level_2' AND status = 'published' AND deleted_at IS NULL
);

-- Read models (columns appended at the end so the views keep their order).
CREATE OR REPLACE VIEW v_premium_level_2_request AS
SELECT
    r.id,
    r.user_id,
    r.program_id,
    r.trainer_id,
    r.training_program_id,
    r.status,
    r.intake_version,
    r.delivered_at,
    r.created_at,
    r.updated_at,
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
    np.status AS nutrition_plan_status,
    r.handled_by_ryze,
    r.ryze_admin_id
FROM premium_level_2_requests r
LEFT JOIN programs o ON o.id = r.program_id
LEFT JOIN users u ON u.id = r.user_id
LEFT JOIN trainers t ON t.id = r.trainer_id
LEFT JOIN users tu ON tu.id = t.user_id
LEFT JOIN programs tp ON tp.id = r.training_program_id
LEFT JOIN premium_level_2_nutrition_plans np ON np.request_id = r.id AND np.deleted_at IS NULL
WHERE r.deleted_at IS NULL;

CREATE OR REPLACE VIEW v_premium_level_2_nutrition_plan_meal AS
SELECT
    m.id,
    m.plan_id,
    m.position,
    m.label,
    m.kind,
    m.notes,
    m.created_at,
    m.image_key
FROM premium_level_2_nutrition_plan_meals m;
