-- Premium Level 2 offers go to every available trainer at once (owner
-- decision, 2026-11-24): a request can have many open offers, one per
-- trainer; the first trainer to accept takes it and the others are withdrawn.
ALTER TABLE premium_level_2_offers DROP CONSTRAINT uq_premium_level_2_offers_open;
ALTER TABLE premium_level_2_offers
    ADD CONSTRAINT uq_premium_level_2_offers_request_trainer UNIQUE (request_id, trainer_id);
