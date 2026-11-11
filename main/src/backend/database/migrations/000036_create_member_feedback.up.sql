-- Member feedback: a star rating and a short message from a member who holds
-- (or held) at least one plan. Every submission starts pending and is only
-- shown publicly after an administrator approves it. reviewed_by stores the
-- configured administrator identity (ADMIN_1/ADMIN_2), not a database user.
-- pending_user_id enforces at most one pending submission per member.
CREATE TABLE member_feedback (
    id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    rating TINYINT UNSIGNED NOT NULL,
    message TEXT NOT NULL,
    display_name VARCHAR(120) NOT NULL,
    status VARCHAR(16) NOT NULL,
    reviewed_by VARCHAR(40) NULL DEFAULT NULL,
    reviewed_at DATETIME(6) NULL DEFAULT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL DEFAULT NULL,
    pending_user_id VARCHAR(36) GENERATED ALWAYS AS (IF(deleted_at IS NULL AND status = 'pending', user_id, NULL)) STORED,
    PRIMARY KEY (id),
    UNIQUE KEY uq_member_feedback_pending_user_id (pending_user_id),
    KEY idx_member_feedback_status_created_at (status, created_at),
    KEY idx_member_feedback_user_id (user_id),
    CONSTRAINT fk_member_feedback_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_member_feedback_rating CHECK (rating BETWEEN 1 AND 5),
    CONSTRAINT chk_member_feedback_status CHECK (status IN ('pending', 'approved', 'rejected'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
