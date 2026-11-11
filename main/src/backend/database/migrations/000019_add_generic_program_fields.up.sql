-- Generic Program Builder: platform-owned catalogue programs (trainer_id NULL)
-- now carry optional target-audience and structure metadata. level uses the
-- documented catalogue difficulty vocabulary and is validated by the service
-- layer; duration_weeks and frequency_per_week are positive when provided.
ALTER TABLE programs
    ADD COLUMN level VARCHAR(20),
    ADD COLUMN duration_weeks INTEGER,
    ADD COLUMN frequency_per_week INTEGER,
    ADD CONSTRAINT chk_programs_duration_weeks CHECK (duration_weeks IS NULL OR duration_weeks > 0),
    ADD CONSTRAINT chk_programs_frequency_per_week CHECK (frequency_per_week IS NULL OR frequency_per_week > 0);

CREATE INDEX idx_programs_level ON programs (level);
CREATE INDEX idx_programs_duration_weeks ON programs (duration_weeks);
CREATE INDEX idx_programs_frequency_per_week ON programs (frequency_per_week);
