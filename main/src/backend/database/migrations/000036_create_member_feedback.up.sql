-- Member feedback: a star rating and a short message from a member who holds
-- (or held) at least one plan. Every submission starts pending and is only
-- shown publicly after an administrator approves it. reviewed_by stores the
-- configured administrator identity (ADMIN_1/ADMIN_2), not a database user.
-- pending_user_id enforces at most one pending submission per member.
CREATE TABLE member_feedback (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    rating SMALLINT NOT NULL,
    message TEXT NOT NULL,
    display_name VARCHAR(120) NOT NULL,
    status VARCHAR(16) NOT NULL,
    reviewed_by VARCHAR(40),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    pending_user_id VARCHAR(36) GENERATED ALWAYS AS (
        CASE WHEN deleted_at IS NULL AND status = 'pending' THEN user_id END
    ) STORED,
    CONSTRAINT member_feedback_pkey PRIMARY KEY (id),
    CONSTRAINT uq_member_feedback_pending_user_id UNIQUE (pending_user_id),
    CONSTRAINT fk_member_feedback_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_member_feedback_rating CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT chk_member_feedback_status CHECK (status IN ('pending', 'approved', 'rejected'))
);

CREATE INDEX idx_member_feedback_status_created_at ON member_feedback (status, created_at);
CREATE INDEX idx_member_feedback_user_id ON member_feedback (user_id);

CREATE TRIGGER trg_member_feedback_touch_updated_at BEFORE UPDATE ON member_feedback
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
