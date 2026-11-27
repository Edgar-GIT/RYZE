-- Purchases keep the price they were sold at; only the discount link goes.
DROP INDEX IF EXISTS idx_purchases_discount_code;
ALTER TABLE purchases
    DROP CONSTRAINT IF EXISTS chk_purchases_discount,
    DROP CONSTRAINT IF EXISTS fk_purchases_discount_code,
    DROP COLUMN IF EXISTS discount_minor_units,
    DROP COLUMN IF EXISTS original_price_minor_units,
    DROP COLUMN IF EXISTS discount_code_id;
DROP TABLE IF EXISTS discount_codes;
