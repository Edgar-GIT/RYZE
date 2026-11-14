DROP TABLE IF EXISTS premium_level_2_intake_drafts;
DROP TABLE IF EXISTS premium_level_1_matches;

ALTER TABLE programs DROP CONSTRAINT IF EXISTS chk_programs_objective;
ALTER TABLE programs
    ADD CONSTRAINT chk_programs_objective CHECK (
        objective IS NULL OR objective IN ('gain_muscle', 'lose_fat', 'strength_conditioning')
    );
