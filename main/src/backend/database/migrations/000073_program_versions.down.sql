DROP INDEX IF EXISTS idx_programs_version_root;
DROP INDEX IF EXISTS idx_programs_previous_version;
ALTER TABLE programs
    DROP CONSTRAINT IF EXISTS fk_programs_version_root,
    DROP CONSTRAINT IF EXISTS fk_programs_previous_version,
    DROP COLUMN IF EXISTS superseded_at,
    DROP COLUMN IF EXISTS version_root_id,
    DROP COLUMN IF EXISTS previous_version_id;
