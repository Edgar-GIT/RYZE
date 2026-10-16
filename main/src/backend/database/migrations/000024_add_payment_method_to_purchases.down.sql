ALTER TABLE purchases
    DROP CONSTRAINT chk_purchases_payment_method,
    DROP COLUMN payment_method;