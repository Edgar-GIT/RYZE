-- Program view counters for the public coach showcase.
--
-- One row per program that has been viewed at least once. view_count counts
-- opens of the public program page, at most once per viewer and program every
-- 30 minutes (the API deduplicates in memory before writing); the owning
-- trainer's own visits are not counted. The counter decides which trainer
-- programs the marketplace features ("Featured coach programs").
--
-- The counter lives outside programs on purpose: incrementing a column there
-- would fire trg_programs_touch_updated_at and make every view look like a
-- content edit. Nothing about the viewer is stored.
CREATE TABLE program_view_stats (
    program_id VARCHAR(36) NOT NULL,
    view_count BIGINT NOT NULL DEFAULT 0,
    last_viewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT program_view_stats_pkey PRIMARY KEY (program_id),
    CONSTRAINT fk_program_view_stats_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT chk_program_view_stats_count CHECK (view_count >= 0)
);

CREATE INDEX idx_program_view_stats_count ON program_view_stats (view_count DESC);

CREATE TRIGGER trg_program_view_stats_touch_updated_at BEFORE UPDATE ON program_view_stats
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
