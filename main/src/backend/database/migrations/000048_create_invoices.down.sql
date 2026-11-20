DROP TRIGGER IF EXISTS trg_invoices_immutable ON invoices;
DROP FUNCTION IF EXISTS ryze_invoices_immutable();
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS invoice_counters;
ALTER TABLE purchases DROP CONSTRAINT IF EXISTS chk_purchases_buyer_tax_id;
ALTER TABLE purchases DROP COLUMN IF EXISTS buyer_tax_id;
