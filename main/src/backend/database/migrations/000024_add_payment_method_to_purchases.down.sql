ALTER TABLE purchases
    DROP CHECK chk_purchases_payment_method,
    DROP COLUMN payment_method;