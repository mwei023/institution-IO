-- Distinguishes a real payment (money received, defaults to an unapplied
-- account credit until allocated) from a pure record-only entry with zero
-- financial effect. Every existing row defaults to false (financial) —
-- including payments entered via the old "record only" dropdown option,
-- since that option actually meant "unallocated", not "non-financial".
-- No historical data is being reclassified as a no-op.
ALTER TABLE fee_payments ADD COLUMN is_record_only BOOLEAN NOT NULL DEFAULT false;

-- Mirrors fee_payments' existing reversal columns (is_reversed,
-- reversed_at, reversed_by, reversal_reason) so an allocation can be
-- deallocated without ever deleting the row — same audit-trail pattern,
-- applied one level down. A deallocated payment can later be reallocated
-- as a brand new allocation row, preserving full history of the original.
ALTER TABLE payment_allocations ADD COLUMN is_reversed BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE payment_allocations ADD COLUMN reversed_at TIMESTAMPTZ;
ALTER TABLE payment_allocations ADD COLUMN reversed_by UUID REFERENCES users(id);
ALTER TABLE payment_allocations ADD COLUMN reversal_reason TEXT;