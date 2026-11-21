-- The person's public "about" (owner request, 2026-11-21): a biography, the
-- date of birth (only the age is shown), the year they started training
-- (shown as years of training), their weight, and social network handles
-- (only the handle is stored; the server builds the link, so a profile can
-- never carry an arbitrary URL). Every field is optional and shown on the
-- public profile only when set.
CREATE TABLE user_profile_details (
    user_id VARCHAR(36) NOT NULL,
    bio TEXT NULL,
    birth_date DATE NULL,
    training_since SMALLINT NULL,
    weight_kg NUMERIC(5, 1) NULL,
    social_links JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_profile_details_pkey PRIMARY KEY (user_id),
    CONSTRAINT fk_user_profile_details_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chk_user_profile_details_bio CHECK (bio IS NULL OR char_length(bio) <= 500),
    CONSTRAINT chk_user_profile_details_training_since CHECK (training_since IS NULL OR training_since BETWEEN 1950 AND 2100),
    CONSTRAINT chk_user_profile_details_weight CHECK (weight_kg IS NULL OR (weight_kg >= 20 AND weight_kg <= 400)),
    CONSTRAINT chk_user_profile_details_social_links CHECK (jsonb_typeof(social_links) = 'object')
);

CREATE TRIGGER trg_user_profile_details_touch_updated_at BEFORE UPDATE ON user_profile_details
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
