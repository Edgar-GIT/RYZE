-- Programme versions (v2.133.0, owner decision 2026-11-27): a buyer keeps the
-- version they bought. Editing a sold package creates a new version (a copy);
-- publishing it supersedes the old one, which leaves the market but stays
-- readable for the people who bought it.
ALTER TABLE programs
    ADD COLUMN previous_version_id VARCHAR(36) NULL,
    ADD COLUMN version_root_id VARCHAR(36) NULL,
    ADD COLUMN superseded_at TIMESTAMPTZ NULL,
    ADD CONSTRAINT fk_programs_previous_version FOREIGN KEY (previous_version_id) REFERENCES programs (id),
    ADD CONSTRAINT fk_programs_version_root FOREIGN KEY (version_root_id) REFERENCES programs (id);

CREATE INDEX idx_programs_previous_version ON programs (previous_version_id) WHERE previous_version_id IS NOT NULL;
CREATE INDEX idx_programs_version_root ON programs (version_root_id) WHERE version_root_id IS NOT NULL;
