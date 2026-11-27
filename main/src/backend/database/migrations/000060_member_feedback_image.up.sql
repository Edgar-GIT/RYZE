-- Feedback with one image, and one feedback per member (owner request,
-- 2026-11-23). A member keeps at most one active submission — pending or
-- approved — instead of one pending at a time; a rejected one lets them try
-- again. Older extra active submissions of a member are soft-deleted (newest
-- kept) before the new rule applies. The optional image is stored in the
-- uploads "feedback" folder and served only through the application (public
-- only once approved).
ALTER TABLE member_feedback ADD COLUMN image_key VARCHAR(80) NULL;
ALTER TABLE member_feedback ADD CONSTRAINT chk_member_feedback_image_key
    CHECK (image_key IS NULL OR image_key ~ '^feedback/[0-9a-f-]{36}\.jpg$');

UPDATE member_feedback f
SET deleted_at = CURRENT_TIMESTAMP
WHERE f.deleted_at IS NULL
  AND f.status IN ('pending', 'approved')
  AND EXISTS (
      SELECT 1 FROM member_feedback newer
      WHERE newer.user_id = f.user_id
        AND newer.deleted_at IS NULL
        AND newer.status IN ('pending', 'approved')
        AND (newer.created_at, newer.id) > (f.created_at, f.id)
  );

ALTER TABLE member_feedback DROP CONSTRAINT uq_member_feedback_pending_user_id;
ALTER TABLE member_feedback DROP COLUMN pending_user_id;
ALTER TABLE member_feedback ADD COLUMN active_user_id VARCHAR(36) GENERATED ALWAYS AS (
    CASE WHEN deleted_at IS NULL AND status IN ('pending', 'approved') THEN user_id END
) STORED;
ALTER TABLE member_feedback ADD CONSTRAINT uq_member_feedback_active_user_id UNIQUE (active_user_id);
