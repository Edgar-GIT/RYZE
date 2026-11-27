-- Official plans lose their fixed cover (they become plans without a cover).
UPDATE programs SET cover_image = NULL WHERE cover_image = 'programs/official/ryze_base_plan.png';
ALTER TABLE programs DROP CONSTRAINT chk_programs_cover_image;
ALTER TABLE programs
    ADD CONSTRAINT chk_programs_cover_image CHECK (
        cover_image IS NULL
        OR cover_image ~ '^programs/(generic|l1|l2)/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(jpg|png|webp)$'
    );
