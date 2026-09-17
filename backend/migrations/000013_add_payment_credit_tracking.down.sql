-- Down for 000013: drop credit-tracking columns added in up.
ALTER TABLE payment_allocations DROP COLUMN IF EXISTS reversal_reason;
ALTER TABLE payment_allocations DROP COLUMN IF EXISTS reversed_by;
ALTER TABLE payment_allocations DROP COLUMN IF EXISTS reversed_at;
ALTER TABLE payment_allocations DROP COLUMN IF EXISTS is_reversed;
ALTER TABLE fee_payments DROP COLUMN IF EXISTS is_record_only;
