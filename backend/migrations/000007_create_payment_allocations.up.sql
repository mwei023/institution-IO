CREATE TABLE IF NOT EXISTS payment_allocations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id  UUID NOT NULL REFERENCES institutions(id),
    payment_id      UUID NOT NULL REFERENCES fee_payments(id),
    invoice_id      UUID NOT NULL REFERENCES fee_invoices(id),
    amount_applied  NUMERIC(12,2) NOT NULL CHECK (amount_applied > 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_allocations_invoice ON payment_allocations(institution_id, invoice_id);
CREATE INDEX IF NOT EXISTS idx_payment_allocations_payment ON payment_allocations(institution_id, payment_id);