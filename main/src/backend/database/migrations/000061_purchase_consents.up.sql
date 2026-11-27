-- Evidence of the buyer's consents at checkout (owner request, 2026-11-23:
-- EU and Portuguese consumer rules). Before a paid purchase starts, the buyer
-- accepts the terms and asks for immediate access to the digital content,
-- acknowledging the loss of the 14-day right of withdrawal (Directive
-- 2011/83/EU art. 16(m); Decreto-Lei 24/2014 art. 17). The server records when.
ALTER TABLE purchases ADD COLUMN terms_accepted_at TIMESTAMPTZ NULL;
ALTER TABLE purchases ADD COLUMN immediate_access_requested_at TIMESTAMPTZ NULL;
