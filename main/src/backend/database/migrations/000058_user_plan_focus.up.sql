-- The plan a member chose to focus on (owner request, 2026-11-22): with
-- several active plans, the home shows the plan of the member's latest action —
-- adding a plan, training one, or choosing one here. One row per member; the
-- choice is kept on the account so every device shows the same plan. Access is
-- re-checked whenever it is read or written; a plan that is no longer
-- accessible is simply ignored.
CREATE TABLE user_plan_focus (
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    chosen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_plan_focus_pkey PRIMARY KEY (user_id),
    CONSTRAINT fk_user_plan_focus_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_user_plan_focus_program FOREIGN KEY (program_id) REFERENCES programs (id)
);
