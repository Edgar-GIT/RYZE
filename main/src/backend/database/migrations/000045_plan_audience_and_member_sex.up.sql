-- Plans for women or men, and the member's sex (owner request, 2026-11-14).
--
-- A training plan may say it is designed more for women or more for men; NULL
-- means it is for everyone. A Premium Level 2 offer is personalised, so it
-- never carries an audience.
--
-- The member's sex is asked the first time they open the training plans and
-- in the Premium Level 1 journey. plan_audience_filter records whether they
-- prefer to see the plans made for their sex first: NULL means "not asked yet".
-- Each Premium Level 1 match also records the sex it was made for.
ALTER TABLE programs ADD COLUMN audience VARCHAR(10) NULL;
ALTER TABLE programs
    ADD CONSTRAINT chk_programs_audience CHECK (
        audience IS NULL OR (audience IN ('female', 'male') AND product_type <> 'premium_level_2')
    );

ALTER TABLE users ADD COLUMN sex VARCHAR(10) NULL;
ALTER TABLE users ADD CONSTRAINT chk_users_sex CHECK (sex IS NULL OR sex IN ('female', 'male'));
ALTER TABLE users ADD COLUMN plan_audience_filter BOOLEAN NULL;

ALTER TABLE premium_level_1_matches ADD COLUMN sex VARCHAR(10) NULL;
ALTER TABLE premium_level_1_matches
    ADD CONSTRAINT chk_premium_level_1_matches_sex CHECK (sex IS NULL OR sex IN ('female', 'male'));
