ALTER TABLE premium_level_1_matches DROP CONSTRAINT IF EXISTS chk_premium_level_1_matches_sex;
ALTER TABLE premium_level_1_matches DROP COLUMN IF EXISTS sex;

ALTER TABLE users DROP COLUMN IF EXISTS plan_audience_filter;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_sex;
ALTER TABLE users DROP COLUMN IF EXISTS sex;

ALTER TABLE programs DROP CONSTRAINT IF EXISTS chk_programs_audience;
ALTER TABLE programs DROP COLUMN IF EXISTS audience;
