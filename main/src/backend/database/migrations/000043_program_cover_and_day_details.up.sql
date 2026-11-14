-- Program cover images and training-day details (owner request, 2026-11-14).
--
-- programs.cover_image is the storage key of the plan's cover picture inside
-- the uploads directory (programs/<family folder>/<uuid>.<ext>). The API only
-- ever writes keys it generated itself; the CHECK keeps a corrupted row from
-- naming any other path.
--
-- program_workouts gains the training-day name, a short description and the
-- muscle group the day focuses on, so plans read like an app ("Day 1 - Push")
-- and statistics can later be grouped by focus. Existing rows keep an empty
-- name/description and no focus. Purely additive.
ALTER TABLE programs
    ADD COLUMN cover_image VARCHAR(120),
    ADD CONSTRAINT chk_programs_cover_image CHECK (
        cover_image IS NULL
        OR cover_image ~ '^programs/(generic|l1|l2)/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(jpg|png|webp)$'
    );

ALTER TABLE program_workouts
    ADD COLUMN name VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN description VARCHAR(1000) NOT NULL DEFAULT '',
    ADD COLUMN focus VARCHAR(20),
    ADD CONSTRAINT chk_program_workouts_focus CHECK (
        focus IS NULL OR focus IN (
            'full_body', 'upper_body', 'lower_body', 'push', 'pull', 'legs',
            'chest', 'back', 'shoulders', 'arms', 'core', 'glutes', 'cardio', 'mobility'
        )
    );
