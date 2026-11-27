-- Official RYZE plans (owner request, 2026-11-23): a free plan marked by
-- ADMIN_2 as the platform's own carries the fixed cover key below, whose
-- pictures ship with the API. Uploaded covers keep their pattern.
ALTER TABLE programs DROP CONSTRAINT chk_programs_cover_image;
ALTER TABLE programs
    ADD CONSTRAINT chk_programs_cover_image CHECK (
        cover_image IS NULL
        OR cover_image = 'programs/official/ryze_base_plan.png'
        OR cover_image ~ '^programs/(generic|l1|l2)/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(jpg|png|webp)$'
    );
