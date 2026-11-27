-- Archived programmes (owner request, 2026-11-27): the owner (a trainer or the
-- platform) takes a published programme off the market for a while and can
-- put it back later. An archived programme stays published for the people who
-- already have it (purchases, Free activations, nutrition), but it is not
-- listed, matched, bought or activated again. At most 15 archived programmes
-- per owner (enforced by the API).
ALTER TABLE programs ADD COLUMN archived_at TIMESTAMPTZ NULL;
CREATE INDEX idx_programs_archived_owner ON programs (trainer_id) WHERE archived_at IS NOT NULL AND deleted_at IS NULL;
