-- Payment method selected by the buyer at payment initiation. It is validated
-- against the provider allowlist and recorded only after the server has
-- resolved a configured provider for it; from that point the method is
-- immutable, so the capture flow can always resolve the correct capture
-- provider server-side instead of trusting any client input. NULL marks
-- purchases created before this column existed (legacy pending purchases) and
-- Test Mode purchases, which never contact a payment provider.
ALTER TABLE purchases
    ADD COLUMN payment_method VARCHAR(20) NULL,
    ADD CONSTRAINT chk_purchases_payment_method CHECK (
        payment_method IN ('card', 'mbway', 'paypal')
    );