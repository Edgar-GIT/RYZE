-- Persists the generated Premium Level 1 nutrition plan as a normalised,
-- versioned domain instead of a single opaque JSON blob on the assignment.
--
-- The food catalog itself is server-owned code that is versioned with the
-- generation engine, so it is deliberately not a table: there is no mutable
-- catalog row a delivered plan could drift against. Each meal item therefore
-- stores an immutable snapshot of the food it resolved to (name, category,
-- unit and macros) plus the catalog code it came from, which keeps a delivered
-- plan byte-stable for ever while remaining fully auditable.
--
-- Versioning is explicit. An assignment owns at most one ACTIVE plan, enforced
-- by the unique index on the active_plan generated column. Regenerating an
-- assignment inserts the next version and supersedes the previous one, so a
-- delivered plan is never silently rewritten and history is retained.
CREATE TABLE IF NOT EXISTS nutrition_plans (
    id CHAR(36) PRIMARY KEY,
    -- Deliberately VARCHAR rather than CHAR, unlike the other UUID columns.
    -- MariaDB refuses to reference a CHAR column inside a generated column
    -- expression (CHAR blank-padding is not deterministic for that purpose), and
    -- this column has to be referenced by active_plan below in order to get a
    -- database-enforced "one active plan per assignment" guarantee. A VARCHAR(36)
    -- foreign key against the CHAR(36) parent is accepted by the engine.
    assignment_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    questionnaire_id CHAR(36) NOT NULL,
    questionnaire_version INT NOT NULL,
    version INT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    engine_version INT NOT NULL,
    fingerprint VARCHAR(64) NOT NULL,
    dietary_pattern VARCHAR(32) NOT NULL,
    maintenance_calories INT NOT NULL,
    target_calories INT NOT NULL,
    daily_protein_grams INT NOT NULL,
    daily_carbohydrate_grams INT NOT NULL,
    daily_fat_grams INT NOT NULL,
    daily_fiber_grams INT NOT NULL,
    protein_percent INT NOT NULL,
    carbohydrate_percent INT NOT NULL,
    fat_percent INT NOT NULL,
    meals_per_day INT NOT NULL,
    snacks_per_day INT NOT NULL,
    hydration_litres DECIMAL(4,1) NOT NULL,
    hydration_note VARCHAR(255) NOT NULL,
    summary VARCHAR(255) NOT NULL,
    prep_guidance VARCHAR(255) NOT NULL,
    cautions JSON NOT NULL,
    generated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    active_plan VARCHAR(36) GENERATED ALWAYS AS (
        IF(deleted_at IS NULL AND status = 'active', assignment_id, NULL)
    ) STORED,
    UNIQUE KEY uq_nutrition_plans_active (active_plan),
    UNIQUE KEY uq_nutrition_plans_version (assignment_id, version),
    CONSTRAINT fk_nutrition_plans_assignment FOREIGN KEY (assignment_id) REFERENCES nutrition_assignments(id),
    CONSTRAINT fk_nutrition_plans_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_nutrition_plans_program FOREIGN KEY (program_id) REFERENCES programs(id),
    CONSTRAINT fk_nutrition_plans_questionnaire FOREIGN KEY (questionnaire_id) REFERENCES nutrition_questionnaires(id),
    CONSTRAINT chk_nutrition_plans_status CHECK (status IN ('active', 'superseded')),
    CONSTRAINT chk_nutrition_plans_version CHECK (version > 0),
    CONSTRAINT chk_nutrition_plans_questionnaire_version CHECK (questionnaire_version > 0),
    CONSTRAINT chk_nutrition_plans_energy CHECK (
        maintenance_calories > 0 AND target_calories > 0
    ),
    CONSTRAINT chk_nutrition_plans_macros CHECK (
        daily_protein_grams >= 0 AND daily_carbohydrate_grams >= 0
        AND daily_fat_grams >= 0 AND daily_fiber_grams >= 0
    ),
    CONSTRAINT chk_nutrition_plans_structure CHECK (
        meals_per_day > 0 AND snacks_per_day >= 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_nutrition_plans_assignment_id ON nutrition_plans (assignment_id);
CREATE INDEX idx_nutrition_plans_user_id ON nutrition_plans (user_id);
CREATE INDEX idx_nutrition_plans_program_id ON nutrition_plans (program_id);
CREATE INDEX idx_nutrition_plans_questionnaire_id ON nutrition_plans (questionnaire_id);
CREATE INDEX idx_nutrition_plans_status ON nutrition_plans (status);

-- One row per eating occasion. Position is the stable presentation order and is
-- unique per plan, so a meal can never be duplicated or reordered inconsistently.
CREATE TABLE IF NOT EXISTS nutrition_plan_meals (
    id CHAR(36) PRIMARY KEY,
    plan_id CHAR(36) NOT NULL,
    position INT NOT NULL,
    label VARCHAR(48) NOT NULL,
    kind VARCHAR(16) NOT NULL,
    percent_of_daily INT NOT NULL,
    calories INT NOT NULL,
    protein_grams INT NOT NULL,
    carbohydrate_grams INT NOT NULL,
    fat_grams INT NOT NULL,
    notes VARCHAR(255) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_nutrition_plan_meals_plan FOREIGN KEY (plan_id) REFERENCES nutrition_plans(id) ON DELETE CASCADE,
    UNIQUE KEY uq_nutrition_plan_meals_position (plan_id, position),
    CONSTRAINT chk_nutrition_plan_meals_kind CHECK (kind IN ('meal', 'snack')),
    CONSTRAINT chk_nutrition_plan_meals_position CHECK (position > 0),
    CONSTRAINT chk_nutrition_plan_meals_share CHECK (
        percent_of_daily > 0 AND percent_of_daily <= 100
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_nutrition_plan_meals_plan_id ON nutrition_plan_meals (plan_id);

-- One row per food in a meal. The food columns are an immutable snapshot taken
-- at generation time: catalog_code records provenance, and the remaining columns
-- record exactly what the client was shown. Nothing here is re-read from a
-- mutable catalog, so an old plan version never changes after delivery.
CREATE TABLE IF NOT EXISTS nutrition_plan_meal_items (
    id CHAR(36) PRIMARY KEY,
    plan_id CHAR(36) NOT NULL,
    meal_id CHAR(36) NOT NULL,
    position INT NOT NULL,
    catalog_code VARCHAR(64) NOT NULL,
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
    CONSTRAINT fk_nutrition_plan_meal_items_plan FOREIGN KEY (plan_id) REFERENCES nutrition_plans(id) ON DELETE CASCADE,
    CONSTRAINT fk_nutrition_plan_meal_items_meal FOREIGN KEY (meal_id) REFERENCES nutrition_plan_meals(id) ON DELETE CASCADE,
    UNIQUE KEY uq_nutrition_plan_meal_items_position (meal_id, position),
    CONSTRAINT chk_nutrition_plan_meal_items_position CHECK (position > 0),
    CONSTRAINT chk_nutrition_plan_meal_items_quantity CHECK (quantity > 0),
    CONSTRAINT chk_nutrition_plan_meal_items_macros CHECK (
        calories >= 0 AND protein_grams >= 0 AND carbohydrate_grams >= 0
        AND fat_grams >= 0 AND fiber_grams >= 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_nutrition_plan_meal_items_plan_id ON nutrition_plan_meal_items (plan_id);
CREATE INDEX idx_nutrition_plan_meal_items_meal_id ON nutrition_plan_meal_items (meal_id);
CREATE INDEX idx_nutrition_plan_meal_items_catalog_code ON nutrition_plan_meal_items (catalog_code);

-- Audit trail of what a generation run excluded and why. This is what makes the
-- dietary guarantee inspectable instead of implicit: every strict exclusion and
-- every swapped food is recorded against the plan version that applied it.
-- reason_code is a controlled token; token is the normalised vocabulary entry
-- (for example an allergen token), never free questionnaire text.
CREATE TABLE IF NOT EXISTS nutrition_plan_exclusions (
    id CHAR(36) PRIMARY KEY,
    plan_id CHAR(36) NOT NULL,
    reason_code VARCHAR(32) NOT NULL,
    token VARCHAR(120) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_nutrition_plan_exclusions_plan FOREIGN KEY (plan_id) REFERENCES nutrition_plans(id) ON DELETE CASCADE,
    UNIQUE KEY uq_nutrition_plan_exclusions (plan_id, reason_code, token),
    CONSTRAINT chk_nutrition_plan_exclusions_reason CHECK (
        reason_code IN (
            'allergy', 'intolerance', 'diet', 'excluded_food',
            'disliked_food', 'insufficient_alternatives'
        )
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_nutrition_plan_exclusions_plan_id ON nutrition_plan_exclusions (plan_id);

-- Read models. Every nutrition read goes through these views, matching the
-- existing convention: table names are never exposed to a query and no view
-- carries internal diagnostics. failure_reason stays confined to the assignment
-- table for administrators.
CREATE OR REPLACE VIEW v_nutrition_plan AS
SELECT
    p.id,
    p.assignment_id,
    p.user_id,
    p.program_id,
    p.questionnaire_id,
    p.questionnaire_version,
    p.version,
    p.status,
    p.engine_version,
    p.fingerprint,
    p.dietary_pattern,
    p.maintenance_calories,
    p.target_calories,
    p.daily_protein_grams,
    p.daily_carbohydrate_grams,
    p.daily_fat_grams,
    p.daily_fiber_grams,
    p.protein_percent,
    p.carbohydrate_percent,
    p.fat_percent,
    p.meals_per_day,
    p.snacks_per_day,
    p.hydration_litres,
    p.hydration_note,
    p.summary,
    p.prep_guidance,
    p.cautions,
    p.generated_at,
    p.created_at,
    p.updated_at
FROM nutrition_plans p
WHERE p.deleted_at IS NULL;

CREATE OR REPLACE VIEW v_nutrition_plan_meal AS
SELECT
    m.id,
    m.plan_id,
    m.position,
    m.label,
    m.kind,
    m.percent_of_daily,
    m.calories,
    m.protein_grams,
    m.carbohydrate_grams,
    m.fat_grams,
    m.notes,
    m.created_at
FROM nutrition_plan_meals m;

CREATE OR REPLACE VIEW v_nutrition_plan_meal_item AS
SELECT
    i.id,
    i.plan_id,
    i.meal_id,
    i.position,
    i.catalog_code,
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
FROM nutrition_plan_meal_items i;

CREATE OR REPLACE VIEW v_nutrition_plan_exclusion AS
SELECT
    e.id,
    e.plan_id,
    e.reason_code,
    e.token,
    e.created_at
FROM nutrition_plan_exclusions e;
