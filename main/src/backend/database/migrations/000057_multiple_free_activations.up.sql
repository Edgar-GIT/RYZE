-- Several Free plans can be active at once (owner decision, 2026-11-22, as in
-- Lyfta): the library holds any number of Free plans. The one-active rule now
-- applies per (user, program) instead of per user; Premium Level 1 and Level 2
-- stay mutually exclusive (checked at checkout).
ALTER TABLE free_program_activations DROP CONSTRAINT uq_free_program_activations_active_user;
ALTER TABLE free_program_activations DROP COLUMN active_free_activation;
ALTER TABLE free_program_activations ADD COLUMN active_free_activation VARCHAR(80) GENERATED ALWAYS AS (
    CASE WHEN deleted_at IS NULL THEN user_id || ':' || program_id END
) STORED;
ALTER TABLE free_program_activations
    ADD CONSTRAINT uq_free_program_activations_active_program UNIQUE (active_free_activation);
