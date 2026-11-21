-- Premium Level 2 allocation by offer (owner decision, 2026-11-21, until AI
-- exists): after the intake, the request is offered to an available trainer
-- (one who turned on accepts_level2_requests, with the fewest open requests).
-- The trainer accepts (the request is allocated to them) or declines; with
-- no answer in 24 hours the offer expires. Declined or expired, the request
-- is offered to the next trainer; with nobody left, ADMIN_2 allocates it by
-- hand as before (which withdraws an open offer).
ALTER TABLE trainers ADD COLUMN accepts_level2_requests BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE premium_level_2_offers (
    id VARCHAR(36) NOT NULL,
    request_id VARCHAR(36) NOT NULL,
    trainer_id VARCHAR(36) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'offered',
    offered_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    responded_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- One open offer per request.
    open_request_id VARCHAR(36) GENERATED ALWAYS AS (CASE WHEN status = 'offered' THEN request_id END) STORED,
    CONSTRAINT premium_level_2_offers_pkey PRIMARY KEY (id),
    CONSTRAINT fk_premium_level_2_offers_request FOREIGN KEY (request_id) REFERENCES premium_level_2_requests (id),
    CONSTRAINT fk_premium_level_2_offers_trainer FOREIGN KEY (trainer_id) REFERENCES trainers (id),
    CONSTRAINT chk_premium_level_2_offers_status CHECK (status IN ('offered', 'accepted', 'declined', 'expired', 'withdrawn')),
    CONSTRAINT chk_premium_level_2_offers_expiry CHECK (expires_at > offered_at),
    CONSTRAINT uq_premium_level_2_offers_open UNIQUE (open_request_id)
);

CREATE INDEX idx_premium_level_2_offers_trainer ON premium_level_2_offers (trainer_id, status);
CREATE INDEX idx_premium_level_2_offers_request ON premium_level_2_offers (request_id);
CREATE INDEX idx_premium_level_2_offers_expiry ON premium_level_2_offers (status, expires_at);

CREATE TRIGGER trg_premium_level_2_offers_touch_updated_at BEFORE UPDATE ON premium_level_2_offers
    FOR EACH ROW EXECUTE FUNCTION ryze_touch_updated_at();
