-- Premium Level 2 is a purchased, trainer-delivered service. The commercial
-- offer is a platform-owned program with product_type 'premium_level_2'. It has
-- no training structure or nutrition content of its own: the buyer purchases
-- the service first, submits a detailed intake, and is later allocated a trainer
-- who authors an individual programme and nutrition plan.
ALTER TABLE programs
    DROP CONSTRAINT chk_programs_product_type,
    ADD CONSTRAINT chk_programs_product_type CHECK (
        product_type IN ('generic', 'premium_level_1', 'premium_level_2')
    );

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
    );

-- One active service request per completed purchase. The request is created by
-- the purchase fulfillment hook, never by the client. A request starts in
-- awaiting_intake and progresses through the server-owned lifecycle. The
-- training_program_id is the trainer-authored individual programme; it remains
-- draft and is never published, so it can only be reached through the delivered
-- entitlement.
CREATE TABLE IF NOT EXISTS premium_level_2_requests (
    id CHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    trainer_id VARCHAR(36) NULL,
    training_program_id VARCHAR(36) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'awaiting_intake',
    intake_version INT NOT NULL DEFAULT 0,
    delivered_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    active_request VARCHAR(73) GENERATED ALWAYS AS (
        IF(deleted_at IS NULL, CONCAT(user_id, ':', program_id), NULL)
    ) STORED,
    UNIQUE KEY uq_premium_level_2_requests_active (active_request),
    CONSTRAINT fk_premium_level_2_requests_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_premium_level_2_requests_program FOREIGN KEY (program_id) REFERENCES programs(id),
    CONSTRAINT fk_premium_level_2_requests_trainer FOREIGN KEY (trainer_id) REFERENCES trainers(id),
    CONSTRAINT fk_premium_level_2_requests_training_program FOREIGN KEY (training_program_id) REFERENCES programs(id),
    CONSTRAINT chk_premium_level_2_requests_status CHECK (
        status IN (
            'awaiting_intake',
            'awaiting_trainer_assignment',
            'in_progress',
            'ready_for_delivery',
            'delivered',
            'cancelled'
        )
    ),
    CONSTRAINT chk_premium_level_2_requests_intake_version CHECK (intake_version >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_premium_level_2_requests_user_id ON premium_level_2_requests (user_id);
CREATE INDEX idx_premium_level_2_requests_program_id ON premium_level_2_requests (program_id);
CREATE INDEX idx_premium_level_2_requests_trainer_id ON premium_level_2_requests (trainer_id);
CREATE INDEX idx_premium_level_2_requests_status ON premium_level_2_requests (status);
CREATE INDEX idx_premium_level_2_requests_training_program ON premium_level_2_requests (training_program_id);

-- Every submitted revision is retained. The intake contains sensitive health
-- and lifestyle information: it is never logged and is only projected to the
-- owner, the allocated trainer and an authorised administrator.
CREATE TABLE IF NOT EXISTS premium_level_2_intakes (
    id CHAR(36) PRIMARY KEY,
    request_id CHAR(36) NOT NULL,
    version INT NOT NULL,
    answers JSON NOT NULL,
    submitted_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    UNIQUE KEY uq_premium_level_2_intakes_version (request_id, version),
    CONSTRAINT fk_premium_level_2_intakes_request FOREIGN KEY (request_id) REFERENCES premium_level_2_requests(id),
    CONSTRAINT chk_premium_level_2_intakes_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_premium_level_2_intakes_request_id ON premium_level_2_intakes (request_id);

-- The trainer-authored individual nutrition plan. It reuses the validated
-- curated nutrition shape, but belongs to one request/client rather than to a
-- package. It is invisible until the request is delivered.
CREATE TABLE IF NOT EXISTS premium_level_2_nutrition_plans (
    id CHAR(36) PRIMARY KEY,
    request_id VARCHAR(36) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    dietary_pattern VARCHAR(32) NOT NULL DEFAULT 'balanced',
    summary VARCHAR(255) NOT NULL,
    prep_guidance VARCHAR(255) NOT NULL,
    hydration_litres DECIMAL(4,1) NOT NULL,
    hydration_note VARCHAR(255) NOT NULL,
    cautions JSON NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    active_plan VARCHAR(36) GENERATED ALWAYS AS (
        IF(deleted_at IS NULL, request_id, NULL)
    ) STORED,
    UNIQUE KEY uq_premium_level_2_nutrition_plans_active (active_plan),
    CONSTRAINT fk_premium_level_2_nutrition_plans_request FOREIGN KEY (request_id) REFERENCES premium_level_2_requests(id),
    CONSTRAINT chk_premium_level_2_nutrition_plans_status CHECK (status IN ('draft', 'delivered')),
    CONSTRAINT chk_premium_level_2_nutrition_plans_version CHECK (version > 0),
    CONSTRAINT chk_premium_level_2_nutrition_plans_hydration CHECK (
        hydration_litres > 0 AND hydration_litres <= 20
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_premium_level_2_nutrition_plans_request_id ON premium_level_2_nutrition_plans (request_id);

CREATE TABLE IF NOT EXISTS premium_level_2_nutrition_plan_meals (
    id CHAR(36) PRIMARY KEY,
    plan_id CHAR(36) NOT NULL,
    position INT NOT NULL,
    label VARCHAR(48) NOT NULL,
    kind VARCHAR(16) NOT NULL DEFAULT 'meal',
    notes VARCHAR(255) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_premium_level_2_plan_meals_plan FOREIGN KEY (plan_id) REFERENCES premium_level_2_nutrition_plans(id) ON DELETE CASCADE,
    UNIQUE KEY uq_premium_level_2_plan_meals_position (plan_id, position),
    CONSTRAINT chk_premium_level_2_plan_meals_kind CHECK (kind IN ('meal', 'snack')),
    CONSTRAINT chk_premium_level_2_plan_meals_position CHECK (position > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_premium_level_2_plan_meals_plan_id ON premium_level_2_nutrition_plan_meals (plan_id);

CREATE TABLE IF NOT EXISTS premium_level_2_nutrition_plan_meal_items (
    id CHAR(36) PRIMARY KEY,
    plan_id CHAR(36) NOT NULL,
    meal_id CHAR(36) NOT NULL,
    position INT NOT NULL,
    food_name VARCHAR(120) NOT NULL,
    category VARCHAR(32) NOT NULL,
    quantity DECIMAL(7,2) NOT NULL,
    unit VARCHAR(24) NOT NULL,
    calories INT NOT NULL,
    protein_grams INT NOT NULL,
    carbohydrate_grams INT NOT NULL,
    fat_grams INT NOT NULL,
    fiber_grams INT NOT NULL,
    substitution_note VARCHAR(255) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_premium_level_2_plan_items_plan FOREIGN KEY (plan_id) REFERENCES premium_level_2_nutrition_plans(id) ON DELETE CASCADE,
    CONSTRAINT fk_premium_level_2_plan_items_meal FOREIGN KEY (meal_id) REFERENCES premium_level_2_nutrition_plan_meals(id) ON DELETE CASCADE,
    UNIQUE KEY uq_premium_level_2_plan_items_position (meal_id, position),
    CONSTRAINT chk_premium_level_2_plan_items_position CHECK (position > 0),
    CONSTRAINT chk_premium_level_2_plan_items_quantity CHECK (quantity > 0),
    CONSTRAINT chk_premium_level_2_plan_items_macros CHECK (
        calories >= 0 AND protein_grams >= 0 AND carbohydrate_grams >= 0
        AND fat_grams >= 0 AND fiber_grams >= 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_premium_level_2_plan_items_plan_id ON premium_level_2_nutrition_plan_meal_items (plan_id);
CREATE INDEX idx_premium_level_2_plan_items_meal_id ON premium_level_2_nutrition_plan_meal_items (meal_id);

-- Append-only audit trail for allocation, reassignment and delivery. It stores
-- actor identities and status transitions, never intake answer content.
CREATE TABLE IF NOT EXISTS premium_level_2_request_events (
    id CHAR(36) PRIMARY KEY,
    request_id CHAR(36) NOT NULL,
    actor_kind VARCHAR(16) NOT NULL,
    actor_id VARCHAR(36) NOT NULL,
    event_type VARCHAR(32) NOT NULL,
    from_status VARCHAR(32) NULL,
    to_status VARCHAR(32) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_premium_level_2_events_request FOREIGN KEY (request_id) REFERENCES premium_level_2_requests(id),
    CONSTRAINT chk_premium_level_2_events_actor_kind CHECK (actor_kind IN ('client', 'trainer', 'admin', 'system'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_premium_level_2_events_request_id ON premium_level_2_request_events (request_id);

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
    np.status AS nutrition_plan_status
FROM premium_level_2_requests r
LEFT JOIN programs o ON o.id = r.program_id
LEFT JOIN users u ON u.id = r.user_id
LEFT JOIN trainers t ON t.id = r.trainer_id
LEFT JOIN users tu ON tu.id = t.user_id
LEFT JOIN programs tp ON tp.id = r.training_program_id
LEFT JOIN premium_level_2_nutrition_plans np ON np.request_id = r.id AND np.deleted_at IS NULL
WHERE r.deleted_at IS NULL;

CREATE OR REPLACE VIEW v_premium_level_2_intake AS
SELECT
    i.id,
    i.request_id,
    i.version,
    i.answers,
    i.submitted_at,
    i.created_at
FROM premium_level_2_intakes i
INNER JOIN premium_level_2_requests r ON r.id = i.request_id AND r.deleted_at IS NULL;

CREATE OR REPLACE VIEW v_premium_level_2_nutrition_plan AS
SELECT
    p.id,
    p.request_id,
    p.version,
    p.status,
    p.dietary_pattern,
    p.summary,
    p.prep_guidance,
    p.hydration_litres,
    p.hydration_note,
    p.cautions,
    p.created_at,
    p.updated_at
FROM premium_level_2_nutrition_plans p
WHERE p.deleted_at IS NULL;

CREATE OR REPLACE VIEW v_premium_level_2_nutrition_plan_meal AS
SELECT
    m.id,
    m.plan_id,
    m.position,
    m.label,
    m.kind,
    m.notes,
    m.created_at
FROM premium_level_2_nutrition_plan_meals m;

CREATE OR REPLACE VIEW v_premium_level_2_nutrition_plan_meal_item AS
SELECT
    i.id,
    i.plan_id,
    i.meal_id,
    i.position,
    i.food_name,
    i.category,
    i.quantity,
    i.unit,
    i.calories,
    i.protein_grams,
    i.carbohydrate_grams,
    i.fat_grams,
    i.fiber_grams,
    i.substitution_note,
    i.created_at
FROM premium_level_2_nutrition_plan_meal_items i;
