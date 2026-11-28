-- Trainers' Premium plans made before v2.132.0 become Premium Level 1 packages
-- (owner decision, 2026-11-27). They have no objective and no meal plan yet,
-- so they become drafts: off the market until the trainer completes them and
-- publishes. People who bought them keep their access (entitlements do not
-- depend on the status).
--
-- A curated draft may still lack its objective; publishing requires it (the
-- service refuses a package without objective and complete meal plan).
ALTER TABLE programs
    DROP CONSTRAINT chk_programs_product_metadata,
    ADD CONSTRAINT chk_programs_product_metadata CHECK (
        (product_type = 'generic' AND objective IS NULL AND nutrition_mode IS NULL)
        OR
        (product_type = 'premium_level_1' AND (
            (nutrition_mode = 'questionnaire_generated' AND objective IS NULL)
            OR
            (nutrition_mode = 'curated' AND (objective IS NOT NULL OR status = 'draft'))
        ))
        OR
        (product_type = 'premium_level_2' AND objective IS NULL AND nutrition_mode IS NULL)
    );

UPDATE programs
SET product_type = 'premium_level_1',
    nutrition_mode = 'curated',
    status = 'draft',
    price_minor_units = 1499,
    currency = 'EUR',
    updated_at = now()
WHERE trainer_id IS NOT NULL
  AND type = 'premium'
  AND product_type = 'generic'
  AND deleted_at IS NULL;
