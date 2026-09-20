-- RYZE purchases are final. A customer purchases a plan once and the purchase
-- is final: there is no refund workflow, so 'refunded' is no longer a valid
-- purchase status.
ALTER TABLE purchases DROP CONSTRAINT chk_purchases_status;
ALTER TABLE purchases ADD CONSTRAINT chk_purchases_status CHECK (status IN ('pending', 'completed', 'failed'));