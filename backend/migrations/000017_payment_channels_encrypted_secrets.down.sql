-- Drop the recovered table if this migration is rolled back on a fresh
-- database. On manually-migrated databases where the table already held
-- live rows, down is a no-op — dropping it would destroy payment
-- channel configuration. Leave the data; only fresh DBs unwind fully.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'fee_payments' AND column_name = 'receipt_number'
    ) THEN
        -- Fresh database created entirely by this migration chain:
        -- safe to remove the recovered table.
        DROP TABLE IF EXISTS payment_channels;
    ELSE
        RAISE NOTICE 'payment_channels holds pre-existing rows; down migration skipped';
    END IF;
END $$;
