-- EXP adjustments (owner request, 2026-11-21): EXP is still derived from the
-- workout history, but an operator can add or remove EXP with a recorded
-- reason, for example to put a test account one EXP short of a level and see
-- the level-up animation. Only the `set-xp` command writes this table; the
-- member's total never goes below zero.
CREATE TABLE xp_adjustments (
    id VARCHAR(36) NOT NULL DEFAULT gen_random_uuid()::text,
    user_id VARCHAR(36) NOT NULL,
    xp INTEGER NOT NULL,
    reason VARCHAR(200) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT xp_adjustments_pkey PRIMARY KEY (id),
    CONSTRAINT fk_xp_adjustments_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_xp_adjustments_xp CHECK (xp BETWEEN -1000000 AND 1000000 AND xp <> 0),
    CONSTRAINT chk_xp_adjustments_reason CHECK (char_length(btrim(reason)) BETWEEN 1 AND 200)
);

CREATE INDEX idx_xp_adjustments_user ON xp_adjustments (user_id);
