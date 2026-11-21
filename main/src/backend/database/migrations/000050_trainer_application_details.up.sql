-- A complete trainer application (owner request, 2026-11-20): the applicant
-- fills in who they are and their background, writes about themselves and
-- attaches their evidence in one submission. Each stored document keeps the
-- result of the automatic screening (signals for the mandatory human review);
-- files the screening rejects are never stored.

ALTER TABLE trainer_applications ADD COLUMN full_name VARCHAR(120);
ALTER TABLE trainer_applications ADD COLUMN birth_date DATE;
ALTER TABLE trainer_applications ADD COLUMN city VARCHAR(80);
ALTER TABLE trainer_applications ADD COLUMN profession VARCHAR(120);
ALTER TABLE trainer_applications ADD COLUMN years_experience SMALLINT;
ALTER TABLE trainer_applications ADD COLUMN work_history JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE trainer_applications ADD COLUMN education JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE trainer_applications ADD COLUMN about TEXT;
ALTER TABLE trainer_applications
    ADD CONSTRAINT chk_trainer_applications_years_experience CHECK (years_experience IS NULL OR years_experience BETWEEN 0 AND 60);
ALTER TABLE trainer_applications
    ADD CONSTRAINT chk_trainer_applications_about CHECK (about IS NULL OR length(about) <= 4000);

ALTER TABLE trainer_application_documents ADD COLUMN screening_status VARCHAR(16) NOT NULL DEFAULT 'not_screened';
ALTER TABLE trainer_application_documents ADD COLUMN screening_signals JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE trainer_application_documents
    ADD CONSTRAINT chk_trainer_application_documents_screening CHECK (screening_status IN ('not_screened', 'passed', 'flagged'));

-- The same file submitted by another account is a strong sign of a fake
-- identity: the screening looks documents up by their hash.
CREATE INDEX idx_trainer_application_documents_sha256 ON trainer_application_documents (sha256);
