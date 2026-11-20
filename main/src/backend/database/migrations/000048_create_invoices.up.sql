-- Automatic purchase documents (owner request, 2026-11-20): every completed
-- purchase, real or Test Mode, gets exactly one document, shown to the client
-- and emailed without any client action. Until certified invoicing software is
-- connected the document is a purchase receipt marked as having no tax value
-- (owner decision, 2026-11-20); VAT follows the small-business exemption
-- (article 53 of the Portuguese VAT Code). Free activations are not purchases
-- and never get one.

-- The buyer's optional Portuguese taxpayer number (NIF), given at checkout and
-- validated by the server; NULL means "final consumer".
ALTER TABLE purchases ADD COLUMN buyer_tax_id VARCHAR(9) NULL;
ALTER TABLE purchases
    ADD CONSTRAINT chk_purchases_buyer_tax_id CHECK (buyer_tax_id IS NULL OR buyer_tax_id ~ '^[0-9]{9}$');

-- Gapless numbering per series and year: the counter row is incremented in the
-- same transaction as the document insert, so a rolled-back issue gives its
-- number back and concurrent issues queue on the row lock.
CREATE TABLE invoice_counters (
    series VARCHAR(8) NOT NULL,
    year SMALLINT NOT NULL,
    last_number INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT invoice_counters_pkey PRIMARY KEY (series, year),
    CONSTRAINT chk_invoice_counters_series CHECK (series IN ('RYZE', 'TEST')),
    CONSTRAINT chk_invoice_counters_last_number CHECK (last_number >= 0)
);

-- One document per completed purchase: a snapshot of the sale at issue time
-- (seller, buyer, item, amounts, VAT treatment). It is never updated or
-- deleted (see the trigger below).
CREATE TABLE invoices (
    id VARCHAR(36) NOT NULL,
    number VARCHAR(32) NOT NULL,
    series VARCHAR(8) NOT NULL,
    year SMALLINT NOT NULL,
    sequence INTEGER NOT NULL,
    purchase_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    program_id VARCHAR(36) NOT NULL,
    test BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(16) NOT NULL DEFAULT 'issued',
    issued_at TIMESTAMPTZ NOT NULL,
    purchased_at TIMESTAMPTZ NOT NULL,
    seller_name VARCHAR(160) NOT NULL,
    seller_tax_id VARCHAR(32),
    seller_address VARCHAR(255),
    buyer_name VARCHAR(255) NOT NULL,
    buyer_tax_id VARCHAR(9),
    description VARCHAR(255) NOT NULL,
    product_type VARCHAR(32) NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    unit_price_minor_units BIGINT NOT NULL,
    discount_minor_units BIGINT NOT NULL DEFAULT 0,
    tax_rate_bps INTEGER NOT NULL DEFAULT 0,
    tax_minor_units BIGINT NOT NULL DEFAULT 0,
    tax_exemption_code VARCHAR(8),
    total_minor_units BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    payment_method VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT invoices_pkey PRIMARY KEY (id),
    CONSTRAINT uq_invoices_number UNIQUE (number),
    CONSTRAINT uq_invoices_series_sequence UNIQUE (series, year, sequence),
    CONSTRAINT uq_invoices_purchase UNIQUE (purchase_id),
    CONSTRAINT fk_invoices_purchase FOREIGN KEY (purchase_id) REFERENCES purchases (id),
    CONSTRAINT fk_invoices_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_invoices_program FOREIGN KEY (program_id) REFERENCES programs (id),
    CONSTRAINT chk_invoices_series CHECK (series IN ('RYZE', 'TEST')),
    CONSTRAINT chk_invoices_test_series CHECK ((series = 'TEST') = test),
    CONSTRAINT chk_invoices_status CHECK (status IN ('issued')),
    CONSTRAINT chk_invoices_sequence CHECK (sequence > 0),
    CONSTRAINT chk_invoices_buyer_tax_id CHECK (buyer_tax_id IS NULL OR buyer_tax_id ~ '^[0-9]{9}$'),
    CONSTRAINT chk_invoices_quantity CHECK (quantity > 0),
    CONSTRAINT chk_invoices_amounts CHECK (
        unit_price_minor_units >= 0 AND discount_minor_units >= 0 AND tax_minor_units >= 0 AND total_minor_units >= 0
        AND tax_rate_bps >= 0
        AND total_minor_units = unit_price_minor_units * quantity - discount_minor_units + tax_minor_units
    )
);

CREATE INDEX idx_invoices_user ON invoices (user_id, issued_at DESC);

-- An issued document is immutable: corrections are future credit notes, never
-- edits, and documents are kept (no deletion).
CREATE FUNCTION ryze_invoices_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'invoices are immutable once issued' USING ERRCODE = 'restrict_violation';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_invoices_immutable BEFORE UPDATE OR DELETE ON invoices
    FOR EACH ROW EXECUTE FUNCTION ryze_invoices_immutable();
