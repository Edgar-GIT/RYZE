ALTER TABLE member_feedback DROP CONSTRAINT IF EXISTS uq_member_feedback_active_user_id;
ALTER TABLE member_feedback DROP COLUMN IF EXISTS active_user_id;
ALTER TABLE member_feedback ADD COLUMN pending_user_id VARCHAR(36) GENERATED ALWAYS AS (
    CASE WHEN deleted_at IS NULL AND status = 'pending' THEN user_id END
) STORED;
ALTER TABLE member_feedback ADD CONSTRAINT uq_member_feedback_pending_user_id UNIQUE (pending_user_id);
ALTER TABLE member_feedback DROP CONSTRAINT IF EXISTS chk_member_feedback_image_key;
ALTER TABLE member_feedback DROP COLUMN IF EXISTS image_key;
