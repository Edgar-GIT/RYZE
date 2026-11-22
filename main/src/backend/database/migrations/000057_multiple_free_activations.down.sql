-- Back to one active Free plan per user: keep each user's newest activation
-- and soft-delete the others before restoring the per-user rule.
UPDATE free_program_activations f
SET deleted_at = CURRENT_TIMESTAMP
WHERE f.deleted_at IS NULL
  AND EXISTS (
      SELECT 1 FROM free_program_activations newer
      WHERE newer.user_id = f.user_id
        AND newer.deleted_at IS NULL
        AND (newer.activated_at, newer.id) > (f.activated_at, f.id)
  );
ALTER TABLE free_program_activations DROP CONSTRAINT uq_free_program_activations_active_program;
ALTER TABLE free_program_activations DROP COLUMN active_free_activation;
ALTER TABLE free_program_activations ADD COLUMN active_free_activation VARCHAR(36) GENERATED ALWAYS AS (
    CASE WHEN deleted_at IS NULL THEN user_id END
) STORED;
ALTER TABLE free_program_activations
    ADD CONSTRAINT uq_free_program_activations_active_user UNIQUE (active_free_activation);
