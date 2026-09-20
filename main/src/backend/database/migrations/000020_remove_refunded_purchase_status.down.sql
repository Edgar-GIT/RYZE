ALTER TABLE purchases DROP CONSTRAINT chk_purchases_status;
ALTER TABLE purchases ADD CONSTRAINT chk_purchases_status CHECK (status IN ('pending', 'completed', 'failed', 'refunded'));