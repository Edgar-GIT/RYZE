-- Curated Premium Level 1 packages.
--
-- `objective` classifies a package into the product's closed goal vocabulary.
-- `nutrition_mode` persists which nutrition delivery model the package uses:
-- `curated` ships a predefined plan, while `questionnaire_generated` preserves
-- the legacy intake-first flow for packages sold before this model existed.
-- Generic programs carry NULL in both columns because nutrition is not part of
-- their product.
--
-- Existing Premium Level 1 rows are backfilled as legacy. No questionnaire,
-- generated plan, purchase or entitlement is rewritten.
ALTER TABLE programs
    ADD COLUMN objective VARCHAR(32),
    ADD COLUMN nutrition_mode VARCHAR(24),
    ADD COLUMN content_version INTEGER NOT NULL DEFAULT 1;

UPDATE programs
SET nutrition_mode = 'questionnaire_generated'
WHERE product_type = 'premium_level_1';

CREATE INDEX idx_programs_objective ON programs (objective);
CREATE INDEX idx_programs_nutrition_mode ON programs (nutrition_mode);

ALTER TABLE programs
    ADD CONSTRAINT chk_programs_objective CHECK (
        objective IS NULL OR objective IN ('gain_muscle', 'lose_fat', 'strength_conditioning')
    ),
    ADD CONSTRAINT chk_programs_nutrition_mode CHECK (
        nutrition_mode IS NULL OR nutrition_mode IN ('curated', 'questionnaire_generated')
    ),
    ADD CONSTRAINT chk_programs_product_metadata CHECK (
        (product_type = 'generic' AND objective IS NULL AND nutrition_mode IS NULL)
        OR
        (product_type = 'premium_level_1' AND (
            (nutrition_mode = 'questionnaire_generated' AND objective IS NULL)
            OR
            (nutrition_mode = 'curated' AND objective IS NOT NULL)
        ))
    ),
    ADD CONSTRAINT chk_programs_content_version CHECK (content_version > 0);

-- A curated plan belongs to one package and is delivered unchanged to every
-- buyer of that package. The header carries the plan's editorial identity and
-- the daily hydration guidance; meal energy and macros are derived from the
-- items so those totals can never disagree with the foods shown.
CREATE TABLE IF NOT EXISTS curated_nutrition_plans (
    id VARCHAR(36) PRIMARY KEY,
    program_id VARCHAR(36) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    dietary_pattern VARCHAR(32) NOT NULL DEFAULT 'balanced',
    summary VARCHAR(255) NOT NULL,
    prep_guidance VARCHAR(255) NOT NULL,
    hydration_litres NUMERIC(4,1) NOT NULL,
    hydration_note VARCHAR(255) NOT NULL,
    cautions JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uq_curated_nutrition_plans_program UNIQUE (program_id),
    CONSTRAINT fk_curated_nutrition_plans_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT chk_curated_nutrition_plans_version CHECK (version > 0),
    CONSTRAINT chk_curated_nutrition_plans_hydration CHECK (
        hydration_litres > 0 AND hydration_litres <= 20
    )
);

CREATE TRIGGER trg_curated_nutrition_plans_touch_updated_at BEFORE UPDATE ON curated_nutrition_plans
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();

CREATE TABLE IF NOT EXISTS curated_nutrition_plan_meals (
    id VARCHAR(36) PRIMARY KEY,
    plan_id VARCHAR(36) NOT NULL,
    position INTEGER NOT NULL,
    label VARCHAR(48) NOT NULL,
    kind VARCHAR(16) NOT NULL DEFAULT 'meal',
    notes VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_curated_nutrition_plan_meals_plan FOREIGN KEY (plan_id) REFERENCES curated_nutrition_plans (id) ON DELETE CASCADE,
    CONSTRAINT uq_curated_nutrition_plan_meals_position UNIQUE (plan_id, position),
    CONSTRAINT chk_curated_nutrition_plan_meals_kind CHECK (kind IN ('meal', 'snack')),
    CONSTRAINT chk_curated_nutrition_plan_meals_position CHECK (position > 0)
);

CREATE INDEX idx_curated_nutrition_plan_meals_plan_id ON curated_nutrition_plan_meals (plan_id);

CREATE TABLE IF NOT EXISTS curated_nutrition_plan_meal_items (
    id VARCHAR(36) PRIMARY KEY,
    plan_id VARCHAR(36) NOT NULL,
    meal_id VARCHAR(36) NOT NULL,
    position INTEGER NOT NULL,
    food_name VARCHAR(120) NOT NULL,
    category VARCHAR(32) NOT NULL,
    quantity NUMERIC(7,2) NOT NULL,
    unit VARCHAR(24) NOT NULL,
    calories INTEGER NOT NULL,
    protein_grams INTEGER NOT NULL,
    carbohydrate_grams INTEGER NOT NULL,
    fat_grams INTEGER NOT NULL,
    fiber_grams INTEGER NOT NULL,
    substitution_note VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_curated_nutrition_plan_meal_items_plan FOREIGN KEY (plan_id) REFERENCES curated_nutrition_plans (id) ON DELETE CASCADE,
    CONSTRAINT fk_curated_nutrition_plan_meal_items_meal FOREIGN KEY (meal_id) REFERENCES curated_nutrition_plan_meals (id) ON DELETE CASCADE,
    CONSTRAINT uq_curated_nutrition_plan_meal_items_position UNIQUE (meal_id, position),
    CONSTRAINT chk_curated_nutrition_plan_meal_items_position CHECK (position > 0),
    CONSTRAINT chk_curated_nutrition_plan_meal_items_quantity CHECK (quantity > 0),
    CONSTRAINT chk_curated_nutrition_plan_meal_items_macros CHECK (
        calories >= 0 AND protein_grams >= 0 AND carbohydrate_grams >= 0
        AND fat_grams >= 0 AND fiber_grams >= 0
    )
);

CREATE INDEX idx_curated_nutrition_plan_meal_items_plan_id ON curated_nutrition_plan_meal_items (plan_id);
CREATE INDEX idx_curated_nutrition_plan_meal_items_meal_id ON curated_nutrition_plan_meal_items (meal_id);

-- Safe read surfaces follow the existing view convention so no query has to
-- reference internal table names or deletion markers.
CREATE OR REPLACE VIEW v_curated_nutrition_plan AS
SELECT
    p.id,
    p.program_id,
    p.version,
    p.dietary_pattern,
    p.summary,
    p.prep_guidance,
    p.hydration_litres,
    p.hydration_note,
    p.cautions,
    p.created_at,
    p.updated_at
FROM curated_nutrition_plans p
WHERE p.deleted_at IS NULL;

CREATE OR REPLACE VIEW v_curated_nutrition_plan_meal AS
SELECT
    m.id,
    m.plan_id,
    m.position,
    m.label,
    m.kind,
    m.notes,
    m.created_at
FROM curated_nutrition_plan_meals m;

CREATE OR REPLACE VIEW v_curated_nutrition_plan_meal_item AS
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
FROM curated_nutrition_plan_meal_items i;
