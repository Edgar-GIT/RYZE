-- Public profiles (owner request, 2026-11-21): people follow each other. One
-- row per follower and followee; unfollowing deletes the row (a follow is a
-- preference, not an audit fact). Counts and lists only include active,
-- non-Test-Mode accounts, so a deleted account simply stops counting.
CREATE TABLE user_follows (
    follower_id VARCHAR(36) NOT NULL,
    followee_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_follows_pkey PRIMARY KEY (follower_id, followee_id),
    CONSTRAINT fk_user_follows_follower FOREIGN KEY (follower_id) REFERENCES users (id),
    CONSTRAINT fk_user_follows_followee FOREIGN KEY (followee_id) REFERENCES users (id),
    CONSTRAINT chk_user_follows_not_self CHECK (follower_id <> followee_id)
);

CREATE INDEX idx_user_follows_followee ON user_follows (followee_id, created_at DESC);
